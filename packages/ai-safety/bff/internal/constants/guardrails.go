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
)
