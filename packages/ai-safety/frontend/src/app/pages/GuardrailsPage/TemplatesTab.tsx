import React from 'react';
import {
  Alert,
  Bullseye,
  Button,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  Label,
  SearchInput,
  Spinner,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon, UploadIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { getTemplateDisplayName, formatDate } from '~/app/utilities/guardrailsUtils';
import { GuardrailTemplate } from '~/app/types';

type TemplatesTabProps = {
  templates: GuardrailTemplate[];
  templatesLoaded: boolean;
  templatesError?: Error;
  onUpload: () => void;
  onSelect: (template: GuardrailTemplate | null) => void;
  onDeploy: (template: GuardrailTemplate | null) => void;
};

const TemplatesTab: React.FC<TemplatesTabProps> = ({
  templates,
  templatesLoaded,
  templatesError,
  onUpload,
  onSelect,
  onDeploy,
}) => {
  const [query, setQuery] = React.useState('');

  const filteredTemplates = React.useMemo(() => {
    const trimmed = query.trim().toLowerCase();
    if (!trimmed) {
      return templates;
    }
    return templates.filter((template) =>
      `${template.name} ${getTemplateDisplayName(template.name)} ${template.keys.join(' ')} ${
        template.description ?? ''
      }`
        .toLowerCase()
        .includes(trimmed),
    );
  }, [templates, query]);

  if (templatesError) {
    return (
      <Alert variant="danger" title="Failed to load guardrail templates" isInline>
        {templatesError.message}
      </Alert>
    );
  }

  if (!templatesLoaded) {
    return (
      <Bullseye data-testid="guardrails-templates-loading">
        <Spinner />
      </Bullseye>
    );
  }

  return (
    <>
      <Toolbar data-testid="guardrails-templates-toolbar">
        <ToolbarContent>
          <ToolbarItem>
            <SearchInput
              aria-label="Search guardrail templates"
              placeholder="Search templates"
              value={query}
              onChange={(_event, value) => setQuery(value)}
              onClear={() => setQuery('')}
              data-testid="guardrails-templates-search"
            />
          </ToolbarItem>
          <ToolbarItem>
            <Button
              variant="secondary"
              icon={<UploadIcon />}
              onClick={onUpload}
              data-testid="guardrails-templates-upload-button"
            >
              Upload template
            </Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>
      <Table aria-label="Guardrail templates" data-testid="guardrails-templates-table">
        <Thead>
          <Tr>
            <Th width={30}>Template</Th>
            <Th width={20}>Configuration files</Th>
            <Th width={20}>Models</Th>
            <Th width={10}>Rail flows</Th>
            <Th width={15}>Created</Th>
            <Th width={10}>Actions</Th>
          </Tr>
        </Thead>
        <Tbody>
          {filteredTemplates.map((template: GuardrailTemplate) => (
            <Tr
              key={`${template.namespace}/${template.name}`}
              onRowClick={() => onSelect(template)}
              isClickable
              data-testid={`guardrails-template-row-${getTemplateDisplayName(template.name)}`}
            >
              <Td dataLabel="Template">
                <Title headingLevel="h4" size="md">
                  {getTemplateDisplayName(template.name)}
                  {template.source === 'user' && (
                    <Label
                      color="grey"
                      className="pf-v6-u-ml-sm"
                      data-testid={`guardrails-template-origin-${getTemplateDisplayName(template.name)}`}
                    >
                      {template.namespace}
                    </Label>
                  )}
                </Title>
                {template.description && (
                  <span
                    className="pf-v6-u-font-size-sm pf-v6-u-color-200"
                    data-testid={`guardrails-template-description-${getTemplateDisplayName(template.name)}`}
                  >
                    {template.description}
                  </span>
                )}
              </Td>
              <Td dataLabel="Configuration files">{template.keys.join(', ')}</Td>
              <Td dataLabel="Models">
                {template.summary.models.length > 0
                  ? template.summary.models.join(', ')
                  : 'No model required'}
              </Td>
              <Td dataLabel="Rail flows">{template.summary.railFlows.length}</Td>
              <Td dataLabel="Created">{formatDate(template.createdAt)}</Td>
              <Td dataLabel="Actions">
                <Button
                  variant="secondary"
                  onClick={() => onDeploy(template)}
                  data-testid={`guardrails-template-deploy-${getTemplateDisplayName(template.name)}`}
                >
                  Deploy
                </Button>
              </Td>
            </Tr>
          ))}
          {filteredTemplates.length === 0 && (
            <Tr>
              <Td colSpan={6}>
                <EmptyState
                  variant={EmptyStateVariant.sm}
                  icon={SearchIcon}
                  data-testid="guardrails-templates-empty"
                >
                  <Title headingLevel="h3" size="lg">
                    No templates found
                  </Title>
                  <EmptyStateBody>
                    {templates.length === 0
                      ? 'No guardrail templates are available on this platform.'
                      : 'No templates match the search query.'}
                  </EmptyStateBody>
                </EmptyState>
              </Td>
            </Tr>
          )}
        </Tbody>
      </Table>
    </>
  );
};

export default TemplatesTab;
