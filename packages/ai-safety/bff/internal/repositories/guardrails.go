package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/opendatahub-io/ai-safety/bff/internal/constants"
	k8s "github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/models"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// operatorTemplatePartOfLabel is the label key/value marking ConfigMaps managed
// by the trustyai-service-operator.
const operatorTemplatePartOfLabel = "platform.opendatahub.io/part-of"
const operatorTemplatePartOfValue = "trustyai-module-operator"

// Annotations set by the trustyai-service-operator on guardrail template
// ConfigMaps describing their purpose and behavior.
const (
	descriptionAnnotation = "guardrail-description"
	actionsAnnotation     = "guardrail-actions"
	locationsAnnotation   = "guardrail-locations"
	tagsAnnotation        = "guardrail-tags"
)

type GuardrailsRepository struct{}

func NewGuardrailsRepository() *GuardrailsRepository {
	return &GuardrailsRepository{}
}

// GetTemplates lists the guardrail configuration templates: the platform
// templates plus, when a user namespace is given, the user-created template
// ConfigMaps in that namespace. Each template is tagged with its source.
func (r *GuardrailsRepository) GetTemplates(
	internalClient k8s.KubernetesClientInterface,
	client k8s.KubernetesClientInterface,
	ctx context.Context,
	templateNamespace string,
	userNamespace string,
) ([]models.GuardrailTemplate, error) {
	platformConfigMaps, err := internalClient.ListConfigMapsByLabel(ctx, templateNamespace, constants.GuardrailsTemplateLabel)
	if err != nil {
		return nil, fmt.Errorf("error fetching platform guardrail templates: %w", err)
	}

	var userConfigMaps []corev1.ConfigMap
	if userNamespace != "" {
		userConfigMaps, err = client.ListConfigMapsByLabel(ctx, userNamespace, constants.GuardrailsTemplateLabel)
		if err != nil {
			if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
				// The user cannot list ConfigMaps in this namespace; the
				// platform catalog remains available.
				userConfigMaps = nil
			} else {
				return nil, fmt.Errorf("error fetching guardrail templates in namespace %s: %w", userNamespace, err)
			}
		}
	}

	templates := make([]models.GuardrailTemplate, 0, len(platformConfigMaps)+len(userConfigMaps))
	for i := range platformConfigMaps {
		template := newTemplateModel(&platformConfigMaps[i])
		template.Source = "platform"
		templates = append(templates, template)
	}
	for i := range userConfigMaps {
		template := newTemplateModel(&userConfigMaps[i])
		template.Source = "user"
		templates = append(templates, template)
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates, nil
}

// GetTemplate returns a single guardrail configuration template with its full
// file contents.
func (r *GuardrailsRepository) GetTemplate(client k8s.KubernetesClientInterface, ctx context.Context, namespace, name string) (*models.GuardrailTemplateDetail, error) {
	configMap, err := client.GetConfigMap(ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("error fetching guardrail template %s: %w", name, err)
	}

	template := newTemplateModel(configMap)
	detail := &models.GuardrailTemplateDetail{
		GuardrailTemplate: template,
		Files:             make([]models.GuardrailConfigFile, 0, len(configMap.Data)),
	}
	for _, key := range template.Keys {
		detail.Files = append(detail.Files, models.GuardrailConfigFile{
			Key:     key,
			Content: configMap.Data[key],
		})
	}
	return detail, nil
}

// dashboardManagedLabel marks ConfigMaps created by the dashboard (for example
// the guardrail template copies), used to decide cleanup ownership on delete.
const dashboardManagedLabel = "opendatahub.io/dashboard"

// DeleteDeployment deletes a NemoGuardrails deployment and the ConfigMap
// copies the dashboard created for it. ConfigMaps that are not
// dashboard-managed (for example user-created templates referenced by a
// hand-written CR) are left in place.
func (r *GuardrailsRepository) DeleteDeployment(
	client k8s.KubernetesClientInterface,
	ctx context.Context,
	namespace, name string,
) error {
	item, err := client.GetNemoGuardrails(ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("error fetching NemoGuardrails deployment %s: %w", name, err)
	}

	if err := client.DeleteNemoGuardrails(ctx, namespace, name); err != nil {
		if apierrors.IsNotFound(err) {
			return err
		}
		return err
	}

	// Best-effort cleanup of the dashboard-created ConfigMap copies. Missing
	// or forbidden ConfigMaps are skipped: a user without configmap delete
	// permission still gets the deployment removed, and user-managed
	// ConfigMaps are never touched.
	for _, configMapName := range referencedConfigMapNames(item) {
		configMap, err := client.GetConfigMap(ctx, namespace, configMapName)
		if err != nil {
			continue
		}
		if configMap.Labels[dashboardManagedLabel] != "true" {
			continue
		}
		if err := client.DeleteConfigMap(ctx, namespace, configMapName); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("error deleting configmap %s: %w", configMapName, err)
		}
	}

	return nil
}

// resolveTemplateClient returns the client and namespace to read the
// referenced template from: user templates live in their own namespace and are
// read with the requesting user's client; platform templates are read with the
// service account client.
func resolveTemplateClient(
	client k8s.KubernetesClientInterface,
	internalClient k8s.KubernetesClientInterface,
	platformNamespace string,
	templateNamespace string,
) (k8s.KubernetesClientInterface, string) {
	if templateNamespace != "" && templateNamespace != platformNamespace {
		return client, templateNamespace
	}
	return internalClient, platformNamespace
}

// GetDeployments lists the NemoGuardrails deployments in the namespace,
// joined with the ConfigMaps their nemoConfigs reference.
func (r *GuardrailsRepository) GetDeployments(client k8s.KubernetesClientInterface, ctx context.Context, namespace string) ([]models.GuardrailDeployment, error) {
	items, err := client.ListNemoGuardrails(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("error fetching NemoGuardrails deployments: %w", err)
	}

	// Collect all referenced ConfigMap names, then resolve each once.
	referenced := map[string]bool{}
	for i := range items {
		for _, name := range referencedConfigMapNames(&items[i]) {
			referenced[name] = true
		}
	}
	configSources := r.resolveConfigSources(client, ctx, namespace, referenced)

	deployments := make([]models.GuardrailDeployment, 0, len(items))
	for i := range items {
		deployments = append(deployments, newDeploymentModel(&items[i], configSources))
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Name < deployments[j].Name
	})
	return deployments, nil
}

// GetDeployment returns a single NemoGuardrails deployment with its full specification.
func (r *GuardrailsRepository) GetDeployment(client k8s.KubernetesClientInterface, ctx context.Context, namespace, name string) (*models.GuardrailDeploymentDetail, error) {
	item, err := client.GetNemoGuardrails(ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("error fetching NemoGuardrails deployment %s: %w", name, err)
	}

	referenced := map[string]bool{}
	for _, configMapName := range referencedConfigMapNames(item) {
		referenced[configMapName] = true
	}
	configSources := r.resolveConfigSources(client, ctx, namespace, referenced)

	detail := &models.GuardrailDeploymentDetail{
		GuardrailDeployment: newDeploymentModel(item, configSources),
		ExposeRoute:         nestedBool(item, "spec", "exposeRoute"),
		NemoConfigs:         nestedNemoConfigs(item),
		EnvVarNames:         nestedEnvVarNames(item),
	}
	for _, source := range detail.ConfigSources {
		if source.MainModel != "" {
			detail.MainModel = source.MainModel
			break
		}
	}
	return detail, nil
}

// deploymentNamePattern matches DNS-1123 lowercase names up to 63 characters.
var deploymentNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateDeploymentName checks that the deployment name is a valid
// DNS-1123 lowercase name suitable for both the CR and the derived ConfigMap.
func ValidateDeploymentName(name string) error {
	if !deploymentNamePattern.MatchString(name) {
		return fmt.Errorf("invalid deployment name %q: must be lowercase alphanumeric with hyphens, start and end with alphanumeric, and be at most 63 characters", name)
	}
	return nil
}

// CreateDeploymentFromTemplate deploys a guardrail template into a namespace:
// it copies the template ConfigMap into the target namespace and creates a
// NemoGuardrails custom resource referencing the copy. The template is read
// with the internal (service account) client; the copy and the CR are created
// with the requesting user's client. When the request carries model wiring,
// the copied config.yaml gains a main model entry and the CR gains the
// MAIN_MODEL_BASE_URL and MAIN_MODEL_ENGINE environment variables, so
// requests work with or without a model field.
func (r *GuardrailsRepository) CreateDeploymentFromTemplate(
	client k8s.KubernetesClientInterface,
	internalClient k8s.KubernetesClientInterface,
	ctx context.Context,
	templateNamespace string,
	request models.GuardrailDeploymentCreateRequest,
) (*models.GuardrailDeployment, error) {
	if err := ValidateDeploymentName(request.Name); err != nil {
		return nil, err
	}

	var wiring *models.GuardrailModelWiring
	if request.Model != nil {
		validated, err := ValidateModelWiring(request.Model)
		if err != nil {
			return nil, err
		}
		wiring = validated
	}

	templateClient, templateNamespace := resolveTemplateClient(
		client, internalClient, templateNamespace, request.TemplateNamespace)
	template, err := templateClient.GetConfigMap(ctx, templateNamespace, request.TemplateName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("error fetching guardrail template %s: %w", request.TemplateName, err)
	}

	// Copy the template ConfigMap into the target namespace. nemoConfigs
	// reference ConfigMaps in the CR's own namespace, so the copy is required.
	configMapName := fmt.Sprintf("guardrail-%s", request.Name)
	configMapCopy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
			Labels: map[string]string{
				"nemo-guardrails-config":   "true",
				"opendatahub.io/dashboard": "true",
			},
		},
		Data: template.Data,
	}

	hasModelSpec := false
	mainModel := ""
	if wiring != nil {
		configYAML, ok := configMapCopy.Data[constants.GuardrailsConfigYAMLKey]
		if !ok {
			return nil, fmt.Errorf("template %s has no %s to wire the model into", request.TemplateName, constants.GuardrailsConfigYAMLKey)
		}
		modified, err := injectMainModel(configYAML, wiring)
		if err != nil {
			return nil, fmt.Errorf("error wiring model into template config: %w", err)
		}
		configMapCopy.Data[constants.GuardrailsConfigYAMLKey] = modified
		hasModelSpec = true
		mainModel = wiring.Name
	}

	if _, err := client.CreateConfigMap(ctx, request.Namespace, configMapCopy); err != nil {
		return nil, err
	}

	spec := map[string]interface{}{
		"replicas":    int64(1),
		"exposeRoute": true,
		"nemoConfigs": []interface{}{
			map[string]interface{}{
				"name":       request.Name,
				"configMaps": []interface{}{configMapName},
				"default":    true,
			},
		},
	}
	if wiring != nil {
		// The server reads these when a request carries a model field.
		spec["env"] = []interface{}{
			map[string]interface{}{
				"name":  constants.MainModelBaseURLEnv,
				"value": wiring.Endpoint,
			},
			map[string]interface{}{
				"name":  constants.MainModelEngineEnv,
				"value": constants.MainModelEngineDefault,
			},
		}
	}

	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", constants.NemoGuardrailsAPIGroup, constants.NemoGuardrailsAPIVersion),
			"kind":       "NemoGuardrails",
			"metadata": map[string]interface{}{
				"name": request.Name,
				"labels": map[string]interface{}{
					"opendatahub.io/dashboard": "true",
				},
				"annotations": map[string]interface{}{
					"security.opendatahub.io/enable-auth": "true",
				},
			},
			"spec": spec,
		},
	}
	createdCR, err := client.CreateNemoGuardrails(ctx, request.Namespace, cr)
	if err != nil {
		return nil, err
	}

	configSources := map[string]models.GuardrailConfigSource{
		configMapName: {
			Name:               configMapName,
			Found:              true,
			Keys:               sortedKeys(template.Data),
			HasModelSpec:       hasModelSpec,
			IsOperatorTemplate: false,
			MainModel:          mainModel,
		},
	}
	deployment := newDeploymentModel(createdCR, configSources)
	return &deployment, nil
}

// ValidateModelWiring checks the model wiring fields and normalizes the
// endpoint: it must be a parseable http(s) URL with a host and no embedded
// credentials, and it gains a /v1 suffix when missing (the OpenAI-compatible
// path chat completions are served under).
func ValidateModelWiring(wiring *models.GuardrailModelWiring) (*models.GuardrailModelWiring, error) {
	name := strings.TrimSpace(wiring.Name)
	if name == "" {
		return nil, fmt.Errorf("model name is required when model wiring is provided")
	}
	if len(name) > 253 {
		return nil, fmt.Errorf("model name must be at most 253 characters")
	}

	endpoint := strings.TrimSpace(wiring.Endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("model endpoint %q is not a valid URL", wiring.Endpoint)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("model endpoint scheme must be http or https, got %q", parsed.Scheme)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("model endpoint must not contain credentials")
	}

	return &models.GuardrailModelWiring{
		Name:     name,
		Endpoint: ensureV1Suffix(parsed.String()),
	}, nil
}

// injectMainModel writes the main model entry into a NeMo Guardrails
// config.yaml: an existing model with type "main" is replaced (merging its
// parameters, mirroring the server's own update logic), otherwise the entry
// is appended. Other models and configuration keys are preserved.
func injectMainModel(configYAML string, wiring *models.GuardrailModelWiring) (string, error) {
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		return "", fmt.Errorf("cannot parse config.yaml: %w", err)
	}
	if parsed == nil {
		parsed = map[string]interface{}{}
	}

	modelsRaw, _ := parsed["models"].([]interface{})
	entry := map[string]interface{}{
		"type":       "main",
		"engine":     constants.MainModelEngineDefault,
		"model":      wiring.Name,
		"parameters": map[string]interface{}{"base_url": wiring.Endpoint},
	}

	replaced := false
	for i, modelRaw := range modelsRaw {
		model, ok := modelRaw.(map[string]interface{})
		if !ok || stringValue(model["type"]) != "main" {
			continue
		}
		parameters, _ := model["parameters"].(map[string]interface{})
		merged := make(map[string]interface{}, len(parameters)+1)
		for key, value := range parameters {
			merged[key] = value
		}
		merged["base_url"] = wiring.Endpoint
		model["engine"] = constants.MainModelEngineDefault
		model["model"] = wiring.Name
		model["parameters"] = merged
		modelsRaw[i] = model
		replaced = true
		break
	}
	if !replaced {
		modelsRaw = append(modelsRaw, entry)
	}
	parsed["models"] = modelsRaw

	modified, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("cannot serialize config.yaml: %w", err)
	}
	return string(modified), nil
}

// resolveConfigSources fetches each referenced ConfigMap once and builds the
// config source descriptors. Missing ConfigMaps are reported as not found
// rather than failing the whole listing.
func (r *GuardrailsRepository) resolveConfigSources(client k8s.KubernetesClientInterface, ctx context.Context, namespace string, names map[string]bool) map[string]models.GuardrailConfigSource {
	sources := make(map[string]models.GuardrailConfigSource, len(names))
	for name := range names {
		source := models.GuardrailConfigSource{Name: name, Found: false}
		configMap, err := client.GetConfigMap(ctx, namespace, name)
		if err == nil {
			source.Found = true
			source.Keys = sortedKeys(configMap.Data)
			source.IsOperatorTemplate = configMap.Labels[operatorTemplatePartOfLabel] == operatorTemplatePartOfValue
			if configYAML, ok := configMap.Data[constants.GuardrailsConfigYAMLKey]; ok {
				summary := parseConfigSummary(configYAML)
				source.HasModelSpec = summary.ModelCount > 0
				source.MainModel = mainModelNameFromConfig(configYAML)
			}
		}
		sources[name] = source
	}
	return sources
}

// mainModelNameFromConfig extracts the model name of the first type "main"
// entry in a NeMo Guardrails config.yaml. Missing or unparsable configs yield
// an empty string.
func mainModelNameFromConfig(configYAML string) string {
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		return ""
	}
	modelsRaw, ok := parsed["models"].([]interface{})
	if !ok {
		return ""
	}
	for _, modelRaw := range modelsRaw {
		model, ok := modelRaw.(map[string]interface{})
		if !ok || stringValue(model["type"]) != "main" {
			continue
		}
		if name := strings.TrimSpace(stringValue(model["model"])); name != "" {
			return name
		}
	}
	return ""
}

func newTemplateModel(configMap *corev1.ConfigMap) models.GuardrailTemplate {
	template := models.GuardrailTemplate{
		Name:        configMap.Name,
		Namespace:   configMap.Namespace,
		Keys:        sortedKeys(configMap.Data),
		CreatedAt:   configMap.CreationTimestamp.UTC().Format(time.RFC3339),
		Summary:     models.GuardrailConfigSummary{},
		Description: strings.TrimSpace(configMap.Annotations[descriptionAnnotation]),
		Actions:     parseAnnotationList(configMap.Annotations[actionsAnnotation]),
		Locations:   parseAnnotationList(configMap.Annotations[locationsAnnotation]),
		Tags:        parseAnnotationList(configMap.Annotations[tagsAnnotation]),
	}
	if configYAML, ok := configMap.Data[constants.GuardrailsConfigYAMLKey]; ok {
		template.Summary = parseConfigSummary(configYAML)
	}
	return template
}

// parseAnnotationList parses a JSON string array annotation (for example
// guardrail-actions). Missing or malformed values yield nil.
func parseAnnotationList(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return nil
	}
	return result
}

func newDeploymentModel(item *unstructured.Unstructured, configSources map[string]models.GuardrailConfigSource) models.GuardrailDeployment {
	phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
	endpoint, _, _ := unstructured.NestedString(item.Object, "status", "endpoint")

	deployment := models.GuardrailDeployment{
		Name:          item.GetName(),
		Namespace:     item.GetNamespace(),
		Phase:         phase,
		Ready:         isDeploymentReady(item),
		ModelWired:    hasMainModelBaseURL(item),
		Endpoint:      endpoint,
		CreatedAt:     item.GetCreationTimestamp().UTC().Format(time.RFC3339),
		Conditions:    nestedConditions(item),
		ConfigSources: []models.GuardrailConfigSource{},
	}

	if replicas, ok, _ := unstructured.NestedInt64(item.Object, "spec", "replicas"); ok {
		deployment.Replicas = &replicas
	}

	names := referencedConfigMapNames(item)
	for _, name := range names {
		source := configSources[name]
		deployment.ConfigSources = append(deployment.ConfigSources, source)
		// A config-side main model also makes the deployment testable: the
		// server accepts requests without a model field through it.
		if source.MainModel != "" {
			deployment.ModelWired = true
		}
	}

	return deployment
}

// hasMainModelBaseURL reports whether the CR spec carries the
// MAIN_MODEL_BASE_URL environment variable (name only; values are never read).
func hasMainModelBaseURL(item *unstructured.Unstructured) bool {
	for _, name := range nestedEnvVarNames(item) {
		if name == constants.MainModelBaseURLEnv {
			return true
		}
	}
	return false
}

func isDeploymentReady(item *unstructured.Unstructured) bool {
	conditions, found, err := unstructured.NestedSlice(item.Object, "status", "conditions")
	if !found || err != nil {
		return false
	}
	for _, raw := range conditions {
		condition, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if conditionType, _ := condition["type"].(string); conditionType == "DeploymentReady" {
			status, _ := condition["status"].(string)
			return status == "True"
		}
	}
	return false
}

func nestedConditions(item *unstructured.Unstructured) []models.GuardrailCondition {
	conditions, found, err := unstructured.NestedSlice(item.Object, "status", "conditions")
	if !found || err != nil {
		return []models.GuardrailCondition{}
	}
	result := make([]models.GuardrailCondition, 0, len(conditions))
	for _, raw := range conditions {
		condition, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		conditionModel := models.GuardrailCondition{
			Type:               stringValue(condition["type"]),
			Status:             stringValue(condition["status"]),
			Reason:             stringValue(condition["reason"]),
			Message:            stringValue(condition["message"]),
			LastTransitionTime: stringValue(condition["lastTransitionTime"]),
		}
		if conditionModel.Type == "" && conditionModel.Status == "" {
			continue
		}
		result = append(result, conditionModel)
	}
	return result
}

func nestedNemoConfigs(item *unstructured.Unstructured) []models.GuardrailNemoConfig {
	nemoConfigs, found, err := unstructured.NestedSlice(item.Object, "spec", "nemoConfigs")
	if !found || err != nil {
		return nil
	}
	result := make([]models.GuardrailNemoConfig, 0, len(nemoConfigs))
	for _, raw := range nemoConfigs {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		configMaps, _, _ := unstructured.NestedStringSlice(entry, "configMaps")
		isDefault, _, _ := unstructured.NestedBool(entry, "default")
		name := stringValue(entry["name"])
		if name == "" {
			continue
		}
		result = append(result, models.GuardrailNemoConfig{
			Name:       name,
			ConfigMaps: configMaps,
			Default:    isDefault,
		})
	}
	return result
}

// nestedEnvVarNames extracts only the names of the env entries. Values are
// intentionally never surfaced by the BFF.
func nestedEnvVarNames(item *unstructured.Unstructured) []string {
	envEntries, found, err := unstructured.NestedSlice(item.Object, "spec", "env")
	if !found || err != nil {
		return nil
	}
	names := make([]string, 0, len(envEntries))
	for _, raw := range envEntries {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if name := stringValue(entry["name"]); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func referencedConfigMapNames(item *unstructured.Unstructured) []string {
	nemoConfigs, found, err := unstructured.NestedSlice(item.Object, "spec", "nemoConfigs")
	if !found || err != nil {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	for _, raw := range nemoConfigs {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		configMaps, _, _ := unstructured.NestedStringSlice(entry, "configMaps")
		for _, name := range configMaps {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

func nestedBool(item *unstructured.Unstructured, fields ...string) bool {
	value, found, err := unstructured.NestedBool(item.Object, fields...)
	return found && err == nil && value
}

func stringValue(value interface{}) string {
	result, _ := value.(string)
	return result
}

func sortedKeys(data map[string]string) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// parseConfigSummary performs a best-effort parse of a NeMo Guardrails
// config.yaml, extracting the model descriptors, rail flows, prompt count and
// passthrough flag. Unparsable configs yield an empty summary.
func parseConfigSummary(configYAML string) models.GuardrailConfigSummary {
	summary := models.GuardrailConfigSummary{
		Models:    []string{},
		RailFlows: []models.GuardrailRailFlow{},
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		return summary
	}

	summary.Passthrough, _ = parsed["passthrough"].(bool)

	if modelsRaw, ok := parsed["models"].([]interface{}); ok {
		for _, modelRaw := range modelsRaw {
			model, ok := modelRaw.(map[string]interface{})
			if !ok {
				continue
			}
			modelType := stringValue(model["type"])
			engine := stringValue(model["engine"])
			modelName := stringValue(model["model"])
			var descriptor string
			switch {
			case engine != "" && modelName != "":
				descriptor = fmt.Sprintf("%s/%s", engine, modelName)
			case modelName != "":
				descriptor = modelName
			case engine != "":
				descriptor = engine
			default:
				continue
			}
			if modelType != "" {
				descriptor = fmt.Sprintf("%s (%s)", descriptor, modelType)
			}
			summary.Models = append(summary.Models, descriptor)
		}
	}
	summary.ModelCount = len(summary.Models)

	if rails, ok := parsed["rails"].(map[string]interface{}); ok {
		for category, railsRaw := range rails {
			rail, ok := railsRaw.(map[string]interface{})
			if !ok {
				continue
			}
			flows, ok := rail["flows"].([]interface{})
			if !ok {
				continue
			}
			for _, flowRaw := range flows {
				flow, ok := flowRaw.(string)
				if !ok {
					continue
				}
				summary.RailFlows = append(summary.RailFlows, models.GuardrailRailFlow{
					Category: strings.TrimSpace(category),
					Flow:     strings.TrimSpace(flow),
				})
			}
		}
	}

	if prompts, ok := parsed["prompts"].([]interface{}); ok {
		summary.PromptCount = len(prompts)
	}

	return summary
}
