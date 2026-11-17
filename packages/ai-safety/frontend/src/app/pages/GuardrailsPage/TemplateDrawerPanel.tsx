import React from 'react';
import {
  Alert,
  Bullseye,
  Button,
  CodeBlock,
  CodeBlockCode,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  DrawerActions,
  DrawerCloseButton,
  DrawerHead,
  DrawerPanelBody,
  DrawerPanelContent,
  DrawerPanelDescription,
  Label,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { PencilAltIcon } from '@patternfly/react-icons';
import { useGuardrailTemplate } from '~/app/hooks/useGuardrails';
import { getTemplateDisplayName } from '~/app/utilities/guardrailsUtils';
import { GuardrailTemplate } from '~/app/types';

type TemplateDrawerPanelProps = {
  template: GuardrailTemplate | null;
  onClose: () => void;
  onEdit: (template: GuardrailTemplate) => void;
};

const TemplateDrawerPanel: React.FC<TemplateDrawerPanelProps> = ({ template, onClose, onEdit }) => {
  const [detail, detailLoaded, detailError] = useGuardrailTemplate(
    template?.name ?? null,
    template?.namespace ?? '',
  );

  return (
    <DrawerPanelContent
      widths={{ default: 'width_50' }}
      isResizable
      data-testid="guardrails-template-drawer"
    >
      <DrawerHead>
        <DrawerActions>
          {template?.source === 'user' && (
            <Button
              variant="plain"
              aria-label={`Edit ${template.name}`}
              icon={<PencilAltIcon />}
              onClick={() => onEdit(template)}
              data-testid={`guardrails-template-edit-${getTemplateDisplayName(template.name)}`}
            />
          )}
          <DrawerCloseButton onClick={onClose} data-testid="guardrails-template-drawer-close" />
        </DrawerActions>
        {template && (
          <>
            <Title headingLevel="h2" size="lg">
              {getTemplateDisplayName(template.name)}
            </Title>
            {detail?.description && (
              <DrawerPanelDescription>{detail.description}</DrawerPanelDescription>
            )}
          </>
        )}
      </DrawerHead>
      <DrawerPanelBody>
        {detailError && (
          <Alert variant="danger" title="Failed to load template contents" isInline>
            {detailError.message}
          </Alert>
        )}
        {template && (!detail || !detailLoaded) && !detailError && (
          <Bullseye>
            <Spinner />
          </Bullseye>
        )}
        {template && detail && detailLoaded && (
          <>
            <DescriptionList
              isHorizontal
              columnModifier={{ default: '1Col' }}
              className="pf-v6-u-mb-md"
            >
              {detail.locations && detail.locations.length > 0 && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Enforcement locations</DescriptionListTerm>
                  <DescriptionListDescription>
                    {detail.locations.join(', ')}
                  </DescriptionListDescription>
                </DescriptionListGroup>
              )}
              {detail.actions && detail.actions.length > 0 && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Actions</DescriptionListTerm>
                  <DescriptionListDescription>
                    {detail.actions.join(', ')}
                  </DescriptionListDescription>
                </DescriptionListGroup>
              )}
            </DescriptionList>
            {detail.tags && detail.tags.length > 0 && (
              <div className="pf-v6-u-mb-md">
                {detail.tags.map((tag) => (
                  <Label key={tag} className="pf-v6-u-mr-xs pf-v6-u-mb-xs">
                    {tag}
                  </Label>
                ))}
              </div>
            )}
            {detail.summary.passthrough && (
              <Alert variant="info" isInline title="Passthrough mode" className="pf-v6-u-mb-md">
                This template runs without calling an LLM (passthrough).
              </Alert>
            )}
            {detail.files
              .filter((file) => file.key === 'config.yaml')
              .map((file) => (
                <div key={file.key} className="pf-v6-u-mb-lg">
                  <Title headingLevel="h4" size="md" className="pf-v6-u-mb-xs">
                    {file.key}
                  </Title>
                  <CodeBlock>
                    <CodeBlockCode>{file.content}</CodeBlockCode>
                  </CodeBlock>
                </div>
              ))}
            {detail.files.length > 0 &&
              !detail.files.some((file) => file.key === 'config.yaml') && (
                <Alert variant="info" isInline title="No config.yaml">
                  This template does not include a config.yaml file.
                </Alert>
              )}
          </>
        )}
      </DrawerPanelBody>
    </DrawerPanelContent>
  );
};

export default TemplateDrawerPanel;
