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
}

export const guardrailsPage = new GuardrailsPage();
