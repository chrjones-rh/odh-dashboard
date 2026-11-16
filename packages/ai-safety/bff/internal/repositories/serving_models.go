package repositories

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	k8s "github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/models"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetServingModels lists the models served in the namespace by KServe
// InferenceServices and LLMInferenceServices, with the in-cluster
// OpenAI-compatible endpoint each one can be reached at. Resource types the
// user may not list are skipped: the endpoint returns whatever the user is
// allowed to see.
func (r *GuardrailsRepository) GetServingModels(
	client k8s.KubernetesClientInterface,
	ctx context.Context,
	namespace string,
) ([]models.ServingModel, error) {
	result := make([]models.ServingModel, 0)

	inferenceServices, err := client.ListInferenceServices(ctx, namespace)
	if err != nil {
		if !apierrors.IsForbidden(err) {
			return nil, fmt.Errorf("error fetching InferenceServices: %w", err)
		}
		inferenceServices = nil
	}
	for i := range inferenceServices {
		if model, ok := newServingModelFromInferenceService(&inferenceServices[i]); ok {
			result = append(result, model)
		}
	}

	llmInferenceServices, err := client.ListLLMInferenceServices(ctx, namespace)
	if err != nil {
		if !apierrors.IsForbidden(err) {
			return nil, fmt.Errorf("error fetching LLMInferenceServices: %w", err)
		}
		llmInferenceServices = nil
	}
	for i := range llmInferenceServices {
		if model, ok := newServingModelFromLLMInferenceService(&llmInferenceServices[i]); ok {
			result = append(result, model)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// newServingModelFromInferenceService builds a ServingModel from a KServe
// InferenceService. The served model name is the resource name.
func newServingModelFromInferenceService(item *unstructured.Unstructured) (models.ServingModel, bool) {
	model := models.ServingModel{
		Name:         item.GetName(),
		ResourceName: item.GetName(),
		Kind:         "InferenceService",
		Namespace:    item.GetNamespace(),
		Ready:        hasReadyCondition(item),
	}
	endpoint, ok := servingEndpoint(item)
	model.Endpoint = endpoint
	// A model without a resolvable in-cluster endpoint cannot be wired.
	return model, ok
}

// newServingModelFromLLMInferenceService builds a ServingModel from a KServe
// LLMInferenceService. The served model name is spec.model.name when set,
// matching the name the model server accepts in requests.
func newServingModelFromLLMInferenceService(item *unstructured.Unstructured) (models.ServingModel, bool) {
	modelName, _, _ := unstructured.NestedString(item.Object, "spec", "model", "name")
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = item.GetName()
	}
	model := models.ServingModel{
		Name:         modelName,
		ResourceName: item.GetName(),
		Kind:         "LLMInferenceService",
		Namespace:    item.GetNamespace(),
		Ready:        hasReadyCondition(item),
	}
	endpoint, ok := servingEndpoint(item)
	model.Endpoint = endpoint
	return model, ok
}

// servingEndpoint derives the in-cluster OpenAI-compatible base URL for a
// serving resource:
//   - the internal address (status.address.url), never the external route URL,
//   - https only when the resource terminates TLS itself (the enable-auth
//     annotation), http otherwise,
//   - a /v1 suffix, which OpenAI-compatible servers serve chat completions under.
//
// Note: unlike the model-serving stack's fuller derivation, headless predictor
// services whose address URL carries no port are not special-cased here; those
// resources are surfaced without an endpoint and fall back to manual entry.
func servingEndpoint(item *unstructured.Unstructured) (string, bool) {
	address, found, _ := unstructured.NestedString(item.Object, "status", "address", "url")
	if !found || strings.TrimSpace(address) == "" {
		return "", false
	}

	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	if item.GetAnnotations()["security.opendatahub.io/enable-auth"] != "true" {
		// The predictor does not terminate TLS; force plain http.
		parsed.Scheme = "http"
	}

	return ensureV1Suffix(parsed.String()), true
}

// ensureV1Suffix returns the URL with a /v1 path suffix, stripping trailing
// slashes first to avoid double slashes.
func ensureV1Suffix(endpoint string) string {
	endpoint = strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(endpoint, "/v1") {
		return endpoint
	}
	return endpoint + "/v1"
}

// hasReadyCondition reports whether the resource reports a Ready condition
// with status True.
func hasReadyCondition(item *unstructured.Unstructured) bool {
	conditions, found, err := unstructured.NestedSlice(item.Object, "status", "conditions")
	if !found || err != nil {
		return false
	}
	for _, raw := range conditions {
		condition, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(condition["type"]) == "Ready" && stringValue(condition["status"]) == "True" {
			return true
		}
	}
	return false
}
