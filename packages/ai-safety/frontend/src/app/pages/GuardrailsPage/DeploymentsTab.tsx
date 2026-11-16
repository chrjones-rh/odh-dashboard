import React from 'react';
import {
  Alert,
  Bullseye,
  Button,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  Label,
  Spinner,
  Title,
  Tooltip,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import {
  getConfigSourceMode,
  getDeploymentStatus,
  formatDate,
} from '~/app/utilities/guardrailsUtils';
import { GuardrailConfigSource, GuardrailDeployment } from '~/app/types';

const configSourceLabels: Record<string, string> = {
  template: 'Template',
  custom: 'Custom config',
  missing: 'Missing',
};

const ConfigSourceLabel: React.FC<{ source: GuardrailConfigSource }> = ({ source }) => {
  const mode = getConfigSourceMode(source);
  const color = mode === 'template' ? 'teal' : mode === 'custom' ? 'blue' : 'red';
  return (
    <Label key={source.name} color={color} className="pf-v6-u-mr-xs">
      {configSourceLabels[mode]} · {source.name}
    </Label>
  );
};

const StatusLabel: React.FC<{ deployment: GuardrailDeployment }> = ({ deployment }) => {
  const status = getDeploymentStatus(deployment);
  const color = status.isReady ? 'green' : status.isFailed ? 'red' : 'orange';
  return (
    <Label color={color} data-testid={`guardrails-deployment-status-${deployment.name}`}>
      {status.label}
    </Label>
  );
};

type DeploymentsTabProps = {
  onSelect: (name: string | null) => void;
  onTest: (deployment: GuardrailDeployment) => void;
  onDelete: (deployment: GuardrailDeployment) => void;
  deployments: GuardrailDeployment[];
  deploymentsLoaded: boolean;
  deploymentsError?: Error;
};

const DeploymentsTab: React.FC<DeploymentsTabProps> = ({
  onSelect,
  onTest,
  onDelete,
  deployments,
  deploymentsLoaded,
  deploymentsError,
}) => {
  if (!deploymentsLoaded) {
    return (
      <Bullseye data-testid="guardrails-deployments-loading">
        <Spinner />
      </Bullseye>
    );
  }

  if (deploymentsError) {
    return (
      <Alert variant="danger" title="Failed to load guardrails deployments" isInline>
        {deploymentsError.message}
      </Alert>
    );
  }

  return (
    <Table aria-label="Guardrails deployments" data-testid="guardrails-deployments-table">
      <Thead>
        <Tr>
          <Th width={20}>Name</Th>
          <Th width={10}>Status</Th>
          <Th width={25}>Configuration</Th>
          <Th width={20}>Endpoint</Th>
          <Th width={10}>Created</Th>
          <Th width={15}>Actions</Th>
        </Tr>
      </Thead>
      <Tbody>
        {deployments.map((deployment: GuardrailDeployment) => (
          <Tr
            key={deployment.name}
            onRowClick={() => onSelect(deployment.name)}
            isClickable
            data-testid={`guardrails-deployment-row-${deployment.name}`}
          >
            <Td dataLabel="Name">{deployment.name}</Td>
            <Td dataLabel="Status">
              <StatusLabel deployment={deployment} />
            </Td>
            <Td dataLabel="Configuration">
              {deployment.configSources.length > 0 ? (
                deployment.configSources.map((source) => (
                  <ConfigSourceLabel key={source.name} source={source} />
                ))
              ) : (
                <span>—</span>
              )}
            </Td>
            <Td dataLabel="Endpoint">{deployment.endpoint ?? '—'}</Td>
            <Td dataLabel="Created">{formatDate(deployment.createdAt)}</Td>
            <Td dataLabel="Actions">
              <Tooltip
                content={
                  deployment.ready && deployment.modelWired
                    ? 'Run a test message through this guardrail'
                    : 'The deployment must be ready and wired to a model before it can be tested'
                }
              >
                <span className="pf-v6-u-display-inline-block">
                  <Button
                    variant="secondary"
                    isDisabled={!deployment.ready || !deployment.modelWired}
                    onClick={() => onTest(deployment)}
                    data-testid={`guardrails-deployment-test-${deployment.name}`}
                  >
                    Test
                  </Button>
                </span>
              </Tooltip>{' '}
              <Button
                variant="danger"
                onClick={() => onDelete(deployment)}
                data-testid={`guardrails-deployment-delete-${deployment.name}`}
              >
                Delete
              </Button>
            </Td>
          </Tr>
        ))}
        {deployments.length === 0 && (
          <Tr>
            <Td colSpan={6}>
              <EmptyState
                variant={EmptyStateVariant.sm}
                icon={SearchIcon}
                data-testid="guardrails-deployments-empty"
              >
                <Title headingLevel="h3" size="lg">
                  No guardrails deployments
                </Title>
                <EmptyStateBody>No NeMo Guardrails deployments were found.</EmptyStateBody>
              </EmptyState>
            </Td>
          </Tr>
        )}
      </Tbody>
    </Table>
  );
};

export default DeploymentsTab;
