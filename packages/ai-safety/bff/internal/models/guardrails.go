package models

// GuardrailRailFlow is a single rail flow within a NeMo Guardrails config.
type GuardrailRailFlow struct {
	Category string `json:"category"`
	Flow     string `json:"flow"`
}

// GuardrailConfigSummary is a best-effort summary of a NeMo Guardrails config.yaml.
type GuardrailConfigSummary struct {
	ModelCount  int                 `json:"modelCount"`
	Models      []string            `json:"models"`
	RailFlows   []GuardrailRailFlow `json:"railFlows"`
	PromptCount int                 `json:"promptCount"`
	Passthrough bool                `json:"passthrough"`
}

// GuardrailTemplate is a guardrail configuration template. Templates either
// ship with the platform (Source "platform") or live in a user namespace
// (Source "user").
type GuardrailTemplate struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Keys        []string               `json:"keys"`
	CreatedAt   string                 `json:"createdAt"`
	Summary     GuardrailConfigSummary `json:"summary"`
	Description string                 `json:"description,omitempty"`
	Actions     []string               `json:"actions,omitempty"`
	Locations   []string               `json:"locations,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Source      string                 `json:"source"`
}

// GuardrailConfigFile is a single configuration file within a guardrail ConfigMap.
type GuardrailConfigFile struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

// GuardrailTemplateDetail is a guardrail configuration template with full file contents.
type GuardrailTemplateDetail struct {
	GuardrailTemplate
	Files []GuardrailConfigFile `json:"files"`
}

// GuardrailCondition is a status condition on a NemoGuardrails resource.
type GuardrailCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

// GuardrailConfigSource describes a ConfigMap referenced by a NemoGuardrails deployment.
type GuardrailConfigSource struct {
	Name               string   `json:"name"`
	Found              bool     `json:"found"`
	Keys               []string `json:"keys,omitempty"`
	HasModelSpec       bool     `json:"hasModelSpec,omitempty"`
	IsOperatorTemplate bool     `json:"isOperatorTemplate,omitempty"`
	MainModel          string   `json:"mainModel,omitempty"`
}

// GuardrailDeployment is a NemoGuardrails deployment in a namespace.
type GuardrailDeployment struct {
	Name          string                  `json:"name"`
	Namespace     string                  `json:"namespace"`
	Phase         string                  `json:"phase"`
	Ready         bool                    `json:"ready"`
	ModelWired    bool                    `json:"modelWired"`
	Endpoint      string                  `json:"endpoint,omitempty"`
	Replicas      *int64                  `json:"replicas,omitempty"`
	CreatedAt     string                  `json:"createdAt"`
	Conditions    []GuardrailCondition    `json:"conditions"`
	ConfigSources []GuardrailConfigSource `json:"configSources"`
}

// GuardrailNemoConfig is a single nemoConfigs entry of a NemoGuardrails resource.
type GuardrailNemoConfig struct {
	Name       string   `json:"name"`
	ConfigMaps []string `json:"configMaps"`
	Default    bool     `json:"default"`
}

// GuardrailDeploymentDetail is a NemoGuardrails deployment with its full specification.
type GuardrailDeploymentDetail struct {
	GuardrailDeployment
	ExposeRoute bool                  `json:"exposeRoute,omitempty"`
	EnvVarNames []string              `json:"envVarNames,omitempty"`
	NemoConfigs []GuardrailNemoConfig `json:"nemoConfigs,omitempty"`
	MainModel   string                `json:"mainModel,omitempty"`
}

// GuardrailModelWiring captures the model a guardrail deployment is wired to:
// the served model name used in requests and the OpenAI-compatible base URL
// (including the /v1 suffix) the guardrail proxies to.
type GuardrailModelWiring struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

// GuardrailDeploymentCreateRequest is the request body for deploying a guardrail
// template into a namespace. The optional Model wires the deployment to a
// serving model: it sets MAIN_MODEL_BASE_URL on the CR and injects a main
// model entry into the copied config.yaml.
type GuardrailDeploymentCreateRequest struct {
	Name              string                `json:"name"`
	Namespace         string                `json:"namespace"`
	TemplateName      string                `json:"templateName"`
	TemplateNamespace string                `json:"templateNamespace,omitempty"`
	Model             *GuardrailModelWiring `json:"model,omitempty"`
}

// ServingModel is a model served in a namespace that a guardrail deployment
// can be wired to.
type ServingModel struct {
	// Name is the model name to send in the request "model" field.
	Name string `json:"name"`
	// ResourceName is the name of the serving kubernetes resource.
	ResourceName string `json:"resourceName"`
	// Kind is the serving resource kind (InferenceService or LLMInferenceService).
	Kind string `json:"kind"`
	// Namespace is the namespace the serving resource lives in.
	Namespace string `json:"namespace"`
	// Endpoint is the in-cluster OpenAI-compatible base URL (including /v1).
	Endpoint string `json:"endpoint,omitempty"`
	// Ready reports whether the serving resource reports a Ready condition.
	Ready bool `json:"ready"`
}

// GuardrailTestRequest is the request body for testing a guardrail deployment.
type GuardrailTestRequest struct {
	// Message is the user message to run through the guardrails.
	Message string `json:"message"`
	// Model optionally overrides the model name; defaults to the deployment's
	// configured main model.
	Model string `json:"model,omitempty"`
}

// GuardrailTestResult is the outcome of running a test message through a
// guardrail deployment.
type GuardrailTestResult struct {
	// Status is the guardrail verdict: "passed", "blocked", or "unknown" when
	// the deployment's configuration does not support check requests.
	Status string `json:"status"`
	// Rail names the rail that produced the verdict, when the guardrail reports it.
	Rail string `json:"rail,omitempty"`
	// Content is the guardrail's response content (the block message when blocked).
	Content string `json:"content,omitempty"`
	// ModelResponse is the model's answer for a message that passed the rails.
	ModelResponse string `json:"modelResponse,omitempty"`
	// ConfigID is the guardrails configuration id the server reports.
	ConfigID string `json:"configId,omitempty"`
}
