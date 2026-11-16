export type DisplayNameAnnotations = Partial<{
  'openshift.io/description': string;
  'openshift.io/display-name': string;
}>;

export type K8sCondition = {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastProbeTime?: string | null;
  lastTransitionTime?: string;
  lastHeartbeatTime?: string;
};

export type ListConfigSecretsResponse = {
  secrets: { name: string; keys: string[] }[];
  configMaps: { name: string; keys: string[] }[];
};

export type ConfigSecretItem = {
  name: string;
  keys: string[];
};

export type NamespaceKind = {
  name: string;
  displayName?: string;
};

export type GuardrailRailFlow = {
  category: string;
  flow: string;
};

export type GuardrailConfigSummary = {
  modelCount: number;
  models: string[];
  railFlows: GuardrailRailFlow[];
  promptCount: number;
  passthrough: boolean;
};

export type GuardrailTemplate = {
  name: string;
  namespace: string;
  keys: string[];
  createdAt: string;
  summary: GuardrailConfigSummary;
  description?: string;
  actions?: string[];
  locations?: string[];
  tags?: string[];
  source: 'platform' | 'user';
};

export type GuardrailConfigFile = {
  key: string;
  content: string;
};

export type GuardrailTemplateDetail = GuardrailTemplate & {
  files: GuardrailConfigFile[];
};

export type GuardrailCondition = {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
};

export type GuardrailConfigSource = {
  name: string;
  found: boolean;
  keys?: string[];
  hasModelSpec?: boolean;
  isOperatorTemplate?: boolean;
  mainModel?: string;
};

export type GuardrailDeployment = {
  name: string;
  namespace: string;
  phase: string;
  ready: boolean;
  modelWired: boolean;
  endpoint?: string;
  replicas?: number;
  createdAt: string;
  conditions: GuardrailCondition[];
  configSources: GuardrailConfigSource[];
};

export type GuardrailNemoConfig = {
  name: string;
  configMaps: string[];
  default: boolean;
};

export type GuardrailDeploymentDetail = GuardrailDeployment & {
  exposeRoute?: boolean;
  envVarNames?: string[];
  nemoConfigs?: GuardrailNemoConfig[];
  mainModel?: string;
};

export type GuardrailModelWiring = {
  name: string;
  endpoint: string;
};

export type GuardrailTemplateUpsertRequest = {
  name: string;
  namespace: string;
  description?: string;
  configYAML: string;
};

export type GuardrailDeploymentCreateRequest = {
  name: string;
  namespace: string;
  templateName: string;
  templateNamespace?: string;
  model?: GuardrailModelWiring;
};

export type ServingModel = {
  name: string;
  resourceName: string;
  kind: 'InferenceService' | 'LLMInferenceService';
  namespace: string;
  endpoint?: string;
  ready: boolean;
};

export type GuardrailTestRequest = {
  message: string;
  model?: string;
};

export type GuardrailTestResult = {
  status: 'passed' | 'blocked' | 'unknown' | string;
  rail?: string;
  content?: string;
  modelResponse?: string;
  configId?: string;
};
