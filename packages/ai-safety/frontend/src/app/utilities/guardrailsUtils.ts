import { GuardrailConfigSource, GuardrailDeployment } from '~/app/types';

/** Strips the operator template prefix for display purposes. */
export const getTemplateDisplayName = (name: string): string =>
  name.replace(/^trustyai-service-operator-nemo-guardrails-/, '');

export type GuardrailStatus = {
  label: string;
  isReady: boolean;
  isFailed: boolean;
};

export const getDeploymentStatus = (deployment: GuardrailDeployment): GuardrailStatus => {
  // A freshly created CR has no phase until the operator reconciles it.
  const phase = deployment.phase || 'Pending';
  if (deployment.ready) {
    return { label: phase, isReady: true, isFailed: false };
  }
  const isFailed = /fail|error|degrad|unknown/i.test(phase);
  return { label: phase, isReady: false, isFailed };
};

export type ConfigSourceMode = 'template' | 'custom' | 'missing';

export const getConfigSourceMode = (source: GuardrailConfigSource): ConfigSourceMode => {
  if (!source.found) {
    return 'missing';
  }
  // A config.yaml models section is NOT a custom-config signal: the
  // dashboard's own model wiring injects one into template copies. Origin is
  // what the ConfigMap records: dashboard copies (with their recorded
  // template), operator templates, or user-managed ConfigMaps.
  if (source.templateName || source.isOperatorTemplate || source.isDashboardManaged) {
    return 'template';
  }
  return 'custom';
};

export const formatDate = (iso: string): string => {
  if (!iso) {
    return 'Unknown';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleString();
};
