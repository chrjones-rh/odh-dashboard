import { testHook } from '~/__tests__/unit/testUtils/hooks';
import {
  useGuardrailDeployment,
  useGuardrailDeployments,
  useGuardrailTemplate,
  useGuardrailTemplates,
} from '~/app/hooks/useGuardrails';
import * as guardrailsApi from '~/app/api/guardrails';
import { GuardrailTemplate } from '~/app/types';

jest.mock('~/app/api/guardrails', () => ({
  getGuardrailTemplates: jest.fn(),
  getGuardrailTemplate: jest.fn(),
  getGuardrailDeployments: jest.fn(),
  getGuardrailDeployment: jest.fn(),
}));

const apiMocks = jest.mocked(guardrailsApi);

const mockTemplate: GuardrailTemplate = {
  name: 'trustyai-service-operator-nemo-guardrails-default-pii',
  namespace: 'redhat-ods-applications',
  keys: ['config.yaml', 'rails.co'],
  createdAt: '2026-01-01T00:00:00Z',
  source: 'platform',
  summary: {
    modelCount: 0,
    models: [],
    railFlows: [{ category: 'input', flow: 'mask pii' }],
    promptCount: 0,
    passthrough: true,
  },
};

describe('useGuardrailTemplates', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should fetch templates', async () => {
    apiMocks.getGuardrailTemplates.mockReturnValue(
      jest.fn().mockResolvedValue([mockTemplate]) as never,
    );

    const renderResult = testHook(useGuardrailTemplates)('test-ns');
    await renderResult.waitForNextUpdate();

    const [templates, loaded, error] = renderResult.result.current;
    expect(loaded).toBe(true);
    expect(error).toBeUndefined();
    expect(templates).toEqual([mockTemplate]);
    expect(apiMocks.getGuardrailTemplates).toHaveBeenCalledTimes(1);
  });

  it('should surface fetch errors', async () => {
    apiMocks.getGuardrailTemplates.mockReturnValue(
      jest.fn().mockRejectedValue(new Error('fetch failed')) as never,
    );

    const renderResult = testHook(useGuardrailTemplates)('test-ns');
    await renderResult.waitForNextUpdate();

    const [, loaded, error] = renderResult.result.current;
    expect(loaded).toBe(false);
    expect(error).toEqual(new Error('fetch failed'));
  });
});

describe('useGuardrailTemplate', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should not fetch when no name is selected', async () => {
    const renderResult = testHook(useGuardrailTemplate)(null, '');
    await renderResult.waitForNextUpdate();

    const [detail, loaded] = renderResult.result.current;
    expect(detail).toBeNull();
    expect(loaded).toBe(true);
    expect(apiMocks.getGuardrailTemplate).not.toHaveBeenCalled();
  });

  it('should fetch the template detail when a name is selected', async () => {
    apiMocks.getGuardrailTemplate.mockReturnValue(
      jest.fn().mockResolvedValue({ ...mockTemplate, files: [] }) as never,
    );

    const renderResult = testHook(useGuardrailTemplate)(
      mockTemplate.name,
      'redhat-ods-applications',
    );
    await renderResult.waitForNextUpdate();

    const [detail, loaded, error] = renderResult.result.current;
    expect(loaded).toBe(true);
    expect(error).toBeUndefined();
    expect(detail).toEqual({ ...mockTemplate, files: [] });
  });
});

describe('useGuardrailDeployments', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return an empty list when no namespace is set', async () => {
    const renderResult = testHook(useGuardrailDeployments)('');
    await renderResult.waitForNextUpdate();

    const [deployments, loaded] = renderResult.result.current;
    expect(deployments).toEqual([]);
    expect(loaded).toBe(true);
    expect(apiMocks.getGuardrailDeployments).not.toHaveBeenCalled();
  });

  it('should fetch deployments for the namespace', async () => {
    const deployment = {
      name: 'my-guardrails',
      namespace: 'test-ns',
      phase: 'Ready',
      ready: true,
      createdAt: '2026-01-01T00:00:00Z',
      conditions: [],
      configSources: [],
    };
    apiMocks.getGuardrailDeployments.mockReturnValue(
      jest.fn().mockResolvedValue([deployment]) as never,
    );

    const renderResult = testHook(useGuardrailDeployments)('test-ns');
    await renderResult.waitForNextUpdate();

    const [result, loaded, error] = renderResult.result.current;
    expect(loaded).toBe(true);
    expect(error).toBeUndefined();
    expect(result).toEqual([deployment]);
    expect(apiMocks.getGuardrailDeployments).toHaveBeenCalledWith('', 'test-ns');
  });
});

describe('useGuardrailDeployment', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should not fetch when name or namespace is missing', async () => {
    const renderResult = testHook(useGuardrailDeployment)('test-ns', null);
    await renderResult.waitForNextUpdate();

    const [detail, loaded] = renderResult.result.current;
    expect(detail).toBeNull();
    expect(loaded).toBe(true);
    expect(apiMocks.getGuardrailDeployment).not.toHaveBeenCalled();
  });

  it('should fetch the deployment detail', async () => {
    const detail = {
      name: 'my-guardrails',
      namespace: 'test-ns',
      phase: 'Ready',
      ready: true,
      createdAt: '2026-01-01T00:00:00Z',
      conditions: [],
      configSources: [],
      envVarNames: ['OPENAI_API_KEY'],
    };
    apiMocks.getGuardrailDeployment.mockReturnValue(jest.fn().mockResolvedValue(detail) as never);

    const renderResult = testHook(useGuardrailDeployment)('test-ns', 'my-guardrails');
    await renderResult.waitForNextUpdate();

    const [result, loaded, error] = renderResult.result.current;
    expect(loaded).toBe(true);
    expect(error).toBeUndefined();
    expect(result).toEqual(detail);
  });
});
