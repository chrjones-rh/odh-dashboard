package constants

const (
	// NemoGuardrailsAPIGroup is the API group of the NemoGuardrails custom resource
	// managed by the trustyai-service-operator.
	NemoGuardrailsAPIGroup = "trustyai.opendatahub.io"
	// NemoGuardrailsAPIVersion is the API version of the NemoGuardrails custom resource.
	NemoGuardrailsAPIVersion = "v1alpha1"
	// NemoGuardrailsPlural is the plural resource name of the NemoGuardrails custom resource.
	NemoGuardrailsPlural = "nemoguardrails"

	// GuardrailsTemplateLabel is the label selector matching guardrail configuration
	// template ConfigMaps, both operator-shipped templates and user namespace copies.
	GuardrailsTemplateLabel = "nemo-guardrails-config=true"

	// GuardrailsOperatorTemplateLabel is the label applied to ConfigMaps managed by the
	// trustyai-service-operator, used to distinguish operator templates from user copies.
	GuardrailsOperatorTemplateLabel = "platform.opendatahub.io/part-of=trustyai-module-operator"

	// DefaultGuardrailsTemplateNamespace is the default platform namespace holding the
	// guardrail template ConfigMaps.
	DefaultGuardrailsTemplateNamespace = "redhat-ods-applications"

	// GuardrailsConfigYAMLKey is the ConfigMap key holding the NeMo Guardrails config.yaml.
	GuardrailsConfigYAMLKey = "config.yaml"

	// KServeAPIGroup is the API group of the KServe serving resources.
	KServeAPIGroup = "serving.kserve.io"
	// InferenceServiceAPIVersion is the API version of the KServe InferenceService resource.
	InferenceServiceAPIVersion = "v1beta1"
	// InferenceServicePlural is the plural resource name of the KServe InferenceService.
	InferenceServicePlural = "inferenceservices"
	// LLMInferenceServiceAPIVersion is the API version of the KServe LLMInferenceService resource.
	LLMInferenceServiceAPIVersion = "v1alpha1"
	// LLMInferenceServicePlural is the plural resource name of the KServe LLMInferenceService.
	LLMInferenceServicePlural = "llminferenceservices"

	// MainModelBaseURLEnv is the environment variable the NeMo Guardrails server
	// reads the main model base URL from (used when a request carries a model field).
	MainModelBaseURLEnv = "MAIN_MODEL_BASE_URL"
	// MainModelEngineEnv is the environment variable naming the LLM engine the
	// NeMo Guardrails server uses for the main model.
	MainModelEngineEnv = "MAIN_MODEL_ENGINE"
	// MainModelEngineDefault is the engine used for OpenAI-compatible model endpoints.
	MainModelEngineDefault = "openai"

	// EnableAuthAnnotation marks KServe resources that terminate TLS themselves;
	// their in-cluster endpoints are served over https.
	EnableAuthAnnotation = "security.opendatahub.io/enable-auth"

	// ServiceCAConfigMapKey is the ConfigMap key holding the OpenShift service
	// serving CA bundle (PEM).
	ServiceCAConfigMapKey = "service-ca.crt"
	// ServiceCAConfigMapFallback is the namespace-synced ConfigMap holding the
	// service serving CA bundle, used when the operator-managed per-deployment
	// bundle is not present.
	ServiceCAConfigMapFallback = "openshift-service-ca.crt"
)
