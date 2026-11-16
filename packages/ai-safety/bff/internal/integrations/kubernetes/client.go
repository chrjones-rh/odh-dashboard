package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const ComponentLabelValue = "mod-arch"

// KubernetesClientInterface exposes the surface needed by the ai-safety BFF.
type KubernetesClientInterface interface {
	GetNamespaces(ctx context.Context, identity *RequestIdentity) ([]corev1.Namespace, error)
	IsClusterAdmin(identity *RequestIdentity) (bool, error)
	GetUser(identity *RequestIdentity) (string, error)

	// ListConfigMapsByLabel lists ConfigMaps in the namespace matching the label selector.
	ListConfigMapsByLabel(ctx context.Context, namespace string, labelSelector string) ([]corev1.ConfigMap, error)
	// GetConfigMap returns a single ConfigMap.
	GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error)
	// ListNemoGuardrails lists NemoGuardrails custom resources in the namespace.
	ListNemoGuardrails(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	// GetNemoGuardrails returns a single NemoGuardrails custom resource.
	GetNemoGuardrails(ctx context.Context, namespace string, name string) (*unstructured.Unstructured, error)
	// CreateNemoGuardrails creates a NemoGuardrails custom resource in the namespace.
	CreateNemoGuardrails(ctx context.Context, namespace string, cr *unstructured.Unstructured) (*unstructured.Unstructured, error)
	// CreateConfigMap creates a ConfigMap in the namespace.
	CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	// UpdateConfigMap updates a ConfigMap in the namespace.
	UpdateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	// DeleteNemoGuardrails deletes a NemoGuardrails custom resource in the namespace.
	DeleteNemoGuardrails(ctx context.Context, namespace string, name string) error
	// DeleteConfigMap deletes a ConfigMap in the namespace.
	DeleteConfigMap(ctx context.Context, namespace string, name string) error
	// ListInferenceServices lists KServe InferenceServices in the namespace.
	ListInferenceServices(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	// ListLLMInferenceServices lists KServe LLMInferenceServices in the namespace.
	ListLLMInferenceServices(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	// BearerToken returns the raw bearer token this client authenticates with.
	BearerToken() (string, error)
	// CanListNemoGuardrails reports whether the request identity may list
	// NemoGuardrails custom resources in the namespace.
	CanListNemoGuardrails(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanListInferenceServices reports whether the request identity may list
	// KServe InferenceServices in the namespace.
	CanListInferenceServices(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanListLLMInferenceServices reports whether the request identity may list
	// KServe LLMInferenceServices in the namespace.
	CanListLLMInferenceServices(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanListConfigMaps reports whether the request identity may list
	// ConfigMaps in the namespace.
	CanListConfigMaps(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanCreateConfigMaps reports whether the request identity may create
	// ConfigMaps in the namespace.
	CanCreateConfigMaps(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanUpdateConfigMaps reports whether the request identity may update
	// ConfigMaps in the namespace.
	CanUpdateConfigMaps(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanCreateGuardrailDeployment reports whether the request identity may
	// create both ConfigMaps and NemoGuardrails custom resources in the namespace.
	CanCreateGuardrailDeployment(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	// CanDeleteNemoGuardrails reports whether the request identity may delete
	// NemoGuardrails custom resources in the namespace.
	CanDeleteNemoGuardrails(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
}
