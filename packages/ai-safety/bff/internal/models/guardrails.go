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
}

// GuardrailDeployment is a NemoGuardrails deployment in a namespace.
type GuardrailDeployment struct {
	Name          string                  `json:"name"`
	Namespace     string                  `json:"namespace"`
	Phase         string                  `json:"phase"`
	Ready         bool                    `json:"ready"`
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
}

// GuardrailDeploymentCreateRequest is the request body for deploying a guardrail
// template into a namespace.
type GuardrailDeploymentCreateRequest struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	TemplateName      string `json:"templateName"`
	TemplateNamespace string `json:"templateNamespace,omitempty"`
}
