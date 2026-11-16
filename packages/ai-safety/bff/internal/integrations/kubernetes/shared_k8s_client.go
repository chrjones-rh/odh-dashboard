package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/opendatahub-io/ai-safety/bff/internal/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// nemoGuardrailsGVR is the GroupVersionResource of the NemoGuardrails CR
// managed by the trustyai-service-operator.
var nemoGuardrailsGVR = schema.GroupVersionResource{
	Group:    constants.NemoGuardrailsAPIGroup,
	Version:  constants.NemoGuardrailsAPIVersion,
	Resource: constants.NemoGuardrailsPlural,
}

// inferenceServiceGVR is the GroupVersionResource of the KServe InferenceService.
var inferenceServiceGVR = schema.GroupVersionResource{
	Group:    constants.KServeAPIGroup,
	Version:  constants.InferenceServiceAPIVersion,
	Resource: constants.InferenceServicePlural,
}

// llmInferenceServiceGVR is the GroupVersionResource of the KServe LLMInferenceService.
var llmInferenceServiceGVR = schema.GroupVersionResource{
	Group:    constants.KServeAPIGroup,
	Version:  constants.LLMInferenceServiceAPIVersion,
	Resource: constants.LLMInferenceServicePlural,
}

type SharedClientLogic struct {
	Client kubernetes.Interface
	Logger *slog.Logger
	Token  BearerToken
	// DynamicClient is used for custom resource access (NemoGuardrails).
	// It may be nil in mocked environments that only exercise core resources;
	// custom resource methods return an error in that case.
	DynamicClient dynamic.Interface
}

// Service discovery helpers removed for minimal starter footprint.

func (kc *SharedClientLogic) BearerToken() (string, error) { return kc.Token.Raw(), nil }

func (kc *SharedClientLogic) GetGroups(ctx context.Context) ([]string, error) { return []string{}, nil }

func (kc *SharedClientLogic) ListConfigMapsByLabel(ctx context.Context, namespace string, labelSelector string) ([]corev1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	list, err := kc.Client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps in namespace %s: %w", namespace, err)
	}
	return list.Items, nil
}

func (kc *SharedClientLogic) GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	configMap, err := kc.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s in namespace %s: %w", name, namespace, err)
	}
	return configMap, nil
}

func (kc *SharedClientLogic) ListNemoGuardrails(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	if kc.DynamicClient == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	list, err := kc.DynamicClient.Resource(nemoGuardrailsGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list NemoGuardrails in namespace %s: %w", namespace, err)
	}
	return list.Items, nil
}

func (kc *SharedClientLogic) GetNemoGuardrails(ctx context.Context, namespace string, name string) (*unstructured.Unstructured, error) {
	if kc.DynamicClient == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	item, err := kc.DynamicClient.Resource(nemoGuardrailsGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get NemoGuardrails %s in namespace %s: %w", name, namespace, err)
	}
	return item, nil
}

func (kc *SharedClientLogic) CreateNemoGuardrails(ctx context.Context, namespace string, cr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if kc.DynamicClient == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	created, err := kc.DynamicClient.Resource(nemoGuardrailsGVR).Namespace(namespace).Create(ctx, cr, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create NemoGuardrails %s in namespace %s: %w", cr.GetName(), namespace, err)
	}
	return created, nil
}

func (kc *SharedClientLogic) CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	created, err := kc.Client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create configmap %s in namespace %s: %w", configMap.Name, namespace, err)
	}
	return created, nil
}

func (kc *SharedClientLogic) DeleteNemoGuardrails(ctx context.Context, namespace string, name string) error {
	if kc.DynamicClient == nil {
		return fmt.Errorf("dynamic kubernetes client is not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := kc.DynamicClient.Resource(nemoGuardrailsGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete NemoGuardrails %s in namespace %s: %w", name, namespace, err)
	}
	return nil
}

func (kc *SharedClientLogic) DeleteConfigMap(ctx context.Context, namespace string, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := kc.Client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete configmap %s in namespace %s: %w", name, namespace, err)
	}
	return nil
}

// listDynamic lists an arbitrary namespaced resource via the dynamic client.
func (kc *SharedClientLogic) listDynamic(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]unstructured.Unstructured, error) {
	if kc.DynamicClient == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	list, err := kc.DynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s in namespace %s: %w", gvr.Resource, namespace, err)
	}
	return list.Items, nil
}

// ListInferenceServices lists KServe InferenceServices in the namespace.
func (kc *SharedClientLogic) ListInferenceServices(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	return kc.listDynamic(ctx, inferenceServiceGVR, namespace)
}

// ListLLMInferenceServices lists KServe LLMInferenceServices in the namespace.
func (kc *SharedClientLogic) ListLLMInferenceServices(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	return kc.listDynamic(ctx, llmInferenceServiceGVR, namespace)
}

// canCreateGuardrailDeployment performs the authorization reviews shared by
// both client flavors; the review itself is supplied by the concrete client
// (SubjectAccessReview for internal, SelfSubjectAccessReview for token).
func (kc *SharedClientLogic) canCreateResources(ctx context.Context, namespace string, review func(verb, group, resource string) (bool, error)) (bool, error) {
	allowed, err := review("create", "", "configmaps")
	if err != nil || !allowed {
		return false, err
	}
	return review("create", constants.NemoGuardrailsAPIGroup, constants.NemoGuardrailsPlural)
}
