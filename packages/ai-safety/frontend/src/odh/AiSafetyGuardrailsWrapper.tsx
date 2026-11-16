import React from 'react';
import {
  ModularArchConfig,
  DeploymentMode,
  ModularArchContextProvider,
  NotificationContextProvider,
  BrowserStorageContextProvider,
} from 'mod-arch-core';
import { URL_PREFIX } from '~/app/utilities/const';
import GuardrailsPage from '~/app/pages/GuardrailsPage/GuardrailsPage';

const modularArchConfig: ModularArchConfig = {
  deploymentMode: DeploymentMode.Federated,
  URL_PREFIX,
  BFF_API_VERSION: 'v1',
};

const AiSafetyGuardrailsWrapper: React.FC = () => (
  <ModularArchContextProvider config={modularArchConfig}>
    <BrowserStorageContextProvider>
      <NotificationContextProvider>
        <GuardrailsPage />
      </NotificationContextProvider>
    </BrowserStorageContextProvider>
  </ModularArchContextProvider>
);

export default AiSafetyGuardrailsWrapper;
