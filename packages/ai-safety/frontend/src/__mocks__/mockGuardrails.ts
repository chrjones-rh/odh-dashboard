import type {
  GuardrailDeployment,
  GuardrailDeploymentDetail,
  GuardrailTemplate,
  GuardrailTemplateDetail,
} from '~/app/types';

export const mockGuardrailTemplate = (options?: Partial<GuardrailTemplate>): GuardrailTemplate => ({
  name: 'trustyai-service-operator-nemo-guardrails-default-pii',
  namespace: 'redhat-ods-applications',
  keys: ['config.yaml', 'rails.co'],
  createdAt: '2026-01-15T10:30:00Z',
  summary: {
    modelCount: 0,
    models: [],
    railFlows: [{ category: 'input', flow: 'mask pii on input' }],
    promptCount: 0,
    passthrough: true,
  },
  description: 'A guardrail that masks personally identifiable information.',
  actions: ['mask'],
  locations: ['input', 'output'],
  tags: ['pii', 'personal-information'],
  source: 'platform',
  ...options,
});

export const mockGuardrailTemplateDetail = (
  options?: Partial<GuardrailTemplateDetail>,
): GuardrailTemplateDetail => ({
  ...mockGuardrailTemplate(),
  files: [
    {
      key: 'config.yaml',
      content: 'passthrough: true\nrails:\n  input:\n    flows:\n      - mask pii on input\n',
    },
    { key: 'rails.co', content: 'define user ask pii\n  "blocked"' },
  ],
  ...options,
});

export const mockGuardrailDeployment = (
  options?: Partial<GuardrailDeployment>,
): GuardrailDeployment => ({
  name: 'my-guardrails',
  namespace: 'test-namespace',
  phase: 'Ready',
  ready: true,
  endpoint: 'https://my-guardrails.test-namespace.svc.cluster.local',
  replicas: 1,
  createdAt: '2026-01-15T10:30:00Z',
  conditions: [
    {
      type: 'DeploymentReady',
      status: 'True',
      reason: 'DeploymentReady',
      message: 'Deployment is ready',
      lastTransitionTime: '2026-01-15T10:31:00Z',
    },
  ],
  configSources: [
    {
      name: 'guardrail-placeholder',
      found: true,
      keys: ['config.yaml', 'prompts.yml'],
      hasModelSpec: true,
      isOperatorTemplate: false,
    },
  ],
  ...options,
});

export const mockGuardrailDeploymentDetail = (
  options?: Partial<GuardrailDeploymentDetail>,
): GuardrailDeploymentDetail => ({
  ...mockGuardrailDeployment(),
  exposeRoute: true,
  envVarNames: ['OPENAI_API_KEY'],
  nemoConfigs: [
    {
      name: 'main-config',
      configMaps: ['guardrail-placeholder'],
      default: true,
    },
  ],
  ...options,
});
