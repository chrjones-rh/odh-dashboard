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

	// DashboardManagedLabel marks ConfigMaps created by the dashboard (for
	// example the per-deployment template copies); the delete flow only
	// removes ConfigMaps carrying it.
	DashboardManagedLabel = "opendatahub.io/dashboard"

	// DeploymentConfigLabel marks the per-deployment ConfigMap copies the
	// dashboard creates when deploying a template. They carry the guardrails
	// config label (the running guardrail needs it) but are runtime
	// configuration, not catalog templates, and are excluded from template
	// listings.
	DeploymentConfigLabel = "opendatahub.io/deployment-config"

	// TemplateSourceAnnotation records the template a dashboard-created
	// ConfigMap was copied from, as "<namespace>/<name>".
	TemplateSourceAnnotation = "opendatahub.io/template-source"

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

// KnownRailFlows lists the rail flow names the NeMo Guardrails server
// recognizes without a template-provided rails.co (library flows plus the
// internal flows the validator also accepts). Enumerated from a live server
// image by instantiating LLMRails and reading the resolved flow set; flow
// entries may trail parameters (for example
// `hf classifier check input $classifier="prompt-injection"`), which are not
// part of the name. Unknown flows fail guardrail config loading at deploy or
// test time with a generic error, so templates are validated at save time
// against this list instead.
var KnownRailFlows = []string{
	"alignscore check facts",
	"content safety check input",
	"content safety check output",
	"context bloat detection on input",
	"context bloat detection on retrieval",
	"detect sensitive data on input",
	"detect sensitive data on output",
	"detect sensitive data on retrieval",
	"f5 guardrails scan input",
	"f5 guardrails scan output",
	"generate bot message",
	"generate next step",
	"generate user intent",
	"gliner detect pii on input",
	"gliner detect pii on output",
	"gliner detect pii on retrieval",
	"gliner mask pii on input",
	"gliner mask pii on output",
	"gliner mask pii on retrieval",
	"guardrailsai check input",
	"guardrailsai check output",
	"hallucination warning",
	"hf classifier check input",
	"hf classifier check output",
	"hf classifier check retrieval",
	"injection detection",
	"llama guard check input",
	"llama guard check output",
	"mask blocked user input",
	"mask sensitive data on input",
	"mask sensitive data on output",
	"mask sensitive data on retrieval",
	"polygraf detect pii on input",
	"polygraf detect pii on output",
	"polygraf detect pii on retrieval",
	"polygraf mask pii on input",
	"polygraf mask pii on output",
	"polygraf mask pii on retrieval",
	"process bot message",
	"process bot tool call",
	"process user input",
	"process user tool messages",
	"regex check input",
	"regex check output",
	"regex check retrieval",
	"run dialog rails",
	"run input rails",
	"run output rails",
	"run retrieval rails",
	"run tool input rails",
	"run tool output rails",
	"self check facts",
	"self check hallucination",
	"self check input",
	"self check output",
	"topic safety check input",
}

// KnownRailsConfigKeys lists the top-level keys the server's rail actions
// read from the rails.config section. The server silently ignores unknown
// keys, which turns a typo into a rail running without its configuration —
// templates are validated at save time against this list instead. Derived
// from the rail action libraries shipped in the server image.
var KnownRailsConfigKeys = []string{
	"content_safety",
	"context_bloat_detection",
	"f5",
	"fact_checking",
	"gliner",
	"guardrails_ai",
	"hf_classifier",
	"injection_detection",
	"jailbreak_detection",
	"llama_guard",
	"polygraf",
	"regex_detection",
	"sensitive_data_detection",
	"topic_safety",
}
