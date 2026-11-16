import {
  APIOptions,
  handleRestFailures,
  isModArchResponse,
  restCREATE,
  restDELETE,
  restGET,
} from 'mod-arch-core';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import {
  GuardrailDeployment,
  GuardrailDeploymentCreateRequest,
  GuardrailDeploymentDetail,
  GuardrailTemplate,
  GuardrailTemplateDetail,
  GuardrailTestRequest,
  GuardrailTestResult,
  ServingModel,
} from '~/app/types';

export const getGuardrailTemplates =
  (hostPath: string, namespace: string) =>
  (opts: APIOptions): Promise<GuardrailTemplate[]> =>
    handleRestFailures(
      restGET(
        hostPath,
        `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/templates`,
        namespace ? { namespace } : {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GuardrailTemplate[]>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getGuardrailTemplate =
  (hostPath: string, name: string, namespace: string) =>
  (opts: APIOptions): Promise<GuardrailTemplateDetail> =>
    handleRestFailures(
      restGET(
        hostPath,
        `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/templates/${name}`,
        namespace ? { namespace } : {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GuardrailTemplateDetail>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getGuardrailDeployments =
  (hostPath: string, namespace: string) =>
  (opts: APIOptions): Promise<GuardrailDeployment[]> =>
    handleRestFailures(
      restGET(
        hostPath,
        `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/deployments`,
        { namespace },
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GuardrailDeployment[]>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getGuardrailDeployment =
  (hostPath: string, namespace: string, name: string) =>
  (opts: APIOptions): Promise<GuardrailDeploymentDetail> =>
    handleRestFailures(
      restGET(
        hostPath,
        `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/deployments/${name}`,
        { namespace },
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GuardrailDeploymentDetail>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const createGuardrailDeployment = (
  hostPath: string,
  request: GuardrailDeploymentCreateRequest,
): Promise<GuardrailDeployment> =>
  handleRestFailures(
    restCREATE(hostPath, `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/deployments`, {
      name: request.name,
      namespace: request.namespace,
      templateName: request.templateName,
      ...(request.templateNamespace ? { templateNamespace: request.templateNamespace } : {}),
      ...(request.model ? { model: request.model } : {}),
    }),
  ).then((response) => {
    if (isModArchResponse<GuardrailDeployment>(response)) {
      return response.data;
    }
    throw new Error('Invalid response format');
  });

export const getServingModels =
  (hostPath: string, namespace: string) =>
  (opts: APIOptions): Promise<ServingModel[]> =>
    handleRestFailures(
      restGET(hostPath, `${URL_PREFIX}/api/${BFF_API_VERSION}/models`, { namespace }, opts),
    ).then((response) => {
      if (isModArchResponse<ServingModel[]>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const testGuardrailDeployment = (
  hostPath: string,
  namespace: string,
  name: string,
  request: GuardrailTestRequest,
): Promise<GuardrailTestResult> =>
  handleRestFailures(
    restCREATE(
      hostPath,
      `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/deployments/${name}/test`,
      request,
    ),
  ).then((response) => {
    if (isModArchResponse<GuardrailTestResult>(response)) {
      return response.data;
    }
    throw new Error('Invalid response format');
  });

export const deleteGuardrailDeployment = (
  hostPath: string,
  namespace: string,
  name: string,
): Promise<void> =>
  handleRestFailures(
    restDELETE(
      hostPath,
      `${URL_PREFIX}/api/${BFF_API_VERSION}/guardrails/deployments/${name}`,
      {},
      { namespace },
      // The BFF responds with 204 No Content (an empty body); skip JSON parsing.
      { parseJSON: false },
    ),
  ).then(() => undefined);
