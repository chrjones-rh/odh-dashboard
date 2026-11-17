package repositories

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opendatahub-io/ai-safety/bff/internal/constants"
	k8s "github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/models"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// guardrailTestTimeout bounds a single test request. Chat completions can take
// a while when the wired model is large or cold.
const guardrailTestTimeout = 120 * time.Second

// DeploymentNotReadyError reports a test attempt against a deployment whose
// operator status is not Ready or that exposes no endpoint yet.
type DeploymentNotReadyError struct {
	Name string
}

func (e *DeploymentNotReadyError) Error() string {
	return fmt.Sprintf("guardrail deployment %s is not ready", e.Name)
}

// ModelNotWiredError reports a test attempt against a deployment with neither
// a MAIN_MODEL_BASE_URL environment variable nor a main model entry in its
// configuration.
type ModelNotWiredError struct {
	Name string
}

func (e *ModelNotWiredError) Error() string {
	return fmt.Sprintf("guardrail deployment %s is not wired to a model", e.Name)
}

// ModelNameRequiredError reports a test attempt against a deployment whose
// model name could not be determined: it is not in the deployment's
// configuration, and the guardrail service did not report it either.
type ModelNameRequiredError struct {
	Name string
}

func (e *ModelNameRequiredError) Error() string {
	return fmt.Sprintf(
		"guardrail deployment %s is wired to a model but its model name could not be determined; provide the model name in the test request",
		e.Name,
	)
}

// TestDeployment runs a test message through a guardrail deployment: it calls
// the deployment's /v1/checks endpoint for a structured pass/block verdict
// and, for messages that pass, /v1/chat/completions for the model's answer.
// Both calls are made with the requesting user's token; the guardrail's
// kube-rbac-proxy authorizes it against the service resource.
func (r *GuardrailsRepository) TestDeployment(
	client k8s.KubernetesClientInterface,
	ctx context.Context,
	namespace, name string,
	request models.GuardrailTestRequest,
) (*models.GuardrailTestResult, error) {
	item, err := client.GetNemoGuardrails(ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("error fetching NemoGuardrails deployment %s: %w", name, err)
	}

	if !isDeploymentReady(item) {
		return nil, &DeploymentNotReadyError{Name: name}
	}
	endpoint, _, _ := unstructured.NestedString(item.Object, "status", "endpoint")
	if strings.TrimSpace(endpoint) == "" {
		return nil, &DeploymentNotReadyError{Name: name}
	}
	if err := validateGuardrailEndpoint(endpoint); err != nil {
		return nil, err
	}

	// Local development: reach the guardrail through its route instead of the
	// cluster-internal service URL. Route certificates are signed by the
	// ingress operator CA, so the service serving CA pool does not apply.
	tlsConfig := r.guardrailTLSConfig(client, ctx, namespace, name)
	if r.local != nil && r.local.RouteSuffix != "" {
		endpoint = r.routeEndpoint(endpoint)
		tlsConfig = &tls.Config{RootCAs: r.local.RouteCAs}
	}

	token, err := client.BearerToken()
	if err != nil {
		return nil, fmt.Errorf("error resolving user token: %w", err)
	}

	var dialMap map[string]string
	if r.local != nil {
		dialMap = r.local.DialMap
	}
	httpClient := &http.Client{
		Timeout:   guardrailTestTimeout,
		Transport: guardrailTransport(tlsConfig, dialMap),
	}

	// Model name resolution: an explicit request override wins, then the
	// deployment's configured main model (dashboard-created deployments),
	// then the guardrail service itself, which reports the model it is wired
	// to through its OpenAI-compatible /v1/models endpoint.
	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		modelName = r.resolveMainModel(client, ctx, namespace, item)
	}
	if modelName == "" {
		modelName = fetchGuardrailModelName(ctx, httpClient, endpoint, token)
	}
	if modelName == "" {
		if hasMainModelBaseURL(item) {
			return nil, &ModelNameRequiredError{Name: name}
		}
		return nil, &ModelNotWiredError{Name: name}
	}

	result, err := runGuardrailTest(ctx, httpClient, endpoint, token, modelName, strings.TrimSpace(request.Message))
	if err != nil {
		// The guardrail service itself failed (connection or error
		// response). The message is sanitized and describes a service the
		// requester is authorized to test, so it is safe to surface.
		return nil, &GuardrailUpstreamError{Detail: err.Error()}
	}
	return result, nil
}

// GuardrailUpstreamError reports a failure of the guardrail service itself
// during a test: the connection failed or the service answered with an error
// status. The detail is derived from the runGuardrailTest messages, which are
// truncated and contain no credentials.
type GuardrailUpstreamError struct {
	Detail string
}

func (e *GuardrailUpstreamError) Error() string {
	return e.Detail
}

// routeEndpoint rewrites a validated in-cluster service endpoint to the
// deployment's OpenShift route ("<service>-<namespace>.<routeSuffix>"), used
// for local development where service URLs are unreachable but routes are.
// Only service URLs are rewritten; anything else is left untouched.
func (r *GuardrailsRepository) routeEndpoint(endpoint string) string {
	if r.local == nil || r.local.RouteSuffix == "" {
		return endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	host := parsed.Hostname()
	trimmed := strings.TrimSuffix(host, ".svc.cluster.local")
	if trimmed == host {
		trimmed = strings.TrimSuffix(host, ".svc")
	}
	if trimmed == host {
		// Not a service URL; do not rewrite.
		return endpoint
	}
	service, namespace, found := strings.Cut(trimmed, ".")
	if !found || service == "" || namespace == "" {
		return endpoint
	}
	parsed.Scheme = "https"
	parsed.Host = fmt.Sprintf("%s-%s.%s", service, namespace, r.local.RouteSuffix)
	return parsed.String()
}

// guardrailTransport builds the HTTP transport for guardrail test calls. The
// dialMap is a local-development override: it redirects TCP connections to a
// guardrail service hostname (typically an `oc port-forward` listener) while
// leaving the request URL untouched, so TLS SNI and certificate validation
// against the service serving CA keep working. Keys are exact hostnames, or
// wildcard patterns like "*.svc.cluster.local" matching any subdomain of the
// suffix; exact entries take precedence over wildcards.
func guardrailTransport(tlsConfig *tls.Config, dialMap map[string]string) *http.Transport {
	transport := &http.Transport{
		// Trust the service serving CA that signs the guardrail service
		// certificate; falls back to the system pool.
		TLSClientConfig: tlsConfig,
	}
	if len(dialMap) == 0 {
		return transport
	}
	baseDialer := &net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if host, _, err := net.SplitHostPort(addr); err == nil {
			if override, ok := dialMap[host]; ok {
				addr = override
			} else {
				for pattern, override := range dialMap {
					if strings.HasPrefix(pattern, "*.") && strings.HasSuffix(host, pattern[1:]) {
						addr = override
						break
					}
				}
			}
		}
		return baseDialer.DialContext(ctx, network, addr)
	}
	return transport
}

// validateGuardrailEndpoint checks that a deployment endpoint is an
// in-cluster service URL. The endpoint comes from the CR status, but it is
// validated before the BFF connects to it.
func validateGuardrailEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("deployment endpoint %q is not a valid URL", endpoint)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("deployment endpoint scheme must be http or https, got %q", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	if !strings.HasSuffix(hostname, ".svc") && !strings.HasSuffix(hostname, ".svc.cluster.local") {
		return fmt.Errorf("deployment endpoint %q is not an in-cluster service URL", endpoint)
	}
	return nil
}

// guardrailTLSConfig builds the TLS configuration trusting the CA that signs
// the guardrail service certificate. The operator creates a per-deployment
// CA bundle ConfigMap (<name>-ca-bundle); the namespace-synced
// openshift-service-ca.crt ConfigMap is the fallback. When neither is
// readable, nil is returned and the system pool is used.
func (r *GuardrailsRepository) guardrailTLSConfig(client k8s.KubernetesClientInterface, ctx context.Context, namespace, name string) *tls.Config {
	configMapNames := []string{
		fmt.Sprintf("%s-ca-bundle", name),
		constants.ServiceCAConfigMapFallback,
	}
	for _, configMapName := range configMapNames {
		configMap, err := client.GetConfigMap(ctx, namespace, configMapName)
		if err != nil {
			continue
		}
		pemBytes, ok := configMap.Data[constants.ServiceCAConfigMapKey]
		if !ok {
			continue
		}
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM([]byte(pemBytes)) {
			return &tls.Config{RootCAs: pool}
		}
	}
	return nil
}

// resolveMainModel finds the main model name from the deployment's referenced
// configuration ConfigMaps. It returns an empty string when none is wired.
func (r *GuardrailsRepository) resolveMainModel(client k8s.KubernetesClientInterface, ctx context.Context, namespace string, item *unstructured.Unstructured) string {
	for _, configMapName := range referencedConfigMapNames(item) {
		configMap, err := client.GetConfigMap(ctx, namespace, configMapName)
		if err != nil {
			continue
		}
		if configYAML, ok := configMap.Data[constants.GuardrailsConfigYAMLKey]; ok {
			if name := mainModelNameFromConfig(configYAML); name != "" {
				return name
			}
		}
	}
	return ""
}

// guardrailModelsResponse is the subset of the OpenAI-compatible /v1/models
// response the model name resolution needs.
type guardrailModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// fetchGuardrailModelName asks the guardrail service which model it is wired
// to, through its OpenAI-compatible /v1/models endpoint. Deployments wired
// only through the MAIN_MODEL_BASE_URL environment variable carry no model
// name of their own; the guardrail still knows and reports it. Failures and
// empty responses yield an empty string.
func fetchGuardrailModelName(ctx context.Context, httpClient *http.Client, endpoint, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(endpoint, "/")+"/v1/models", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}
	var models guardrailModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		return ""
	}
	for _, model := range models.Data {
		if name := strings.TrimSpace(model.ID); name != "" {
			return name
		}
	}
	return ""
}

// guardrailChatResponse is the subset of the OpenAI-compatible chat completion
// response the test result surfaces.
type guardrailChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Guardrails struct {
		ConfigID string `json:"config_id"`
	} `json:"guardrails"`
}

// guardrailCheckResponse is the /v1/checks response: a structured verdict.
type guardrailCheckResponse struct {
	Status  string `json:"status"`
	Content string `json:"content"`
	Rail    string `json:"rail"`
}

// runGuardrailTest performs the check and chat calls against a guardrail
// endpoint. It is separate from TestDeployment so tests can exercise it
// against an httptest server.
func runGuardrailTest(
	ctx context.Context,
	httpClient *http.Client,
	endpoint, token, model, message string,
) (*models.GuardrailTestResult, error) {
	result := &models.GuardrailTestResult{}

	body, err := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": message}},
	})
	if err != nil {
		return nil, err
	}

	// 1. Structured verdict. Configurations that do not support check
	// requests (Colang 2.0) answer 422; the verdict is then unknown.
	checkBody, checkStatus, err := postGuardrailJSON(ctx, httpClient, endpoint+"/v1/checks", token, body)
	if err != nil {
		return nil, fmt.Errorf("guardrail check request failed: %w", err)
	}
	switch checkStatus {
	case http.StatusOK:
		var check guardrailCheckResponse
		if err := json.Unmarshal(checkBody, &check); err != nil {
			return nil, fmt.Errorf("guardrail check response is not valid JSON: %w", err)
		}
		result.Status = check.Status
		result.Content = check.Content
		result.Rail = check.Rail
	case http.StatusUnprocessableEntity:
		result.Status = "unknown"
	default:
		return nil, fmt.Errorf("guardrail check request failed with status %d: %s", checkStatus, truncateForError(checkBody))
	}

	// 2. The model's answer. Skipped when the rails already blocked the
	// message: the check response carries the block message and the model is
	// not called.
	if result.Status == "blocked" {
		return result, nil
	}

	chatBody, chatStatus, err := postGuardrailJSON(ctx, httpClient, endpoint+"/v1/chat/completions", token, body)
	if err != nil {
		return nil, fmt.Errorf("guardrail chat completion request failed: %w", err)
	}
	if chatStatus != http.StatusOK {
		return nil, fmt.Errorf("guardrail chat completion request failed with status %d: %s", chatStatus, truncateForError(chatBody))
	}
	var chat guardrailChatResponse
	if err := json.Unmarshal(chatBody, &chat); err != nil {
		return nil, fmt.Errorf("guardrail chat completion response is not valid JSON: %w", err)
	}
	if len(chat.Choices) > 0 {
		result.ModelResponse = chat.Choices[0].Message.Content
	}
	result.ConfigID = chat.Guardrails.ConfigID
	if result.Content == "" {
		result.Content = result.ModelResponse
	}
	return result, nil
}

// postGuardrailJSON posts the body to the guardrail endpoint with the user's
// token and returns the response status and body.
func postGuardrailJSON(ctx context.Context, httpClient *http.Client, endpoint, token string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

// truncateForError reduces an upstream error body to a message-safe snippet.
func truncateForError(body []byte) string {
	message := strings.TrimSpace(string(body))
	if len(message) > 200 {
		message = message[:200]
	}
	return message
}
