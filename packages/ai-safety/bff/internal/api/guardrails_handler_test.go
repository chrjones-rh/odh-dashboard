package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/opendatahub-io/ai-safety/bff/internal/config"
	"github.com/opendatahub-io/ai-safety/bff/internal/constants"
	"github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// guardrailsFakeClient implements the guardrails-relevant surface of
// KubernetesClientInterface, backed by in-memory data.
type guardrailsFakeClient struct {
	kubernetes.KubernetesClientInterface
	configMaps           []corev1.ConfigMap
	nemoGuardrails       []unstructured.Unstructured
	inferenceServices    []unstructured.Unstructured
	llmInferenceServices []unstructured.Unstructured
	allowList            bool
	allowCreate          bool
	allowDelete          bool
}

func (f *guardrailsFakeClient) ListConfigMapsByLabel(_ context.Context, namespace string, labelSelector string) ([]corev1.ConfigMap, error) {
	if labelSelector != "nemo-guardrails-config=true" {
		return nil, nil
	}
	var result []corev1.ConfigMap
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace {
			result = append(result, f.configMaps[i])
		}
	}
	return result, nil
}

func (f *guardrailsFakeClient) GetConfigMap(_ context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == name {
			return &f.configMaps[i], nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, name)
}

func (f *guardrailsFakeClient) ListNemoGuardrails(_ context.Context, namespace string) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured
	for _, item := range f.nemoGuardrails {
		if item.GetNamespace() == namespace {
			result = append(result, item)
		}
	}
	return result, nil
}

func (f *guardrailsFakeClient) GetNemoGuardrails(_ context.Context, namespace string, name string) (*unstructured.Unstructured, error) {
	for i := range f.nemoGuardrails {
		if f.nemoGuardrails[i].GetNamespace() == namespace && f.nemoGuardrails[i].GetName() == name {
			clone := f.nemoGuardrails[i]
			return &clone, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, name)
}

func (f *guardrailsFakeClient) CanListNemoGuardrails(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowList, nil
}

func (f *guardrailsFakeClient) ListInferenceServices(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.inferenceServices, nil
}

func (f *guardrailsFakeClient) ListLLMInferenceServices(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.llmInferenceServices, nil
}

func (f *guardrailsFakeClient) CanListInferenceServices(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowList, nil
}

func (f *guardrailsFakeClient) CanListLLMInferenceServices(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowList, nil
}

func (f *guardrailsFakeClient) BearerToken() (string, error) {
	return "test-token", nil
}

func (f *guardrailsFakeClient) UpdateConfigMap(_ context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == configMap.Name {
			updated := *configMap
			updated.Namespace = namespace
			f.configMaps[i] = updated
			return &updated, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, configMap.Name)
}

func (f *guardrailsFakeClient) CanCreateConfigMaps(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowCreate, nil
}

func (f *guardrailsFakeClient) CanUpdateConfigMaps(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowCreate, nil
}

func (f *guardrailsFakeClient) CanListConfigMaps(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowList, nil
}

func (f *guardrailsFakeClient) CanCreateGuardrailDeployment(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowCreate, nil
}

func (f *guardrailsFakeClient) CreateConfigMap(_ context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == configMap.Name {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "", Resource: "configmaps"}, configMap.Name)
		}
	}
	created := *configMap
	created.Namespace = namespace
	f.configMaps = append(f.configMaps, created)
	return &created, nil
}

func (f *guardrailsFakeClient) CreateNemoGuardrails(_ context.Context, namespace string, cr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	for i := range f.nemoGuardrails {
		if f.nemoGuardrails[i].GetNamespace() == namespace && f.nemoGuardrails[i].GetName() == cr.GetName() {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, cr.GetName())
		}
	}
	created := *cr
	created.SetNamespace(namespace)
	f.nemoGuardrails = append(f.nemoGuardrails, created)
	return &created, nil
}

// guardrailsFakeFactory returns the preset client for both client paths.
type guardrailsFakeFactory struct {
	client kubernetes.KubernetesClientInterface
}

func (f *guardrailsFakeFactory) GetClient(_ context.Context) (kubernetes.KubernetesClientInterface, error) {
	return f.client, nil
}

func (f *guardrailsFakeFactory) GetInternalClient() (kubernetes.KubernetesClientInterface, error) {
	return f.client, nil
}

func (f *guardrailsFakeFactory) ExtractRequestIdentity(_ http.Header) (*kubernetes.RequestIdentity, error) {
	return &kubernetes.RequestIdentity{UserID: "test-user", Groups: []string{}}, nil
}

func (f *guardrailsFakeFactory) ValidateRequestIdentity(_ *kubernetes.RequestIdentity) error {
	return nil
}

func guardrailsTestConfigMaps() []corev1.ConfigMap {
	return []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "redhat-ods-applications",
				Name:              "trustyai-service-operator-nemo-guardrails-default-pii",
				Labels:            map[string]string{"nemo-guardrails-config": "true"},
				CreationTimestamp: metav1.Now(),
			},
			Data: map[string]string{
				"config.yaml": "passthrough: true\nrails:\n  input:\n    flows:\n      - mask pii\n",
			},
		},
		{
			// A user-created template in the user's namespace.
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "test-ns",
				Name:              "my-custom-guardrail",
				Labels:            map[string]string{"nemo-guardrails-config": "true"},
				CreationTimestamp: metav1.Now(),
			},
			Data: map[string]string{
				"config.yaml": "passthrough: true\nrails:\n  input:\n    flows:\n      - self check input\n",
			},
		},
	}
}

func guardrailsTestDeploymentCR() unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "trustyai.opendatahub.io/v1alpha1",
			"kind":       "NemoGuardrails",
			"metadata": map[string]interface{}{
				"namespace":         "test-ns",
				"name":              "my-guardrails",
				"creationTimestamp": "2026-01-15T10:30:00Z",
			},
			"spec": map[string]interface{}{
				"nemoConfigs": []interface{}{
					map[string]interface{}{
						"name":       "main-config",
						"configMaps": []interface{}{"guardrail-placeholder"},
						"default":    true,
					},
				},
				"env": []interface{}{
					map[string]interface{}{"name": "OPENAI_API_KEY", "value": "secret"},
				},
			},
			"status": map[string]interface{}{
				"phase": "Ready",
				"conditions": []interface{}{
					map[string]interface{}{"type": "DeploymentReady", "status": "True"},
				},
			},
		},
	}
}

// setupGuardrailsRequest builds an App wired to the fake client and performs
// the request, returning the recorded response and the decoded JSON body.
func setupGuardrailsRequest(t *testing.T, method, url string, allowList bool) (*httptest.ResponseRecorder, map[string]interface{}) {
	return setupGuardrailsRequestWithBody(t, method, url, allowList, false, nil)
}

func setupGuardrailsRequestWithBody(
	t *testing.T,
	method, url string,
	allowList, allowCreate bool,
	requestBody map[string]interface{},
) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()

	client := &guardrailsFakeClient{
		configMaps:     guardrailsTestConfigMaps(),
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestDeploymentCR()},
		allowList:      allowList,
		allowCreate:    allowCreate,
		allowDelete:    allowList,
	}

	app := &App{
		config: config.EnvConfig{
			AllowedOrigins:              []string{"*"},
			AuthMethod:                  config.AuthMethodInternal,
			GuardrailsTemplateNamespace: "redhat-ods-applications",
		},
		logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
		kubernetesClientFactory: &guardrailsFakeFactory{client: client},
		repositories:            repositories.NewRepositories(),
	}

	var reqBody io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		require.NoError(t, err)
		reqBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)
	req.Header.Set(constants.KubeflowUserIDHeader, "test-user")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	identity := &kubernetes.RequestIdentity{UserID: "test-user", Groups: []string{}}
	ctx := context.WithValue(req.Context(), constants.RequestIdentityKey, identity)
	req = req.WithContext(ctx)

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)

	var body map[string]interface{}
	if recorder.Code == http.StatusOK || recorder.Code == http.StatusCreated {
		err = json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
	}
	return recorder, body
}

func TestGetGuardrailTemplatesHandler(t *testing.T) {
	recorder, body := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/templates", true)

	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].([]interface{})
	require.True(t, ok, "expected data array in response")
	require.Len(t, data, 1)

	template := data[0].(map[string]interface{})
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-pii", template["name"])
	assert.Equal(t, "redhat-ods-applications", template["namespace"])
}

func TestGetGuardrailTemplateHandler(t *testing.T) {
	recorder, body := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/templates/trustyai-service-operator-nemo-guardrails-default-pii", true)

	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "expected data object in response")
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-pii", data["name"])
	files, ok := data["files"].([]interface{})
	require.True(t, ok)
	require.Len(t, files, 1)
	assert.Equal(t, "config.yaml", files[0].(map[string]interface{})["key"])
}

func TestGetGuardrailTemplateHandlerNotFound(t *testing.T) {
	recorder, _ := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/templates/missing", true)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestGetGuardrailDeploymentsHandler(t *testing.T) {
	recorder, body := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/deployments?namespace=test-ns", true)

	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].([]interface{})
	require.True(t, ok, "expected data array in response")
	require.Len(t, data, 1)

	deployment := data[0].(map[string]interface{})
	assert.Equal(t, "my-guardrails", deployment["name"])
	assert.Equal(t, "Ready", deployment["phase"])
	assert.Equal(t, true, deployment["ready"])
}

func TestGetGuardrailDeploymentsHandlerForbidden(t *testing.T) {
	recorder, _ := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/deployments?namespace=test-ns", false)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestGetGuardrailDeploymentsHandlerMissingNamespace(t *testing.T) {
	recorder, _ := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/deployments", true)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetGuardrailDeploymentHandler(t *testing.T) {
	recorder, body := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/deployments/my-guardrails?namespace=test-ns", true)

	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "expected data object in response")
	assert.Equal(t, "my-guardrails", data["name"])

	envVarNames, ok := data["envVarNames"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"OPENAI_API_KEY"}, envVarNames)
}

func TestGetGuardrailDeploymentHandlerNotFound(t *testing.T) {
	recorder, _ := setupGuardrailsRequest(t, http.MethodGet, "/api/v1/guardrails/deployments/missing?namespace=test-ns", true)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// Compile-time guard: the handlers must match the httprouter handle signature.
var (
	_ httprouter.Handle = (*App)(nil).GetGuardrailTemplatesHandler
	_ httprouter.Handle = (*App)(nil).GetGuardrailDeploymentsHandler
)

func TestCreateGuardrailDeploymentHandler(t *testing.T) {
	body := map[string]interface{}{
		"name":         "new-guardrail",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
	}
	recorder, responseBody := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)

	require.Equal(t, http.StatusCreated, recorder.Code)
	data, ok := responseBody["data"].(map[string]interface{})
	require.True(t, ok, "expected data object in response")
	assert.Equal(t, "new-guardrail", data["name"])
	assert.Equal(t, "test-ns", data["namespace"])
}

func TestCreateGuardrailDeploymentHandlerForbidden(t *testing.T) {
	body := map[string]interface{}{
		"name":         "new-guardrail",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
	}
	recorder, _ := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, false, body)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreateGuardrailDeploymentHandlerInvalidName(t *testing.T) {
	body := map[string]interface{}{
		"name":         "Invalid Name",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
	}
	recorder, _ := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateGuardrailDeploymentHandlerMissingFields(t *testing.T) {
	body := map[string]interface{}{
		"name": "new-guardrail",
	}
	recorder, _ := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateGuardrailDeploymentHandlerMissingTemplate(t *testing.T) {
	body := map[string]interface{}{
		"name":         "new-guardrail",
		"namespace":    "test-ns",
		"templateName": "missing-template",
	}
	recorder, _ := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestCreateGuardrailDeploymentHandlerAlreadyExists(t *testing.T) {
	// my-guardrails already exists in test-ns via guardrailsTestDeploymentCR
	body := map[string]interface{}{
		"name":         "my-guardrails",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
	}
	recorder, _ := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)
	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func TestGetGuardrailTemplatesWithUserNamespace(t *testing.T) {
	// The fake factory returns the same client for both paths, so requesting a
	// namespace merges the platform catalog with the user namespace templates.
	recorder, body := setupGuardrailsRequest(
		t, http.MethodGet, "/api/v1/guardrails/templates?namespace=test-ns", true)

	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].([]interface{})
	require.True(t, ok, "expected data array in response")
	require.Len(t, data, 2)

	byName := map[string]map[string]interface{}{}
	for _, raw := range data {
		template := raw.(map[string]interface{})
		byName[template["name"].(string)] = template
	}
	assert.Equal(t, "platform", byName["trustyai-service-operator-nemo-guardrails-default-pii"]["source"])
	assert.Equal(t, "user", byName["my-custom-guardrail"]["source"])
}

func TestGetGuardrailTemplatesUserNamespaceForbidden(t *testing.T) {
	recorder, body := setupGuardrailsRequest(
		t, http.MethodGet, "/api/v1/guardrails/templates?namespace=test-ns", false)

	// The platform catalog remains available when the user cannot list
	// ConfigMaps in the requested namespace.
	require.Equal(t, http.StatusOK, recorder.Code)
	data, ok := body["data"].([]interface{})
	require.True(t, ok, "expected data array in response")
	require.Len(t, data, 1)
	template := data[0].(map[string]interface{})
	assert.Equal(t, "platform", template["source"])
}

func TestCreateGuardrailDeploymentFromUserTemplate(t *testing.T) {
	body := map[string]interface{}{
		"name":              "new-guardrail",
		"namespace":         "test-ns",
		"templateName":      "my-custom-guardrail",
		"templateNamespace": "test-ns",
	}
	recorder, responseBody := setupGuardrailsRequestWithBody(
		t, http.MethodPost, "/api/v1/guardrails/deployments", true, true, body)

	require.Equal(t, http.StatusCreated, recorder.Code)
	data, ok := responseBody["data"].(map[string]interface{})
	require.True(t, ok, "expected data object in response")
	assert.Equal(t, "new-guardrail", data["name"])
}

func (f *guardrailsFakeClient) DeleteNemoGuardrails(_ context.Context, namespace string, name string) error {
	for i, item := range f.nemoGuardrails {
		if item.GetNamespace() == namespace && item.GetName() == name {
			f.nemoGuardrails = append(f.nemoGuardrails[:i], f.nemoGuardrails[i+1:]...)
			return nil
		}
	}
	return apierrors.NewNotFound(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, name)
}

func (f *guardrailsFakeClient) DeleteConfigMap(_ context.Context, namespace string, name string) error {
	for i, cm := range f.configMaps {
		if cm.Namespace == namespace && cm.Name == name {
			f.configMaps = append(f.configMaps[:i], f.configMaps[i+1:]...)
			return nil
		}
	}
	return apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, name)
}

func (f *guardrailsFakeClient) CanDeleteNemoGuardrails(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowDelete, nil
}

func TestDeleteGuardrailDeploymentHandler(t *testing.T) {
	// allowDelete needs wiring into the setup helper; use a custom client.
	client := &guardrailsFakeClient{
		configMaps:     guardrailsTestConfigMaps(),
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestDeploymentCR()},
		allowList:      true,
		allowDelete:    true,
	}

	app := &App{
		config: config.EnvConfig{
			AllowedOrigins:              []string{"*"},
			AuthMethod:                  config.AuthMethodInternal,
			GuardrailsTemplateNamespace: "redhat-ods-applications",
		},
		logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
		kubernetesClientFactory: &guardrailsFakeFactory{client: client},
		repositories:            repositories.NewRepositories(),
	}

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/guardrails/deployments/my-guardrails?namespace=test-ns", http.NoBody)
	require.NoError(t, err)
	req.Header.Set(constants.KubeflowUserIDHeader, "test-user")

	identity := &kubernetes.RequestIdentity{UserID: "test-user", Groups: []string{}}
	ctx := context.WithValue(req.Context(), constants.RequestIdentityKey, identity)
	req = req.WithContext(ctx)

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Empty(t, client.nemoGuardrails)
}

func TestDeleteGuardrailDeploymentHandlerForbidden(t *testing.T) {
	client := &guardrailsFakeClient{
		configMaps:     guardrailsTestConfigMaps(),
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestDeploymentCR()},
		allowList:      true,
		allowDelete:    false,
	}

	app := &App{
		config: config.EnvConfig{
			AllowedOrigins:              []string{"*"},
			AuthMethod:                  config.AuthMethodInternal,
			GuardrailsTemplateNamespace: "redhat-ods-applications",
		},
		logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
		kubernetesClientFactory: &guardrailsFakeFactory{client: client},
		repositories:            repositories.NewRepositories(),
	}

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/guardrails/deployments/my-guardrails?namespace=test-ns", http.NoBody)
	require.NoError(t, err)
	req.Header.Set(constants.KubeflowUserIDHeader, "test-user")

	identity := &kubernetes.RequestIdentity{UserID: "test-user", Groups: []string{}}
	ctx := context.WithValue(req.Context(), constants.RequestIdentityKey, identity)
	req = req.WithContext(ctx)

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Len(t, client.nemoGuardrails, 1)
}

func TestDeleteGuardrailDeploymentHandlerNotFound(t *testing.T) {
	recorder, _ := setupGuardrailsRequest(
		t, http.MethodDelete, "/api/v1/guardrails/deployments/missing?namespace=test-ns", true)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// performGuardrailsRequest runs a request against an App wired to the given
// fake client and returns the recorder, the decoded body, and the client.
func performGuardrailsRequest(
	t *testing.T,
	client *guardrailsFakeClient,
	method, url string,
	requestBody map[string]interface{},
) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()

	app := &App{
		config: config.EnvConfig{
			AllowedOrigins:              []string{"*"},
			AuthMethod:                  config.AuthMethodInternal,
			GuardrailsTemplateNamespace: "redhat-ods-applications",
		},
		logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
		kubernetesClientFactory: &guardrailsFakeFactory{client: client},
		repositories:            repositories.NewRepositories(),
	}

	var reqBody io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		require.NoError(t, err)
		reqBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)
	req.Header.Set(constants.KubeflowUserIDHeader, "test-user")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	identity := &kubernetes.RequestIdentity{UserID: "test-user", Groups: []string{}}
	ctx := context.WithValue(req.Context(), constants.RequestIdentityKey, identity)
	req = req.WithContext(ctx)

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)

	var body map[string]interface{}
	if recorder.Code == http.StatusOK || recorder.Code == http.StatusCreated {
		err = json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
	}
	return recorder, body
}

func guardrailsTestInferenceServices() []unstructured.Unstructured {
	return []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "serving.kserve.io/v1beta1",
				"kind":       "InferenceService",
				"metadata": map[string]interface{}{
					"namespace": "test-ns",
					"name":      "frauddetection",
					"annotations": map[string]interface{}{
						"security.opendatahub.io/enable-auth": "true",
					},
				},
				"status": map[string]interface{}{
					"address": map[string]interface{}{"url": "https://frauddetection-predictor.test-ns.svc.cluster.local:8080"},
					"conditions": []interface{}{
						map[string]interface{}{"type": "Ready", "status": "True"},
					},
				},
			},
		},
	}
}

func TestGetServingModelsHandler(t *testing.T) {
	client := &guardrailsFakeClient{
		inferenceServices: guardrailsTestInferenceServices(),
		allowList:         true,
	}

	recorder, body := performGuardrailsRequest(t, client, http.MethodGet, "/api/v1/models?namespace=test-ns", nil)
	require.Equal(t, http.StatusOK, recorder.Code)

	data, ok := body["data"].([]interface{})
	require.True(t, ok, "expected data array in response")
	require.Len(t, data, 1)

	model := data[0].(map[string]interface{})
	assert.Equal(t, "frauddetection", model["name"])
	assert.Equal(t, "InferenceService", model["kind"])
	assert.Equal(t, "https://frauddetection-predictor.test-ns.svc.cluster.local:8080/v1", model["endpoint"])
	assert.Equal(t, true, model["ready"])
}

func TestGetServingModelsHandlerForbidden(t *testing.T) {
	client := &guardrailsFakeClient{
		inferenceServices: guardrailsTestInferenceServices(),
		allowList:         false,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodGet, "/api/v1/models?namespace=test-ns", nil)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestGetServingModelsHandlerMissingNamespace(t *testing.T) {
	client := &guardrailsFakeClient{allowList: true}
	recorder, _ := performGuardrailsRequest(t, client, http.MethodGet, "/api/v1/models", nil)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateGuardrailDeploymentHandlerWithModel(t *testing.T) {
	client := &guardrailsFakeClient{
		configMaps:  guardrailsTestConfigMaps(),
		allowList:   true,
		allowCreate: true,
	}

	requestBody := map[string]interface{}{
		"name":         "wired-guardrail",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
		"model": map[string]interface{}{
			"name":     "my-model",
			"endpoint": "https://my-model.test-ns.svc.cluster.local:8000",
		},
	}
	recorder, body := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments", requestBody)
	require.Equal(t, http.StatusCreated, recorder.Code)

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["modelWired"])

	// The CR carries the model wiring environment variables.
	var created *unstructured.Unstructured
	for i := range client.nemoGuardrails {
		if client.nemoGuardrails[i].GetName() == "wired-guardrail" {
			created = &client.nemoGuardrails[i]
		}
	}
	require.NotNil(t, created)
	env, found, err := unstructured.NestedSlice(created.Object, "spec", "env")
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, env, 2)
	assert.Equal(t, map[string]interface{}{"name": "MAIN_MODEL_BASE_URL", "value": "https://my-model.test-ns.svc.cluster.local:8000/v1"}, env[0])
	assert.Equal(t, map[string]interface{}{"name": "MAIN_MODEL_ENGINE", "value": "openai"}, env[1])

	// The copied ConfigMap's config.yaml gained a main model entry.
	var copied *corev1.ConfigMap
	for i := range client.configMaps {
		if client.configMaps[i].Namespace == "test-ns" && client.configMaps[i].Name == "guardrail-wired-guardrail" {
			copied = &client.configMaps[i]
		}
	}
	require.NotNil(t, copied)
	configYAML := copied.Data["config.yaml"]
	assert.Contains(t, configYAML, "model: my-model")
	assert.Contains(t, configYAML, "base_url: https://my-model.test-ns.svc.cluster.local:8000/v1")
	assert.Contains(t, configYAML, "mask pii", "template rails must be preserved")
}

func TestCreateGuardrailDeploymentHandlerInvalidModel(t *testing.T) {
	client := &guardrailsFakeClient{
		configMaps:  guardrailsTestConfigMaps(),
		allowList:   true,
		allowCreate: true,
	}

	requestBody := map[string]interface{}{
		"name":         "wired-guardrail",
		"namespace":    "test-ns",
		"templateName": "trustyai-service-operator-nemo-guardrails-default-pii",
		"model": map[string]interface{}{
			"name":     "my-model",
			"endpoint": "not a url",
		},
	}
	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments", requestBody)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func guardrailsTestReadyWiredDeploymentCR() unstructured.Unstructured {
	cr := guardrailsTestDeploymentCR()
	cr.Object["status"] = map[string]interface{}{
		"phase":    "Ready",
		"endpoint": "https://my-guardrails.test-ns.svc.cluster.local",
		"conditions": []interface{}{
			map[string]interface{}{"type": "DeploymentReady", "status": "True"},
		},
	}
	cr.Object["spec"].(map[string]interface{})["env"] = []interface{}{
		map[string]interface{}{"name": "MAIN_MODEL_BASE_URL", "value": "https://model.test-ns.svc.cluster.local:8000/v1"},
	}
	return cr
}

func TestTestGuardrailDeploymentHandlerMissingMessage(t *testing.T) {
	client := &guardrailsFakeClient{
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestReadyWiredDeploymentCR()},
		allowList:      true,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments/my-guardrails/test?namespace=test-ns", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestTestGuardrailDeploymentHandlerNotFound(t *testing.T) {
	client := &guardrailsFakeClient{allowList: true}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments/missing/test?namespace=test-ns", map[string]interface{}{
		"message": "hello",
	})
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestTestGuardrailDeploymentHandlerNotReady(t *testing.T) {
	// The default test CR has no status.endpoint and no DeploymentReady-wired state.
	client := &guardrailsFakeClient{
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestDeploymentCR()},
		allowList:      true,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments/my-guardrails/test?namespace=test-ns", map[string]interface{}{
		"message": "hello",
	})
	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func TestTestGuardrailDeploymentHandlerNotWired(t *testing.T) {
	cr := guardrailsTestDeploymentCR()
	cr.Object["status"] = map[string]interface{}{
		"phase":    "Ready",
		"endpoint": "https://my-guardrails.test-ns.svc.cluster.local",
		"conditions": []interface{}{
			map[string]interface{}{"type": "DeploymentReady", "status": "True"},
		},
	}
	client := &guardrailsFakeClient{
		nemoGuardrails: []unstructured.Unstructured{cr},
		allowList:      true,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments/my-guardrails/test?namespace=test-ns", map[string]interface{}{
		"message": "hello",
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestTestGuardrailDeploymentHandlerForbidden(t *testing.T) {
	client := &guardrailsFakeClient{
		nemoGuardrails: []unstructured.Unstructured{guardrailsTestReadyWiredDeploymentCR()},
		allowList:      false,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/deployments/my-guardrails/test?namespace=test-ns", map[string]interface{}{
		"message": "hello",
	})
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreateGuardrailTemplateHandler(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: true}

	requestBody := map[string]interface{}{
		"name":        "my-new-template",
		"namespace":   "test-ns",
		"description": "My custom guardrail",
		"configYAML":  "passthrough: true\n",
	}
	recorder, body := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/templates", requestBody)
	require.Equal(t, http.StatusCreated, recorder.Code)

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-new-template", data["name"])
	assert.Equal(t, "user", data["source"])
	assert.Equal(t, "My custom guardrail", data["description"])

	// The ConfigMap was created with the catalog and ownership labels.
	var created *corev1.ConfigMap
	for i := range client.configMaps {
		if client.configMaps[i].Namespace == "test-ns" && client.configMaps[i].Name == "my-new-template" {
			created = &client.configMaps[i]
		}
	}
	require.NotNil(t, created)
	assert.Equal(t, "true", created.Labels["nemo-guardrails-config"])
	assert.Equal(t, "true", created.Labels["opendatahub.io/dashboard"])
}

func TestCreateGuardrailTemplateHandlerForbidden(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: false}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/templates", map[string]interface{}{
		"name":       "my-new-template",
		"namespace":  "test-ns",
		"configYAML": "passthrough: true\n",
	})
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreateGuardrailTemplateHandlerInvalidYAML(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: true}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/templates", map[string]interface{}{
		"name":       "my-new-template",
		"namespace":  "test-ns",
		"configYAML": "\tinvalid: [yaml",
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateGuardrailTemplateHandlerPlatformNamespace(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: true}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPost, "/api/v1/guardrails/templates", map[string]interface{}{
		"name":       "my-new-template",
		"namespace":  "redhat-ods-applications",
		"configYAML": "passthrough: true\n",
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUpdateGuardrailTemplateHandler(t *testing.T) {
	client := &guardrailsFakeClient{
		configMaps: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "my-template",
					Labels: map[string]string{
						"nemo-guardrails-config":   "true",
						"opendatahub.io/dashboard": "true",
					},
				},
				Data: map[string]string{
					"config.yaml": "passthrough: true\n",
					"rails.co":    "define flow",
				},
			},
		},
		allowCreate: true,
	}

	requestBody := map[string]interface{}{
		"description": "Updated",
		"configYAML":  "passthrough: false\n",
	}
	recorder, body := performGuardrailsRequest(t, client, http.MethodPut, "/api/v1/guardrails/templates/my-template?namespace=test-ns", requestBody)
	require.Equal(t, http.StatusOK, recorder.Code)

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Updated", data["description"])

	updated := client.configMaps[0]
	assert.Equal(t, "passthrough: false\n", updated.Data["config.yaml"])
	assert.Equal(t, "define flow", updated.Data["rails.co"])
}

func TestUpdateGuardrailTemplateHandlerNotDashboardManaged(t *testing.T) {
	client := &guardrailsFakeClient{
		configMaps: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "hand-made",
					Labels:    map[string]string{"nemo-guardrails-config": "true"},
				},
				Data: map[string]string{"config.yaml": "passthrough: true\n"},
			},
		},
		allowCreate: true,
	}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPut, "/api/v1/guardrails/templates/hand-made?namespace=test-ns", map[string]interface{}{
		"configYAML": "passthrough: false\n",
	})
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestUpdateGuardrailTemplateHandlerPlatformNamespace(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: true}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPut, "/api/v1/guardrails/templates/some-template?namespace=redhat-ods-applications", map[string]interface{}{
		"configYAML": "passthrough: false\n",
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUpdateGuardrailTemplateHandlerNotFound(t *testing.T) {
	client := &guardrailsFakeClient{allowCreate: true}

	recorder, _ := performGuardrailsRequest(t, client, http.MethodPut, "/api/v1/guardrails/templates/missing?namespace=test-ns", map[string]interface{}{
		"configYAML": "passthrough: false\n",
	})
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
