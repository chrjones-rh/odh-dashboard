package repositories

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
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

	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		modelName = r.resolveMainModel(client, ctx, namespace, item)
	}
	if modelName == "" {
		return nil, &ModelNotWiredError{Name: name}
	}

	token, err := client.BearerToken()
	if err != nil {
		return nil, fmt.Errorf("error resolving user token: %w", err)
	}

	httpClient := &http.Client{
		Timeout: guardrailTestTimeout,
		Transport: &http.Transport{
			// Trust the service serving CA that signs the guardrail service
			// certificate; falls back to the system pool.
			TLSClientConfig: r.guardrailTLSConfig(client, ctx, namespace, name),
		},
	}

	return runGuardrailTest(ctx, httpClient, endpoint, token, modelName, strings.TrimSpace(request.Message))
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
