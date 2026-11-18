import React from 'react';
import {
  Drawer,
  DrawerContent,
  DrawerContentBody,
  Tab,
  Tabs,
  TabTitleText,
} from '@patternfly/react-core';
import { useNamespaceSelector } from 'mod-arch-core';
import ApplicationsPage from '~/app/components/ApplicationsPage';
import { useGuardrailDeployments, useGuardrailTemplates } from '~/app/hooks/useGuardrails';
import { GuardrailDeployment, GuardrailTemplate } from '~/app/types';
import TemplatesTab from './TemplatesTab';
import DeploymentsTab from './DeploymentsTab';
import TemplateDrawerPanel from './TemplateDrawerPanel';
import DeploymentDrawerPanel from './DeploymentDrawerPanel';
import DeployGuardrailModal from './DeployGuardrailModal';
import DeleteGuardrailModal from './DeleteGuardrailModal';
import TestGuardrailModal from './TestGuardrailModal';
import ManageTemplateModal, { ManageTemplateTarget } from './ManageTemplateModal';
import DeleteTemplateModal from './DeleteTemplateModal';
import NamespaceSelector from './NamespaceSelector';

const TEMPLATES_TAB = 0;
const DEPLOYMENTS_TAB = 1;

const GuardrailsPage: React.FC = () => {
  const [activeTabKey, setActiveTabKey] = React.useState<string | number>(TEMPLATES_TAB);
  const [selectedTemplate, setSelectedTemplate] = React.useState<GuardrailTemplate | null>(null);
  const [selectedDeployment, setSelectedDeployment] = React.useState<string | null>(null);
  const [deployTemplate, setDeployTemplate] = React.useState<GuardrailTemplate | null>(null);
  const [deleteDeployment, setDeleteDeployment] = React.useState<GuardrailDeployment | null>(null);
  const [testDeployment, setTestDeployment] = React.useState<GuardrailDeployment | null>(null);
  const [manageTemplate, setManageTemplate] = React.useState<ManageTemplateTarget | null>(null);
  const [deleteTemplate, setDeleteTemplate] = React.useState<GuardrailTemplate | null>(null);

  // The selected namespace is persisted (last used) and restored on load,
  // falling back to the first accessible namespace — the same pattern the
  // EvalHub module uses.
  const {
    namespaces,
    namespacesLoaded,
    namespacesLoadError,
    preferredNamespace,
    updatePreferredNamespace,
  } = useNamespaceSelector({ storeLastNamespace: true });
  const namespace = preferredNamespace?.name ?? '';

  const [deployments, deploymentsLoaded, deploymentsError, refreshDeployments] =
    useGuardrailDeployments(namespace);
  const [templates, templatesLoaded, templatesError, refreshTemplates] =
    useGuardrailTemplates(namespace);

  const onTabSelect = React.useCallback((_event: React.MouseEvent, tabKey: string | number) => {
    setActiveTabKey(tabKey);
    // Clear the drawer selection when switching tabs so the panel does not
    // show content from the previous tab.
    setSelectedTemplate(null);
    setSelectedDeployment(null);
  }, []);

  const onNamespaceChange = React.useCallback(
    (nextNamespace: string) => {
      const match = namespaces.find((ns) => ns.name === nextNamespace);
      updatePreferredNamespace(match ?? { name: nextNamespace });
      setSelectedTemplate(null);
      setSelectedDeployment(null);
    },
    [namespaces, updatePreferredNamespace],
  );

  const onDeleted = React.useCallback(() => {
    setDeleteDeployment(null);
    setSelectedDeployment(null);
    refreshDeployments();
  }, [refreshDeployments]);

  const onDeployed = React.useCallback(
    (deployedNamespace: string) => {
      setDeployTemplate(null);
      const match = namespaces.find((ns) => ns.name === deployedNamespace);
      updatePreferredNamespace(match ?? { name: deployedNamespace });
      setSelectedTemplate(null);
      setSelectedDeployment(null);
      setActiveTabKey(DEPLOYMENTS_TAB);
      refreshDeployments();
    },
    [namespaces, updatePreferredNamespace, refreshDeployments],
  );

  const onTemplateDeleted = React.useCallback(() => {
    setDeleteTemplate(null);
    setSelectedTemplate(null);
    refreshTemplates();
  }, [refreshTemplates]);

  const onTemplateSaved = React.useCallback(
    (template: GuardrailTemplate) => {
      setManageTemplate(null);
      refreshTemplates();
      // Show the saved template in the drawer with fresh contents.
      setSelectedTemplate(template);
    },
    [refreshTemplates],
  );

  const isTemplatesTab = activeTabKey === TEMPLATES_TAB;
  const isDrawerExpanded = isTemplatesTab ? !!selectedTemplate : !!selectedDeployment;

  const panelContent = isTemplatesTab ? (
    <TemplateDrawerPanel
      template={selectedTemplate}
      onClose={() => setSelectedTemplate(null)}
      onEdit={(template) => setManageTemplate({ mode: 'edit', template })}
      onDelete={setDeleteTemplate}
    />
  ) : (
    <DeploymentDrawerPanel
      namespace={namespace}
      name={selectedDeployment}
      onClose={() => setSelectedDeployment(null)}
    />
  );

  return (
    <>
      <Drawer isExpanded={isDrawerExpanded}>
        <DrawerContent panelContent={panelContent}>
          <DrawerContentBody>
            <ApplicationsPage
              title="Guardrails"
              description="Browse guardrail configuration templates and monitor NeMo Guardrails deployments."
              headerAction={
                namespacesLoadError ? null : (
                  <NamespaceSelector
                    namespaces={namespaces}
                    namespace={namespace}
                    onChange={onNamespaceChange}
                    loaded={namespacesLoaded}
                  />
                )
              }
              loaded
              empty={false}
              provideChildrenPadding
              removeChildrenTopPadding
            >
              <Tabs activeKey={activeTabKey} onSelect={onTabSelect} data-testid="guardrails-tabs">
                <Tab
                  eventKey={TEMPLATES_TAB}
                  title={<TabTitleText>Templates</TabTitleText>}
                  data-testid="guardrails-templates-tab"
                >
                  <TemplatesTab
                    templates={templates}
                    templatesLoaded={templatesLoaded}
                    templatesError={templatesError}
                    onUpload={() => setManageTemplate({ mode: 'create' })}
                    onSelect={setSelectedTemplate}
                    onDeploy={setDeployTemplate}
                  />
                </Tab>
                <Tab
                  eventKey={DEPLOYMENTS_TAB}
                  title={<TabTitleText>Deployments</TabTitleText>}
                  data-testid="guardrails-deployments-tab"
                >
                  <DeploymentsTab
                    onSelect={setSelectedDeployment}
                    onTest={setTestDeployment}
                    onDelete={setDeleteDeployment}
                    deployments={deployments}
                    deploymentsLoaded={deploymentsLoaded}
                    deploymentsError={deploymentsError}
                  />
                </Tab>
              </Tabs>
            </ApplicationsPage>
          </DrawerContentBody>
        </DrawerContent>
      </Drawer>
      <DeployGuardrailModal
        template={deployTemplate}
        defaultNamespace={namespace}
        namespaces={namespaces}
        onClose={() => setDeployTemplate(null)}
        onDeployed={onDeployed}
      />
      <DeleteGuardrailModal
        deployment={deleteDeployment}
        onClose={() => setDeleteDeployment(null)}
        onDeleted={onDeleted}
      />
      <TestGuardrailModal deployment={testDeployment} onClose={() => setTestDeployment(null)} />
      <DeleteTemplateModal
        template={deleteTemplate}
        onClose={() => setDeleteTemplate(null)}
        onDeleted={onTemplateDeleted}
      />
      <ManageTemplateModal
        target={manageTemplate}
        defaultNamespace={namespace}
        onClose={() => setManageTemplate(null)}
        onSaved={onTemplateSaved}
      />
    </>
  );
};

export default GuardrailsPage;
