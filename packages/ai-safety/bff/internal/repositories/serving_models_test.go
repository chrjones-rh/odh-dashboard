package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// testInferenceService builds an InferenceService fixture. Annotations are
// map[string]interface{} to match what the dynamic client's JSON decoding
// produces (GetAnnotations requires that shape).
func testInferenceService(namespace, name string, annotations map[string]string, status map[string]interface{}) unstructured.Unstructured {
	var rawAnnotations map[string]interface{}
	if annotations != nil {
		rawAnnotations = make(map[string]interface{}, len(annotations))
		for key, value := range annotations {
			rawAnnotations[key] = value
		}
	}
	item := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.kserve.io/v1beta1",
			"kind":       "InferenceService",
			"metadata": map[string]interface{}{
				"namespace":   namespace,
				"name":        name,
				"annotations": rawAnnotations,
			},
			"status": status,
		},
	}
	return item
}

func TestServingEndpointAuthAnnotation(t *testing.T) {
	tests := []struct {
		name             string
		annotations      map[string]string
		address          string
		expectedEndpoint string
		expectOK         bool
	}{
		{
			name:             "auth-enabled keeps https and appends /v1",
			annotations:      map[string]string{"security.opendatahub.io/enable-auth": "true"},
			address:          "https://my-model-kserve-workload-svc.my-ns.svc.cluster.local:8000",
			expectedEndpoint: "https://my-model-kserve-workload-svc.my-ns.svc.cluster.local:8000/v1",
			expectOK:         true,
		},
		{
			name:             "without auth annotation forces http",
			annotations:      nil,
			address:          "https://my-model-predictor.my-ns.svc.cluster.local:8080",
			expectedEndpoint: "http://my-model-predictor.my-ns.svc.cluster.local:8080/v1",
			expectOK:         true,
		},
		{
			name:             "existing /v1 suffix is not duplicated",
			annotations:      map[string]string{"security.opendatahub.io/enable-auth": "true"},
			address:          "https://my-model.my-ns.svc.cluster.local:8000/v1",
			expectedEndpoint: "https://my-model.my-ns.svc.cluster.local:8000/v1",
			expectOK:         true,
		},
		{
			name:     "no address yields no endpoint",
			address:  "",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := map[string]interface{}{}
			if tt.address != "" {
				status["address"] = map[string]interface{}{"url": tt.address}
			}
			item := testInferenceService("my-ns", "my-model", tt.annotations, status)

			endpoint, ok := servingEndpoint(&item)
			assert.Equal(t, tt.expectOK, ok)
			assert.Equal(t, tt.expectedEndpoint, endpoint)
		})
	}
}

func TestNewServingModelFromLLMInferenceService(t *testing.T) {
	item := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.kserve.io/v1alpha1",
			"kind":       "LLMInferenceService",
			"metadata": map[string]interface{}{
				"namespace": "my-ns",
				"name":      "my-llm",
			},
			"spec": map[string]interface{}{
				"model": map[string]interface{}{"name": "Qwen/Qwen3-VL-30B-A3B-Instruct"},
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{"url": "https://my-llm.my-ns.svc.cluster.local:8000"},
				"conditions": []interface{}{
					map[string]interface{}{"type": "Ready", "status": "True"},
				},
			},
		},
	}
	item.SetAnnotations(map[string]string{"security.opendatahub.io/enable-auth": "true"})

	model, ok := newServingModelFromLLMInferenceService(&item)
	require.True(t, ok)
	assert.Equal(t, "Qwen/Qwen3-VL-30B-A3B-Instruct", model.Name)
	assert.Equal(t, "my-llm", model.ResourceName)
	assert.Equal(t, "LLMInferenceService", model.Kind)
	assert.Equal(t, "https://my-llm.my-ns.svc.cluster.local:8000/v1", model.Endpoint)
	assert.True(t, model.Ready)
}

func TestNewServingModelFromLLMInferenceServiceDefaultName(t *testing.T) {
	item := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.kserve.io/v1alpha1",
			"kind":       "LLMInferenceService",
			"metadata": map[string]interface{}{
				"namespace": "my-ns",
				"name":      "my-llm",
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{"url": "http://my-llm.my-ns.svc.cluster.local:8080"},
			},
		},
	}
	// spec.model.name absent: resource name is the served name.
	model, ok := newServingModelFromLLMInferenceService(&item)
	require.True(t, ok)
	assert.Equal(t, "my-llm", model.Name)
	assert.False(t, model.Ready)
}

func TestGetServingModels(t *testing.T) {
	isvc := testInferenceService("my-ns", "frauddetection", nil, map[string]interface{}{
		"address": map[string]interface{}{"url": "http://frauddetection-predictor.my-ns.svc.cluster.local:8080"},
		"conditions": []interface{}{
			map[string]interface{}{"type": "Ready", "status": "True"},
		},
	})
	client := &fakeGuardrailsClient{
		inferenceServices:    []unstructured.Unstructured{isvc},
		llmInferenceServices: []unstructured.Unstructured{},
	}

	repo := NewGuardrailsRepository()
	models, err := repo.GetServingModels(client, context.Background(), "my-ns")
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "frauddetection", models[0].Name)
	assert.Equal(t, "http://frauddetection-predictor.my-ns.svc.cluster.local:8080/v1", models[0].Endpoint)
	assert.True(t, models[0].Ready)
}

func TestGetServingModelsSkipsForbiddenResources(t *testing.T) {
	client := &fakeGuardrailsClient{
		forbidInferenceList: true,
	}

	repo := NewGuardrailsRepository()
	models, err := repo.GetServingModels(client, context.Background(), "my-ns")
	require.NoError(t, err)
	assert.Empty(t, models)
}

func TestGetServingModelsSkipsResourcesWithoutEndpoint(t *testing.T) {
	isvc := testInferenceService("my-ns", "not-ready", nil, map[string]interface{}{})
	client := &fakeGuardrailsClient{
		inferenceServices: []unstructured.Unstructured{isvc},
	}

	repo := NewGuardrailsRepository()
	models, err := repo.GetServingModels(client, context.Background(), "my-ns")
	require.NoError(t, err)
	assert.Empty(t, models)
}
