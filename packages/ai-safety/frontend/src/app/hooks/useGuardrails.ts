import React from 'react';
import { APIOptions, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import {
  getGuardrailDeployment,
  getGuardrailDeployments,
  getGuardrailTemplate,
  getGuardrailTemplates,
} from '~/app/api/guardrails';
import {
  GuardrailDeployment,
  GuardrailDeploymentDetail,
  GuardrailTemplate,
  GuardrailTemplateDetail,
} from '~/app/types';

export const useGuardrailTemplates = (
  namespace: string,
): [GuardrailTemplate[], boolean, Error | undefined, () => void] => {
  const callback = React.useCallback<FetchStateCallbackPromise<GuardrailTemplate[]>>(
    (opts: APIOptions) => getGuardrailTemplates('', namespace)(opts),
    [namespace],
  );
  return useFetchState<GuardrailTemplate[]>(callback, []);
};

export const useGuardrailTemplate = (
  name: string | null,
  namespace: string,
): [GuardrailTemplateDetail | null, boolean, Error | undefined, () => void] => {
  const callback = React.useCallback<FetchStateCallbackPromise<GuardrailTemplateDetail | null>>(
    async (opts: APIOptions) => {
      if (!name) {
        return Promise.resolve(null);
      }
      return getGuardrailTemplate('', name, namespace)(opts);
    },
    [name, namespace],
  );
  return useFetchState<GuardrailTemplateDetail | null>(callback, null);
};

export const useGuardrailDeployments = (
  namespace: string,
): [GuardrailDeployment[], boolean, Error | undefined, () => void] => {
  const callback = React.useCallback<FetchStateCallbackPromise<GuardrailDeployment[]>>(
    (opts: APIOptions) => {
      if (!namespace) {
        return Promise.resolve([]);
      }
      return getGuardrailDeployments('', namespace)(opts);
    },
    [namespace],
  );
  return useFetchState<GuardrailDeployment[]>(callback, []);
};

export const useGuardrailDeployment = (
  namespace: string,
  name: string | null,
): [GuardrailDeploymentDetail | null, boolean, Error | undefined, () => void] => {
  const callback = React.useCallback<FetchStateCallbackPromise<GuardrailDeploymentDetail | null>>(
    async (opts: APIOptions) => {
      if (!namespace || !name) {
        return Promise.resolve(null);
      }
      return getGuardrailDeployment('', namespace, name)(opts);
    },
    [namespace, name],
  );
  return useFetchState<GuardrailDeploymentDetail | null>(callback, null);
};
