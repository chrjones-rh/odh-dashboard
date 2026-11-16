import {
  getConfigSourceMode,
  getDeploymentStatus,
  getTemplateDisplayName,
  formatDate,
} from '~/app/utilities/guardrailsUtils';
import { GuardrailConfigSource, GuardrailDeployment } from '~/app/types';

describe('getTemplateDisplayName', () => {
  it('should strip the operator template prefix', () => {
    expect(getTemplateDisplayName('trustyai-service-operator-nemo-guardrails-default-pii')).toBe(
      'default-pii',
    );
  });

  it('should return the name unchanged when the prefix does not match', () => {
    expect(getTemplateDisplayName('guardrail-placeholder')).toBe('guardrail-placeholder');
  });
});

describe('getDeploymentStatus', () => {
  const makeDeployment = (phase: string, ready: boolean): GuardrailDeployment => ({
    name: 'test',
    namespace: 'test-ns',
    phase,
    ready,
    createdAt: '2026-01-01T00:00:00Z',
    conditions: [],
    configSources: [],
  });

  it('should mark ready deployments as ready', () => {
    const status = getDeploymentStatus(makeDeployment('Ready', true));
    expect(status.isReady).toBe(true);
    expect(status.isFailed).toBe(false);
    expect(status.label).toBe('Ready');
  });

  it('should mark failed phases as failed', () => {
    const status = getDeploymentStatus(makeDeployment('Failed', false));
    expect(status.isReady).toBe(false);
    expect(status.isFailed).toBe(true);
  });

  it('should mark unknown phases as failed', () => {
    const status = getDeploymentStatus(makeDeployment('Unknown', false));
    expect(status.isFailed).toBe(true);
  });

  it('should treat empty phases as pending, not failed', () => {
    const status = getDeploymentStatus(makeDeployment('', false));
    expect(status.label).toBe('Pending');
    expect(status.isReady).toBe(false);
    expect(status.isFailed).toBe(false);
  });

  it('should treat pending phases as neither ready nor failed', () => {
    const status = getDeploymentStatus(makeDeployment('Provisioning', false));
    expect(status.isReady).toBe(false);
    expect(status.isFailed).toBe(false);
  });
});

describe('getConfigSourceMode', () => {
  const makeSource = (options: Partial<GuardrailConfigSource>): GuardrailConfigSource => ({
    name: 'test-config',
    found: true,
    ...options,
  });

  it('should classify missing configmaps as missing', () => {
    expect(getConfigSourceMode(makeSource({ found: false }))).toBe('missing');
  });

  it('should classify configmaps with a model spec as custom', () => {
    expect(getConfigSourceMode(makeSource({ hasModelSpec: true }))).toBe('custom');
  });

  it('should classify configmaps without a model spec as template', () => {
    expect(getConfigSourceMode(makeSource({ hasModelSpec: false }))).toBe('template');
  });
});

describe('formatDate', () => {
  it('should return Unknown for empty values', () => {
    expect(formatDate('')).toBe('Unknown');
  });

  it('should return the raw value for unparseable dates', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });

  it('should format valid dates', () => {
    const result = formatDate('2026-01-15T10:30:00Z');
    expect(result).not.toBe('2026-01-15T10:30:00Z');
    expect(result).not.toBe('Unknown');
  });
});
