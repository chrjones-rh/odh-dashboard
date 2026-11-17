import React from 'react';
import {
  Alert,
  Bullseye,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  Label,
  MenuToggle,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { RhUiEllipsisVerticalFillIcon, SearchIcon } from '@patternfly/react-icons';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import {
  getConfigSourceMode,
  getDeploymentStatus,
  getTemplateDisplayName,
  formatDate,
} from '~/app/utilities/guardrailsUtils';
import { GuardrailConfigSource, GuardrailDeployment } from '~/app/types';

const ConfigSourceLabel: React.FC<{ source: GuardrailConfigSource }> = ({ source }) => {
  const mode = getConfigSourceMode(source);
  const color = mode === 'template' ? 'teal' : mode === 'custom' ? 'blue' : 'red';
  let text: string;
  switch (mode) {
    case 'template':
      // Dashboard copies are named after the deployment, which would be
      // redundant — name the template the copy was made from instead.
      text = source.templateName
        ? `Template · ${getTemplateDisplayName(source.templateName)}`
        : 'Template';
      break;
    case 'custom':
      text = `Custom config · ${source.name}`;
      break;
    default:
      text = `Missing · ${source.name}`;
  }
  return (
    <Label key={source.name} color={color} className="pf-v6-u-mr-xs">
      {text}
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
            onRowClick={(event) => {
              // Ignore clicks that propagate up from the actions menu and
              // other interactive descendants — they have their own handlers.
              const clickedInteractiveDescendant =
                event?.target instanceof Element &&
                !!event.target.closest('a, button, input, label');
              if (!clickedInteractiveDescendant) {
                onSelect(deployment.name);
              }
            }}
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
            <Td isActionCell>
              <ActionsColumn
                items={[
                  {
                    title: (
                      <span data-testid={`guardrails-deployment-test-${deployment.name}`}>
                        Test
                      </span>
                    ),
                    isAriaDisabled: !deployment.ready || !deployment.modelWired,
                    ...(deployment.ready && deployment.modelWired
                      ? {}
                      : {
                          tooltipProps: {
                            content:
                              'The deployment must be ready and wired to a model before it can be tested',
                            position: 'left' as const,
                          },
                        }),
                    onClick: () => onTest(deployment),
                  },
                  {
                    title: (
                      <span data-testid={`guardrails-deployment-delete-${deployment.name}`}>
                        Delete
                      </span>
                    ),
                    onClick: () => onDelete(deployment),
                  },
                ]}
                actionsToggle={({ toggleRef, onToggle, isOpen }) => (
                  <MenuToggle
                    aria-label={`${deployment.name} actions`}
                    ref={toggleRef}
                    onClick={(event) => onToggle(event)}
                    isExpanded={isOpen}
                    variant="plain"
                    icon={<RhUiEllipsisVerticalFillIcon />}
                    data-testid={`guardrails-deployment-actions-${deployment.name}`}
                  />
                )}
              />
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
