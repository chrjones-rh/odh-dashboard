import React from 'react';
import {
  Alert,
  Bullseye,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  DrawerActions,
  DrawerCloseButton,
  DrawerHead,
  DrawerPanelBody,
  DrawerPanelContent,
  Label,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { useGuardrailDeployment } from '~/app/hooks/useGuardrails';
import { formatDate } from '~/app/utilities/guardrailsUtils';

type DeploymentDrawerPanelProps = {
  namespace: string;
  name: string | null;
  onClose: () => void;
};

const DeploymentDrawerPanel: React.FC<DeploymentDrawerPanelProps> = ({
  namespace,
  name,
  onClose,
}) => {
  const [detail, detailLoaded, detailError] = useGuardrailDeployment(namespace, name);

  return (
    <DrawerPanelContent
      widths={{ default: 'width_50' }}
      isResizable
      data-testid="guardrails-deployment-drawer"
    >
      <DrawerHead>
        <DrawerActions>
          <DrawerCloseButton onClick={onClose} data-testid="guardrails-deployment-drawer-close" />
        </DrawerActions>
        {name && (
          <Title headingLevel="h2" size="lg">
            {name}
          </Title>
        )}
      </DrawerHead>
      <DrawerPanelBody>
        {detailError && (
          <Alert variant="danger" title="Failed to load deployment details" isInline>
            {detailError.message}
          </Alert>
        )}
        {name && (!detail || !detailLoaded) && !detailError && (
          <Bullseye>
            <Spinner />
          </Bullseye>
        )}
        {name && detail && detailLoaded && (
          <>
            <DescriptionList isHorizontal columnModifier={{ default: '1Col' }}>
              <DescriptionListGroup>
                <DescriptionListTerm>Namespace</DescriptionListTerm>
                <DescriptionListDescription data-testid="guardrails-deployment-detail-namespace">
                  {detail.namespace}
                </DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Phase</DescriptionListTerm>
                <DescriptionListDescription>{detail.phase}</DescriptionListDescription>
              </DescriptionListGroup>
              {detail.endpoint && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Endpoint</DescriptionListTerm>
                  <DescriptionListDescription>{detail.endpoint}</DescriptionListDescription>
                </DescriptionListGroup>
              )}
              {typeof detail.replicas === 'number' && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Replicas</DescriptionListTerm>
                  <DescriptionListDescription>{detail.replicas}</DescriptionListDescription>
                </DescriptionListGroup>
              )}
              {detail.exposeRoute !== undefined && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Route exposure</DescriptionListTerm>
                  <DescriptionListDescription>
                    {detail.exposeRoute ? 'Exposed' : 'Not exposed'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
              )}
            </DescriptionList>

            {detail.conditions.length > 0 && (
              <>
                <Title headingLevel="h3" size="md" className="pf-v6-u-mt-lg pf-v6-u-mb-sm">
                  Conditions
                </Title>
                <Table aria-label="Deployment conditions" variant="compact">
                  <Thead>
                    <Tr>
                      <Th>Type</Th>
                      <Th>Status</Th>
                      <Th width={50}>Message</Th>
                      <Th>Last transition</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {detail.conditions.map((condition) => (
                      <Tr key={`${condition.type}-${condition.lastTransitionTime ?? ''}`}>
                        <Td dataLabel="Type">{condition.type}</Td>
                        <Td dataLabel="Status">{condition.status}</Td>
                        <Td dataLabel="Message">{condition.message ?? condition.reason ?? '—'}</Td>
                        <Td dataLabel="Last transition">
                          {condition.lastTransitionTime
                            ? formatDate(condition.lastTransitionTime)
                            : '—'}
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </>
            )}

            {detail.nemoConfigs && detail.nemoConfigs.length > 0 && (
              <>
                <Title headingLevel="h3" size="md" className="pf-v6-u-mt-lg pf-v6-u-mb-sm">
                  NeMo configurations
                </Title>
                {detail.nemoConfigs.map((nemoConfig) => (
                  <Alert
                    key={nemoConfig.name}
                    variant="info"
                    isInline
                    title={`${nemoConfig.name}${nemoConfig.default ? ' (default)' : ''}`}
                    className="pf-v6-u-mb-sm"
                  >
                    ConfigMaps: {nemoConfig.configMaps.join(', ')}
                  </Alert>
                ))}
              </>
            )}

            {detail.envVarNames && detail.envVarNames.length > 0 && (
              <>
                <Title headingLevel="h3" size="md" className="pf-v6-u-mt-lg pf-v6-u-mb-sm">
                  Environment variables
                </Title>
                {detail.envVarNames.map((envVarName) => (
                  <Label key={envVarName} className="pf-v6-u-mr-xs">
                    {envVarName}
                  </Label>
                ))}
              </>
            )}
          </>
        )}
      </DrawerPanelBody>
    </DrawerPanelContent>
  );
};

export default DeploymentDrawerPanel;
