class GuardrailsPage {
  visit() {
    cy.visit('/guardrails');
    this.wait();
  }

  private wait() {
    cy.findByTestId('guardrails-tabs').should('exist');
    cy.testA11y();
  }

  findTemplatesTab() {
    return cy.findByTestId('guardrails-templates-tab');
  }

  findDeploymentsTabButton() {
    return cy.findByRole('tab', { name: 'Deployments' });
  }

  findTemplatesTable() {
    return cy.findByTestId('guardrails-templates-table');
  }

  findDeploymentsTable() {
    return cy.findByTestId('guardrails-deployments-table');
  }

  findTemplateSearch() {
    return cy.findByTestId('guardrails-templates-search');
  }

  findTemplatesEmpty() {
    return cy.findByTestId('guardrails-templates-empty');
  }

  findDeploymentsEmpty() {
    return cy.findByTestId('guardrails-deployments-empty');
  }

  findTemplateRow(name: string) {
    return cy.findByTestId(`guardrails-template-row-${name}`);
  }

  findDeployButton(name: string) {
    return cy.findByTestId(`guardrails-template-deploy-${name}`);
  }

  findDeployModal() {
    return cy.findByTestId('deploy-guardrail-modal');
  }

  findDeployNameInput() {
    return cy.findByTestId('guardrail-deployment-name-input');
  }

  findDeploySubmitButton() {
    return cy.findByTestId('guardrail-deployment-submit-button');
  }

  findDeployModelSelect() {
    return cy.findByTestId('guardrail-deployment-model-select');
  }

  findDeployModelNameInput() {
    return cy.findByTestId('guardrail-deployment-model-name-input');
  }

  findDeployModelEndpointInput() {
    return cy.findByTestId('guardrail-deployment-model-endpoint-input');
  }

  findTestModal() {
    return cy.findByTestId('test-guardrail-modal');
  }

  findTestMessageInput() {
    return cy.findByTestId('guardrail-test-message-input');
  }

  findTestRunButton() {
    return cy.findByTestId('guardrail-test-run-button');
  }

  findTestResult() {
    return cy.findByTestId('guardrail-test-result');
  }

  findTestVerdict() {
    return cy.findByTestId('guardrail-test-verdict');
  }

  findTestButton(name: string) {
    return cy.findByTestId(`guardrails-deployment-test-${name}`);
  }

  findDeploymentRow(name: string) {
    return cy.findByTestId(`guardrails-deployment-row-${name}`);
  }

  findTemplateDrawer() {
    return cy.findByTestId('guardrails-template-drawer');
  }

  findDeploymentDrawer() {
    return cy.findByTestId('guardrails-deployment-drawer');
  }

  findDeploymentStatus(name: string) {
    return cy.findByTestId(`guardrails-deployment-status-${name}`);
  }

  findNamespaceSelect() {
    return cy.findByTestId('guardrails-namespace-select');
  }

  findUploadTemplateButton() {
    return cy.findByTestId('guardrails-templates-upload-button');
  }

  findManageTemplateModal() {
    return cy.findByTestId('manage-template-modal');
  }

  findTemplateNameInput() {
    return cy.findByTestId('guardrail-template-name-input');
  }

  findTemplateDescriptionInput() {
    return cy.findByTestId('guardrail-template-description-input');
  }

  findTemplateSubmitButton() {
    return cy.findByTestId('guardrail-template-submit-button');
  }

  findTemplateEditButton(name: string) {
    return cy.findByTestId(`guardrails-template-edit-${name}`);
  }

  findTemplateConfigEditor() {
    return cy.findByTestId('guardrail-template-config-editor');
  }

  /**
   * Waits for the config editor to render. With empty content the PF
   * CodeEditor shows its empty state instead of the monaco surface, so this
   * only waits for the editor container.
   */
  waitForTemplateConfigEditor() {
    this.findTemplateConfigEditor().should('exist');
  }

  /**
   * Sets the config editor content by uploading it through the code editor's
   * built-in file input; the change round-trips through onCodeChange.
   */
  setTemplateConfigYAML(value: string) {
    this.findTemplateConfigEditor()
      .find('input[type="file"]')
      .selectFile(
        {
          contents: Cypress.Buffer.from(value),
          fileName: 'config.yaml',
          mimeType: 'text/yaml',
        },
        { force: true },
      );
  }

  /** Reads the config editor content from the Monaco model. */
  getTemplateConfigYAML(): Cypress.Chainable<string> {
    return this.findTemplateConfigEditor().then(($container) =>
      cy.window().then((win) => {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const { monaco } = win as any;
        if (!monaco) {
          throw new Error('Monaco editor was not found on window');
        }
        const editors = monaco.editor
          .getEditors()
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          .filter((editor: any) => {
            const node = editor.getDomNode() as HTMLElement | null;
            return node !== null && $container[0].contains(node);
          });
        expect(editors, 'one Monaco editor in the modal').to.have.length(1);
        const model = editors[0].getModel();
        if (!model) {
          throw new Error('The code editor has no Monaco model');
        }
        return model.getValue() as string;
      }),
    );
  }
}

export const guardrailsPage = new GuardrailsPage();
