package repositories

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeGuardrailsClient is a lightweight KubernetesClientInterface fake backed
// by in-memory ConfigMaps and NemoGuardrails resources.
type fakeGuardrailsClient struct {
	kubernetes.KubernetesClientInterface

	configMaps             []corev1.ConfigMap
	nemoGuardrails         []unstructured.Unstructured
	inferenceServices      []unstructured.Unstructured
	llmInferenceServices   []unstructured.Unstructured
	forbidInferenceList    bool
	forbidLLMInferenceList bool
	allowList              bool
	allowCreate            bool

	createdConfigMaps     []corev1.ConfigMap
	createdNemoGuardrails []unstructured.Unstructured
}

func (f *fakeGuardrailsClient) ListInferenceServices(_ context.Context, namespace string) ([]unstructured.Unstructured, error) {
	if f.forbidInferenceList {
		return nil, apierrors.NewForbidden(schema.GroupResource{Group: "serving.kserve.io", Resource: "inferenceservices"}, namespace, errors.New("forbidden"))
	}
	var result []unstructured.Unstructured
	for _, item := range f.inferenceServices {
		if item.GetNamespace() == namespace {
			result = append(result, item)
		}
	}
	return result, nil
}

func (f *fakeGuardrailsClient) ListLLMInferenceServices(_ context.Context, namespace string) ([]unstructured.Unstructured, error) {
	if f.forbidLLMInferenceList {
		return nil, apierrors.NewForbidden(schema.GroupResource{Group: "serving.kserve.io", Resource: "llminferenceservices"}, namespace, errors.New("forbidden"))
	}
	var result []unstructured.Unstructured
	for _, item := range f.llmInferenceServices {
		if item.GetNamespace() == namespace {
			result = append(result, item)
		}
	}
	return result, nil
}

func (f *fakeGuardrailsClient) BearerToken() (string, error) {
	return "test-token", nil
}

func (f *fakeGuardrailsClient) UpdateConfigMap(_ context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == configMap.Name {
			updated := *configMap
			updated.Namespace = namespace
			updated.ResourceVersion = "2"
			f.configMaps[i] = updated
			return &updated, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, configMap.Name)
}

func (f *fakeGuardrailsClient) ListConfigMapsByLabel(_ context.Context, namespace string, labelSelector string) ([]corev1.ConfigMap, error) {
	// The fake honors only the nemo-guardrails-config selector.
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

func (f *fakeGuardrailsClient) GetConfigMap(_ context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == name {
			return &f.configMaps[i], nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, name)
}

func (f *fakeGuardrailsClient) ListNemoGuardrails(_ context.Context, namespace string) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured
	for _, item := range f.nemoGuardrails {
		if item.GetNamespace() == namespace {
			result = append(result, item)
		}
	}
	return result, nil
}

func (f *fakeGuardrailsClient) GetNemoGuardrails(_ context.Context, namespace string, name string) (*unstructured.Unstructured, error) {
	for i := range f.nemoGuardrails {
		if f.nemoGuardrails[i].GetNamespace() == namespace && f.nemoGuardrails[i].GetName() == name {
			clone := f.nemoGuardrails[i]
			return &clone, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, name)
}

func (f *fakeGuardrailsClient) CanListNemoGuardrails(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowList, nil
}

func (f *fakeGuardrailsClient) CanCreateGuardrailDeployment(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return f.allowCreate, nil
}

func (f *fakeGuardrailsClient) CreateConfigMap(_ context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	for i := range f.configMaps {
		if f.configMaps[i].Namespace == namespace && f.configMaps[i].Name == configMap.Name {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "", Resource: "configmaps"}, configMap.Name)
		}
	}
	created := *configMap
	created.Namespace = namespace
	f.configMaps = append(f.configMaps, created)
	f.createdConfigMaps = append(f.createdConfigMaps, created)
	return &created, nil
}

func (f *fakeGuardrailsClient) CreateNemoGuardrails(_ context.Context, namespace string, cr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	for i := range f.nemoGuardrails {
		if f.nemoGuardrails[i].GetNamespace() == namespace && f.nemoGuardrails[i].GetName() == cr.GetName() {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, cr.GetName())
		}
	}
	created := *cr
	created.SetNamespace(namespace)
	f.nemoGuardrails = append(f.nemoGuardrails, created)
	f.createdNemoGuardrails = append(f.createdNemoGuardrails, created)
	return &created, nil
}

func testConfigMap(namespace, name string, labels map[string]string, data map[string]string) corev1.ConfigMap {
	return testConfigMapWithAnnotations(namespace, name, labels, data, nil)
}

func testConfigMapWithAnnotations(
	namespace, name string,
	labels map[string]string,
	data map[string]string,
	annotations map[string]string,
) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         namespace,
			Name:              name,
			Labels:            labels,
			Annotations:       annotations,
			CreationTimestamp: metav1.Now(),
		},
		Data: data,
	}
}

func testNemoGuardrails(namespace, name string, spec map[string]interface{}, status map[string]interface{}) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "trustyai.opendatahub.io/v1alpha1",
			"kind":       "NemoGuardrails",
			"metadata": map[string]interface{}{
				"namespace":         namespace,
				"name":              name,
				"creationTimestamp": "2026-01-15T10:30:00Z",
			},
			"spec":   spec,
			"status": status,
		},
	}
}

const customConfigYAML = `models:
  - type: main
    engine: openai
    model: placeholder
rails:
  input:
    flows:
      - self check input
  output:
    flows:
      - self check output
      - mask sensitive data
prompts:
  - task: self_check_input
    content: check input
  - task: self_check_output
    content: check output
`

const passthroughConfigYAML = `passthrough: true
rails:
  input:
    flows:
      - injection detection
`

func TestParseConfigSummary(t *testing.T) {
	summary := parseConfigSummary(customConfigYAML)

	assert.Equal(t, 1, summary.ModelCount)
	assert.Equal(t, []string{"openai/placeholder (main)"}, summary.Models)
	assert.Equal(t, 2, summary.PromptCount)
	assert.False(t, summary.Passthrough)
	assert.Len(t, summary.RailFlows, 3)
	assert.Contains(t, summary.RailFlows, models.GuardrailRailFlow{Category: "input", Flow: "self check input"})
	assert.Contains(t, summary.RailFlows, models.GuardrailRailFlow{Category: "output", Flow: "mask sensitive data"})
}

func TestParseConfigSummaryPassthrough(t *testing.T) {
	summary := parseConfigSummary(passthroughConfigYAML)

	assert.True(t, summary.Passthrough)
	assert.Equal(t, 0, summary.ModelCount)
	assert.Empty(t, summary.Models)
	assert.Len(t, summary.RailFlows, 1)
}

func TestParseConfigSummaryInvalidYAML(t *testing.T) {
	summary := parseConfigSummary("\tnot: [valid")

	assert.Equal(t, 0, summary.ModelCount)
	assert.Empty(t, summary.Models)
	assert.Empty(t, summary.RailFlows)
	assert.Equal(t, 0, summary.PromptCount)
}

func TestGetTemplates(t *testing.T) {
	client := &fakeGuardrailsClient{
		configMaps: []corev1.ConfigMap{
			testConfigMapWithAnnotations("redhat-ods-applications", "trustyai-service-operator-nemo-guardrails-default-safety",
				map[string]string{"nemo-guardrails-config": "true"},
				map[string]string{"config.yaml": passthroughConfigYAML, "rails.co": "..."},
				map[string]string{
					"guardrail-description": "A guardrail that checks for violence, profanity, hate, and toxicity.",
					"guardrail-actions":     `["block"]`,
					"guardrail-locations":   `["input", "output"]`,
					"guardrail-tags":        `["content-safety", "toxicity"]`,
				}),
			testConfigMap("redhat-ods-applications", "trustyai-service-operator-nemo-guardrails-default-all",
				map[string]string{"nemo-guardrails-config": "true"},
				map[string]string{"config.yaml": customConfigYAML}),
			testConfigMap("other-namespace", "unrelated",
				map[string]string{"nemo-guardrails-config": "true"},
				map[string]string{"config.yaml": "x: 1"}),
		},
	}

	repo := NewGuardrailsRepository(nil)
	templates, err := repo.GetTemplates(client, nil, context.Background(), "redhat-ods-applications", "")
	require.NoError(t, err)
	require.Len(t, templates, 2)

	// Sorted by name
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-all", templates[0].Name)
	assert.Equal(t, []string{"config.yaml"}, templates[0].Keys)
	assert.Equal(t, 1, templates[0].Summary.ModelCount)
	assert.Empty(t, templates[0].Description)
	// No actions annotation: derived from the config's rail flows (self
	// check flows block, the mask flow masks) on input and output.
	assert.Equal(t, []string{"block", "mask"}, templates[0].Actions)
	assert.Equal(t, []string{"input", "output"}, templates[0].Locations)

	safety := templates[1]
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-safety", safety.Name)
	assert.Equal(t, []string{"config.yaml", "rails.co"}, safety.Keys)
	assert.True(t, safety.Summary.Passthrough)
	assert.Equal(t, "A guardrail that checks for violence, profanity, hate, and toxicity.", safety.Description)
	assert.Equal(t, []string{"block"}, safety.Actions)
	assert.Equal(t, []string{"input", "output"}, safety.Locations)
	assert.Equal(t, []string{"content-safety", "toxicity"}, safety.Tags)
}

func TestParseAnnotationList(t *testing.T) {
	assert.Nil(t, parseAnnotationList(""))
	assert.Nil(t, parseAnnotationList("   "))
	assert.Nil(t, parseAnnotationList("not json"))
	assert.Equal(t, []string{"block", "mask"}, parseAnnotationList(`["block","mask"]`))
	// Non-array JSON values yield nil rather than erroring
	assert.Nil(t, parseAnnotationList(`"block"`))
}

func TestGetTemplateDetail(t *testing.T) {
	client := &fakeGuardrailsClient{
		configMaps: []corev1.ConfigMap{
			testConfigMap("redhat-ods-applications", "my-template",
				map[string]string{"nemo-guardrails-config": "true"},
				map[string]string{"config.yaml": customConfigYAML, "actions.py": "print('hi')"}),
		},
	}

	repo := NewGuardrailsRepository(nil)
	detail, err := repo.GetTemplate(client, context.Background(), "redhat-ods-applications", "my-template")
	require.NoError(t, err)
	assert.Equal(t, "my-template", detail.Name)
	require.Len(t, detail.Files, 2)
	// Files are ordered by key (sorted)
	assert.Equal(t, "actions.py", detail.Files[0].Key)
	assert.Equal(t, "print('hi')", detail.Files[0].Content)
	assert.Equal(t, "config.yaml", detail.Files[1].Key)
	assert.Equal(t, customConfigYAML, detail.Files[1].Content)
}

func TestGetTemplateDetailNotFound(t *testing.T) {
	repo := NewGuardrailsRepository(nil)
	_, err := repo.GetTemplate(&fakeGuardrailsClient{}, context.Background(), "redhat-ods-applications", "missing")
	assert.True(t, apierrors.IsNotFound(err))
}

func deploymentCR() unstructured.Unstructured {
	return testNemoGuardrails("test-ns", "my-guardrails",
		map[string]interface{}{
			"replicas":    int64(1),
			"exposeRoute": true,
			"nemoConfigs": []interface{}{
				map[string]interface{}{
					"name":       "main-config",
					"configMaps": []interface{}{"guardrail-placeholder"},
					"default":    true,
				},
			},
			"env": []interface{}{
				map[string]interface{}{"name": "OPENAI_API_KEY", "value": "super-secret"},
			},
		},
		map[string]interface{}{
			"phase":    "Ready",
			"endpoint": "https://my-guardrails.test-ns.svc.cluster.local",
			"conditions": []interface{}{
				map[string]interface{}{
					"type":               "DeploymentReady",
					"status":             "True",
					"reason":             "DeploymentReady",
					"message":            "Deployment is ready",
					"lastTransitionTime": "2026-01-15T10:31:00Z",
				},
				map[string]interface{}{
					"type":    "RouteReady",
					"status":  "True",
					"reason":  "RouteReady",
					"message": "Route is ready",
				},
			},
		})
}

func TestGetDeployments(t *testing.T) {
	client := &fakeGuardrailsClient{
		nemoGuardrails: []unstructured.Unstructured{deploymentCR()},
		configMaps: []corev1.ConfigMap{
			testConfigMap("test-ns", "guardrail-placeholder",
				map[string]string{"nemo-guardrails-config": "true", "opendatahub.io/dashboard": "true"},
				map[string]string{"config.yaml": customConfigYAML, "prompts.yml": "..."}),
		},
	}

	repo := NewGuardrailsRepository(nil)
	deployments, err := repo.GetDeployments(client, context.Background(), "test-ns")
	require.NoError(t, err)
	require.Len(t, deployments, 1)

	deployment := deployments[0]
	assert.Equal(t, "my-guardrails", deployment.Name)
	assert.Equal(t, "test-ns", deployment.Namespace)
	assert.Equal(t, "Ready", deployment.Phase)
	assert.True(t, deployment.Ready)
	assert.Equal(t, "https://my-guardrails.test-ns.svc.cluster.local", deployment.Endpoint)
	require.NotNil(t, deployment.Replicas)
	assert.Equal(t, int64(1), *deployment.Replicas)
	require.Len(t, deployment.Conditions, 2)
	assert.Equal(t, "DeploymentReady", deployment.Conditions[0].Type)

	require.Len(t, deployment.ConfigSources, 1)
	source := deployment.ConfigSources[0]
	assert.Equal(t, "guardrail-placeholder", source.Name)
	assert.True(t, source.Found)
	assert.Equal(t, []string{"config.yaml", "prompts.yml"}, source.Keys)
	assert.True(t, source.HasModelSpec)
	assert.False(t, source.IsOperatorTemplate)
}

func TestGetDeploymentsMissingConfigMap(t *testing.T) {
	client := &fakeGuardrailsClient{
		nemoGuardrails: []unstructured.Unstructured{deploymentCR()},
	}

	repo := NewGuardrailsRepository(nil)
	deployments, err := repo.GetDeployments(client, context.Background(), "test-ns")
	require.NoError(t, err)
	require.Len(t, deployments, 1)

	require.Len(t, deployments[0].ConfigSources, 1)
	source := deployments[0].ConfigSources[0]
	assert.False(t, source.Found)
	assert.Empty(t, source.Keys)
}

func TestGetDeploymentsNotReady(t *testing.T) {
	cr := deploymentCR()
	_ = unstructured.SetNestedSlice(cr.Object, []interface{}{
		map[string]interface{}{
			"type":    "DeploymentReady",
			"status":  "False",
			"reason":  "DeploymentFailed",
			"message": "deployment failed",
		},
	}, "status", "conditions")
	_ = unstructured.SetNestedField(cr.Object, "Failed", "status", "phase")

	client := &fakeGuardrailsClient{nemoGuardrails: []unstructured.Unstructured{cr}}

	repo := NewGuardrailsRepository(nil)
	deployments, err := repo.GetDeployments(client, context.Background(), "test-ns")
	require.NoError(t, err)
	require.Len(t, deployments, 1)
	assert.False(t, deployments[0].Ready)
	assert.Equal(t, "Failed", deployments[0].Phase)
}

func TestGetDeploymentDetail(t *testing.T) {
	client := &fakeGuardrailsClient{
		nemoGuardrails: []unstructured.Unstructured{deploymentCR()},
		configMaps: []corev1.ConfigMap{
			testConfigMap("test-ns", "guardrail-placeholder",
				map[string]string{
					"nemo-guardrails-config":          "true",
					"platform.opendatahub.io/part-of": "trustyai-module-operator",
				},
				map[string]string{"config.yaml": passthroughConfigYAML}),
		},
	}

	repo := NewGuardrailsRepository(nil)
	detail, err := repo.GetDeployment(client, context.Background(), "test-ns", "my-guardrails")
	require.NoError(t, err)
	assert.Equal(t, "my-guardrails", detail.Name)
	assert.True(t, detail.ExposeRoute)
	require.Len(t, detail.NemoConfigs, 1)
	assert.Equal(t, "main-config", detail.NemoConfigs[0].Name)
	assert.Equal(t, []string{"guardrail-placeholder"}, detail.NemoConfigs[0].ConfigMaps)
	assert.True(t, detail.NemoConfigs[0].Default)

	// Only env var names are surfaced, never values.
	assert.Equal(t, []string{"OPENAI_API_KEY"}, detail.EnvVarNames)

	require.Len(t, detail.ConfigSources, 1)
	source := detail.ConfigSources[0]
	assert.True(t, source.IsOperatorTemplate)
	assert.False(t, source.HasModelSpec)
}

func TestGetDeploymentDetailNotFound(t *testing.T) {
	repo := NewGuardrailsRepository(nil)
	_, err := repo.GetDeployment(&fakeGuardrailsClient{}, context.Background(), "test-ns", "missing")
	assert.True(t, apierrors.IsNotFound(err))
}

func TestValidateDeploymentName(t *testing.T) {
	assert.NoError(t, ValidateDeploymentName("my-guardrail"))
	assert.NoError(t, ValidateDeploymentName("a"))
	assert.NoError(t, ValidateDeploymentName("a-b-c-123"))

	assert.Error(t, ValidateDeploymentName(""))
	assert.Error(t, ValidateDeploymentName("My-Guardrail"))
	assert.Error(t, ValidateDeploymentName("-starts-with-hyphen"))
	assert.Error(t, ValidateDeploymentName("ends-with-hyphen-"))
	assert.Error(t, ValidateDeploymentName("has_underscore"))
	assert.Error(t, ValidateDeploymentName("has.dot"))
	assert.Error(t, ValidateDeploymentName("way-too-long-"+strings.Repeat("a", 60)))
}

func TestCreateDeploymentFromTemplate(t *testing.T) {
	template := testConfigMap("redhat-ods-applications", "trustyai-service-operator-nemo-guardrails-default-pii",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML, "rails.co": "..."})

	client := &fakeGuardrailsClient{}
	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{template}}

	repo := NewGuardrailsRepository(nil)
	deployment, err := repo.CreateDeploymentFromTemplate(
		client, internalClient, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{
			Name:         "my-pii-guardrail",
			Namespace:    "test-ns",
			TemplateName: "trustyai-service-operator-nemo-guardrails-default-pii",
		})
	require.NoError(t, err)
	assert.Equal(t, "my-pii-guardrail", deployment.Name)
	assert.Equal(t, "test-ns", deployment.Namespace)

	// The template ConfigMap is copied into the target namespace with the
	// dashboard and guardrails-config labels.
	require.Len(t, client.createdConfigMaps, 1)
	copyCM := client.createdConfigMaps[0]
	assert.Equal(t, "guardrail-my-pii-guardrail", copyCM.Name)
	assert.Equal(t, "test-ns", copyCM.Namespace)
	assert.Equal(t, template.Data, copyCM.Data)
	assert.Equal(t, "true", copyCM.Labels["nemo-guardrails-config"])
	assert.Equal(t, "true", copyCM.Labels["opendatahub.io/dashboard"])
	// The copy records the template it was made from, so deployments can
	// show their template origin.
	assert.Equal(
		t,
		"redhat-ods-applications/trustyai-service-operator-nemo-guardrails-default-pii",
		copyCM.Annotations["opendatahub.io/template-source"],
	)

	// The NemoGuardrails CR references the copy.
	require.Len(t, client.createdNemoGuardrails, 1)
	cr := client.createdNemoGuardrails[0]
	assert.Equal(t, "my-pii-guardrail", cr.GetName())
	assert.Equal(t, "NemoGuardrails", cr.GetKind())
	nemoConfigs, found, err := unstructured.NestedSlice(cr.Object, "spec", "nemoConfigs")
	require.NoError(t, err)
	require.True(t, found)
	entry := nemoConfigs[0].(map[string]interface{})
	assert.Equal(t, "my-pii-guardrail", entry["name"])
	configMaps, _, _ := unstructured.NestedStringSlice(entry, "configMaps")
	assert.Equal(t, []string{"guardrail-my-pii-guardrail"}, configMaps)
	isDefault, _, _ := unstructured.NestedBool(entry, "default")
	assert.True(t, isDefault)

	// Route exposure and auth annotation match the known-working pattern.
	assert.True(t, nestedBoolForTest(&cr, "spec", "exposeRoute"))
	annotations := cr.GetAnnotations()
	assert.Equal(t, "true", annotations["security.opendatahub.io/enable-auth"])

	// The created deployment reports its config source.
	require.Len(t, deployment.ConfigSources, 1)
	assert.Equal(t, "guardrail-my-pii-guardrail", deployment.ConfigSources[0].Name)
	assert.True(t, deployment.ConfigSources[0].Found)
}

func TestCreateDeploymentFromTemplateInvalidName(t *testing.T) {
	repo := NewGuardrailsRepository(nil)
	_, err := repo.CreateDeploymentFromTemplate(
		&fakeGuardrailsClient{}, &fakeGuardrailsClient{}, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{Name: "Invalid Name", Namespace: "test-ns", TemplateName: "tpl"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid deployment name")
}

func TestCreateDeploymentFromTemplateMissingTemplate(t *testing.T) {
	repo := NewGuardrailsRepository(nil)
	_, err := repo.CreateDeploymentFromTemplate(
		&fakeGuardrailsClient{}, &fakeGuardrailsClient{}, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{Name: "my-guardrail", Namespace: "test-ns", TemplateName: "missing"})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestCreateDeploymentFromTemplateAlreadyExists(t *testing.T) {
	template := testConfigMap("redhat-ods-applications", "tpl",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	existingCR := deploymentCR() // name: my-guardrails in test-ns

	client := &fakeGuardrailsClient{nemoGuardrails: []unstructured.Unstructured{existingCR}}
	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{template}}

	repo := NewGuardrailsRepository(nil)
	_, err := repo.CreateDeploymentFromTemplate(
		client, internalClient, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{Name: "my-guardrails", Namespace: "test-ns", TemplateName: "tpl"})
	// The ConfigMap copy succeeds but the CR conflicts.
	assert.True(t, apierrors.IsAlreadyExists(err))
}

func TestCreateDeploymentFromTemplateConfigMapConflict(t *testing.T) {
	template := testConfigMap("redhat-ods-applications", "tpl",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	existingCopy := testConfigMap("test-ns", "guardrail-my-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": "old"})

	client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{existingCopy}}
	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{template}}

	repo := NewGuardrailsRepository(nil)
	_, err := repo.CreateDeploymentFromTemplate(
		client, internalClient, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{Name: "my-guardrail", Namespace: "test-ns", TemplateName: "tpl"})
	assert.True(t, apierrors.IsAlreadyExists(err))
}

// nestedBoolForTest is a small test helper for unstructured bool access.
func nestedBoolForTest(item *unstructured.Unstructured, fields ...string) bool {
	value, found, err := unstructured.NestedBool(item.Object, fields...)
	return found && err == nil && value
}

func TestGetTemplatesMergesUserNamespace(t *testing.T) {
	platformTemplate := testConfigMap("redhat-ods-applications", "trustyai-service-operator-nemo-guardrails-default-pii",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	userTemplate := testConfigMap("test-ns", "my-custom-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": customConfigYAML})

	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{platformTemplate}}
	userClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{userTemplate}}

	repo := NewGuardrailsRepository(nil)
	templates, err := repo.GetTemplates(
		internalClient, userClient, context.Background(), "redhat-ods-applications", "test-ns")
	require.NoError(t, err)
	require.Len(t, templates, 2)

	// Sorted by name: my-custom-guardrail < trustyai-...
	assert.Equal(t, "my-custom-guardrail", templates[0].Name)
	assert.Equal(t, "user", templates[0].Source)
	assert.Equal(t, "test-ns", templates[0].Namespace)
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-pii", templates[1].Name)
	assert.Equal(t, "platform", templates[1].Source)
}

func TestGetTemplatesExcludesDeploymentCopies(t *testing.T) {
	platformTemplate := testConfigMap("redhat-ods-applications", "trustyai-service-operator-nemo-guardrails-default-pii",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	userTemplate := testConfigMap("test-ns", "my-custom-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": customConfigYAML})
	// The per-deployment copy a template deployment creates: labeled like a
	// template (the running guardrail needs it) but marked as runtime config.
	deploymentCopy := testConfigMap("test-ns", "guardrail-my-deployment",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	deploymentCopy.Labels["opendatahub.io/deployment-config"] = "true"
	deploymentCopy.Annotations = map[string]string{
		"opendatahub.io/template-source": "redhat-ods-applications/trustyai-service-operator-nemo-guardrails-default-pii",
	}
	// A copy created before the deployment-config label existed is still
	// recognized by its recorded template source.
	legacyCopy := testConfigMap("test-ns", "guardrail-legacy-deployment",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})
	legacyCopy.Annotations = map[string]string{
		"opendatahub.io/template-source": "redhat-ods-applications/trustyai-service-operator-nemo-guardrails-default-pii",
	}

	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{platformTemplate}}
	userClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{
		userTemplate, deploymentCopy, legacyCopy,
	}}

	repo := NewGuardrailsRepository(nil)
	templates, err := repo.GetTemplates(
		internalClient, userClient, context.Background(), "redhat-ods-applications", "test-ns")
	require.NoError(t, err)
	require.Len(t, templates, 2)

	assert.Equal(t, "my-custom-guardrail", templates[0].Name)
	assert.Equal(t, "user", templates[0].Source)
	assert.Equal(t, "trustyai-service-operator-nemo-guardrails-default-pii", templates[1].Name)
	assert.Equal(t, "platform", templates[1].Source)
}

func TestGetTemplatesUserNamespaceForbidden(t *testing.T) {
	platformTemplate := testConfigMap("redhat-ods-applications", "tpl",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": passthroughConfigYAML})

	internalClient := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{platformTemplate}}
	forbiddenClient := &forbiddenConfigMapsClient{}

	repo := NewGuardrailsRepository(nil)
	templates, err := repo.GetTemplates(
		internalClient, forbiddenClient, context.Background(), "redhat-ods-applications", "test-ns")
	require.NoError(t, err)
	require.Len(t, templates, 1)
	assert.Equal(t, "platform", templates[0].Source)
}

// forbiddenConfigMapsClient denies all ConfigMap listing.
type forbiddenConfigMapsClient struct {
	fakeGuardrailsClient
}

func (f *forbiddenConfigMapsClient) ListConfigMapsByLabel(_ context.Context, _ string, _ string) ([]corev1.ConfigMap, error) {
	return nil, apierrors.NewForbidden(schema.GroupResource{Group: "", Resource: "configmaps"}, "", nil)
}

func TestCreateDeploymentFromUserTemplate(t *testing.T) {
	userTemplate := testConfigMap("test-ns", "my-custom-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": customConfigYAML})

	// The user template lives in the user's namespace and is read with the
	// user's client; nothing lives in the platform namespace.
	client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{userTemplate}}
	internalClient := &fakeGuardrailsClient{}

	repo := NewGuardrailsRepository(nil)
	deployment, err := repo.CreateDeploymentFromTemplate(
		client, internalClient, context.Background(), "redhat-ods-applications",
		models.GuardrailDeploymentCreateRequest{
			Name:              "from-user-template",
			Namespace:         "test-ns",
			TemplateName:      "my-custom-guardrail",
			TemplateNamespace: "test-ns",
		})
	require.NoError(t, err)
	assert.Equal(t, "from-user-template", deployment.Name)

	// The copy lands in the target namespace with the user template's data.
	require.Len(t, client.createdConfigMaps, 1)
	assert.Equal(t, "guardrail-from-user-template", client.createdConfigMaps[0].Name)
	assert.Equal(t, customConfigYAML, client.createdConfigMaps[0].Data["config.yaml"])
}

func (f *fakeGuardrailsClient) DeleteNemoGuardrails(_ context.Context, namespace string, name string) error {
	for i, item := range f.nemoGuardrails {
		if item.GetNamespace() == namespace && item.GetName() == name {
			f.nemoGuardrails = append(f.nemoGuardrails[:i], f.nemoGuardrails[i+1:]...)
			return nil
		}
	}
	return apierrors.NewNotFound(schema.GroupResource{Group: "trustyai.opendatahub.io", Resource: "nemoguardrails"}, name)
}

func (f *fakeGuardrailsClient) DeleteConfigMap(_ context.Context, namespace string, name string) error {
	for i, cm := range f.configMaps {
		if cm.Namespace == namespace && cm.Name == name {
			f.configMaps = append(f.configMaps[:i], f.configMaps[i+1:]...)
			return nil
		}
	}
	return apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, name)
}

func (f *fakeGuardrailsClient) CanDeleteNemoGuardrails(_ context.Context, _ *kubernetes.RequestIdentity, _ string) (bool, error) {
	return true, nil
}

func TestDeleteDeployment(t *testing.T) {
	cr := deploymentCR() // references guardrail-placeholder in test-ns
	dashboardCopy := testConfigMap("test-ns", "guardrail-placeholder",
		map[string]string{"nemo-guardrails-config": "true", "opendatahub.io/dashboard": "true"},
		map[string]string{"config.yaml": customConfigYAML})

	client := &fakeGuardrailsClient{
		nemoGuardrails: []unstructured.Unstructured{cr},
		configMaps:     []corev1.ConfigMap{dashboardCopy},
	}

	repo := NewGuardrailsRepository(nil)
	err := repo.DeleteDeployment(client, context.Background(), "test-ns", "my-guardrails")
	require.NoError(t, err)

	// Both the CR and the dashboard-created ConfigMap copy are gone.
	assert.Empty(t, client.nemoGuardrails)
	assert.Empty(t, client.configMaps)
}

func TestDeleteDeploymentKeepsUserManagedConfigMaps(t *testing.T) {
	cr := deploymentCR() // references guardrail-placeholder
	// A user-managed ConfigMap with the same reference: no dashboard label.
	userConfigMap := testConfigMap("test-ns", "guardrail-placeholder",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": customConfigYAML})

	client := &fakeGuardrailsClient{
		nemoGuardrails: []unstructured.Unstructured{cr},
		configMaps:     []corev1.ConfigMap{userConfigMap},
	}

	repo := NewGuardrailsRepository(nil)
	err := repo.DeleteDeployment(client, context.Background(), "test-ns", "my-guardrails")
	require.NoError(t, err)

	// The CR is gone; the user-managed ConfigMap remains.
	assert.Empty(t, client.nemoGuardrails)
	require.Len(t, client.configMaps, 1)
	assert.Equal(t, "guardrail-placeholder", client.configMaps[0].Name)
}

func TestDeleteDeploymentNotFound(t *testing.T) {
	repo := NewGuardrailsRepository(nil)
	err := repo.DeleteDeployment(&fakeGuardrailsClient{}, context.Background(), "test-ns", "missing")
	assert.True(t, apierrors.IsNotFound(err))
}

func TestDeriveGuardrailActionsAndLocations(t *testing.T) {
	tests := []struct {
		name              string
		summary           models.GuardrailConfigSummary
		expectedActions   []string
		expectedLocations []string
	}{
		{
			name: "input checks block on input",
			summary: models.GuardrailConfigSummary{RailFlows: []models.GuardrailRailFlow{
				{Category: "input", Flow: "injection detection"},
				{Category: "input", Flow: "hf classifier check windowed input $classifier=\"prompt-injection\""},
			}},
			expectedActions:   []string{"block"},
			expectedLocations: []string{"input"},
		},
		{
			name: "masking flows mask, other flows block",
			summary: models.GuardrailConfigSummary{RailFlows: []models.GuardrailRailFlow{
				{Category: "input", Flow: "mask sensitive data on input"},
				{Category: "output", Flow: "self check output"},
			}},
			expectedActions:   []string{"block", "mask"},
			expectedLocations: []string{"input", "output"},
		},
		{
			name: "redaction flows mask",
			summary: models.GuardrailConfigSummary{RailFlows: []models.GuardrailRailFlow{
				{Category: "output", Flow: "redact sensitive data on output"},
			}},
			expectedActions:   []string{"mask"},
			expectedLocations: []string{"output"},
		},
		{
			name:              "passthrough configs yield neither",
			summary:           models.GuardrailConfigSummary{Passthrough: true},
			expectedActions:   nil,
			expectedLocations: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, locations := deriveGuardrailActionsAndLocations(tt.summary)
			assert.Equal(t, tt.expectedActions, actions)
			assert.Equal(t, tt.expectedLocations, locations)
		})
	}
}

func TestNewTemplateModelDerivesActionsAndLocationsWithoutAnnotations(t *testing.T) {
	// A user-created template carries only the description annotation; the
	// guard actions and locations come from its config.yaml.
	configMap := testConfigMap("test-ns", "my-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": "passthrough: true\nrails:\n  input:\n    flows:\n      - injection detection\n"})
	configMap.Annotations = map[string]string{"guardrail-description": "My guardrail"}

	template := newTemplateModel(&configMap)
	assert.Equal(t, "My guardrail", template.Description)
	assert.Equal(t, []string{"block"}, template.Actions)
	assert.Equal(t, []string{"input"}, template.Locations)

	// Annotations, when present, win over the derived values.
	annotated := testConfigMapWithAnnotations("test-ns", "annotated-guardrail",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": "passthrough: true\nrails:\n  input:\n    flows:\n      - injection detection\n"},
		map[string]string{
			"guardrail-actions":   `["mask"]`,
			"guardrail-locations": `["input", "output"]`,
		})
	derived := newTemplateModel(&annotated)
	assert.Equal(t, []string{"mask"}, derived.Actions)
	assert.Equal(t, []string{"input", "output"}, derived.Locations)
}
