import type { Namespace } from 'mod-arch-core';
import { mockNamespace } from '~/__mocks__/mockNamespace';
import { mockUserSettings } from '~/__mocks__/mockUserSettings';
import {
  mockGuardrailDeployment,
  mockGuardrailDeploymentDetail,
  mockGuardrailTemplate,
  mockGuardrailTemplateDetail,
} from '~/__mocks__/mockGuardrails';
import { guardrailsPage } from '~/__tests__/cypress/cypress/pages/guardrailsPage';
import { CLIENT_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';

type HandlersProps = {
  namespaces?: Namespace[];
  templates?: ReturnType<typeof mockGuardrailTemplate>[];
  userTemplates?: ReturnType<typeof mockGuardrailTemplate>[];
  deployments?: ReturnType<typeof mockGuardrailDeployment>[];
};

const TEST_NAMESPACE = 'test-namespace';

const initIntercepts = ({
  namespaces = [mockNamespace({ name: TEST_NAMESPACE })],
  userTemplates = [
    mockGuardrailTemplate({
      name: 'my-custom-guardrail',
      namespace: TEST_NAMESPACE,
      keys: ['config.yaml'],
      summary: {
        modelCount: 1,
        models: ['openai/placeholder (main)'],
        railFlows: [],
        promptCount: 0,
        passthrough: false,
      },
      source: 'user',
    }),
  ],
  templates = [
    mockGuardrailTemplate({}),
    mockGuardrailTemplate({
      name: 'trustyai-service-operator-nemo-guardrails-default-injection',
      keys: ['actions.py', 'config.yaml'],
      summary: {
        modelCount: 0,
        models: [],
        railFlows: [{ category: 'input', flow: 'injection detection' }],
        promptCount: 0,
        passthrough: true,
      },
    }),
  ],
  deployments = [mockGuardrailDeployment({})],
}: HandlersProps = {}) => {
  cy.interceptApi(
    'GET /api/:apiVersion/user',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
      },
    },
    mockUserSettings({ userId: 'test-user' }),
  );

  cy.interceptApi(
    'GET /api/:apiVersion/namespaces',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
      },
    },
    namespaces,
  );

  // Platform-only catalog (no namespace selected yet)
  cy.interceptApi(
    'GET /api/:apiVersion/guardrails/templates',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
      },
    },
    templates,
  );

  // Merged catalog once a namespace is selected: platform templates plus the
  // user-created templates in the selected namespace.
  cy.interceptApi(
    'GET /api/:apiVersion/guardrails/templates',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
      },
      query: {
        namespace: TEST_NAMESPACE,
      },
    },
    [...templates, ...userTemplates],
  );

  cy.interceptApi(
    'GET /api/:apiVersion/guardrails/templates/:name',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
        name: 'trustyai-service-operator-nemo-guardrails-default-pii',
      },
    },
    mockGuardrailTemplateDetail({}),
  );

  cy.interceptApi(
    'GET /api/:apiVersion/guardrails/deployments',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
      },
      query: {
        namespace: TEST_NAMESPACE,
      },
    },
    deployments,
  );

  cy.interceptApi(
    'GET /api/:apiVersion/guardrails/deployments/:name',
    {
      path: {
        apiVersion: CLIENT_API_VERSION,
        name: 'my-guardrails',
      },
      query: {
        namespace: TEST_NAMESPACE,
      },
    },
    mockGuardrailDeploymentDetail({}),
  );
};

describe('Guardrails', () => {
  it('should list the guardrail templates', () => {
    initIntercepts({});
    guardrailsPage.visit();

    const row = guardrailsPage.findTemplateRow('default-pii');
    row.should('contain', 'config.yaml');
    row
      .findByTestId('guardrails-template-description-default-pii')
      .should('have.text', 'A guardrail that masks personally identifiable information.');
    guardrailsPage.findTemplateRow('default-injection').should('exist');
  });

  it('should persist the selected namespace across reloads', () => {
    initIntercepts({
      namespaces: [
        mockNamespace({ name: TEST_NAMESPACE }),
        mockNamespace({ name: 'other-namespace' }),
      ],
    });
    guardrailsPage.visit();

    // Switch to the other namespace via the header selector
    cy.findByTestId('guardrails-namespace-select').click();
    cy.findByRole('option', { name: 'other-namespace' }).click();
    cy.findByTestId('guardrails-namespace-select').should('contain', 'other-namespace');

    // The selection survives a page reload
    cy.reload();
    guardrailsPage.visit();
    cy.findByTestId('guardrails-namespace-select').should('contain', 'other-namespace');
  });

  it('should show user-created templates for the selected namespace', () => {
    initIntercepts({});
    guardrailsPage.visit();

    // The namespace defaults and the merged catalog is fetched; the user
    // template appears with its namespace as the origin tag.
    guardrailsPage
      .findTemplateRow('my-custom-guardrail')
      .findByTestId('guardrails-template-origin-my-custom-guardrail')
      .should('have.text', TEST_NAMESPACE);
  });

  it('should filter templates by search', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findTemplateSearch().type('injection');
    guardrailsPage.findTemplateRow('default-pii').should('not.exist');
    guardrailsPage.findTemplateRow('default-injection').should('exist');
  });

  it('should show an empty state when no templates exist', () => {
    initIntercepts({ templates: [], userTemplates: [] });
    guardrailsPage.visit();

    guardrailsPage.findTemplatesEmpty().should('exist');
  });

  it('should open the template drawer with the configuration files', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findTemplateRow('default-pii').click();
    guardrailsPage.findTemplateDrawer().findByText('config.yaml').should('exist');
    guardrailsPage.findTemplateDrawer().findByText('rails.co').should('exist');
  });

  it('should list the deployments in the selected namespace', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findDeploymentsTabButton().click();
    guardrailsPage.findDeploymentRow('my-guardrails').should('exist');
    guardrailsPage.findDeploymentStatus('my-guardrails').should('have.text', 'Ready');
  });

  it('should show an empty state when no deployments exist', () => {
    initIntercepts({ deployments: [] });
    guardrailsPage.visit();

    guardrailsPage.findDeploymentsTabButton().click();
    guardrailsPage.findDeploymentsEmpty().should('exist');
  });

  it('should open the deployment drawer with the deployment details', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findDeploymentsTabButton().click();
    guardrailsPage.findDeploymentRow('my-guardrails').click();
    guardrailsPage.findDeploymentDrawer().should('exist');
    guardrailsPage
      .findDeploymentDrawer()
      .findByTestId('guardrails-deployment-detail-namespace')
      .should('have.text', TEST_NAMESPACE);
    guardrailsPage.findDeploymentDrawer().findByText('OPENAI_API_KEY').should('exist');
  });

  it('should deploy a template and switch to the deployments tab', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findDeployButton('default-pii').click();
    guardrailsPage.findDeployModal().should('exist');
    guardrailsPage.findDeployNameInput().type('my-new-guardrail');

    // Register the POST response and the refreshed deployments list including
    // the new deployment before submitting.
    cy.interceptApi(
      'POST /api/:apiVersion/guardrails/deployments',
      {
        path: {
          apiVersion: CLIENT_API_VERSION,
        },
      },
      mockGuardrailDeployment({
        name: 'my-new-guardrail',
        configSources: [
          {
            name: 'guardrail-my-new-guardrail',
            found: true,
            keys: ['config.yaml'],
            hasModelSpec: false,
            isOperatorTemplate: false,
          },
        ],
      }),
    ).as('createDeployment');

    cy.interceptApi(
      'GET /api/:apiVersion/guardrails/deployments',
      {
        path: {
          apiVersion: CLIENT_API_VERSION,
        },
        query: {
          namespace: TEST_NAMESPACE,
        },
      },
      [
        mockGuardrailDeployment({}),
        mockGuardrailDeployment({
          name: 'my-new-guardrail',
          phase: '',
          ready: false,
          endpoint: undefined,
          configSources: [
            {
              name: 'guardrail-my-new-guardrail',
              found: true,
              keys: ['config.yaml'],
              hasModelSpec: false,
              isOperatorTemplate: false,
            },
          ],
        }),
      ],
    );

    // Wait for the submit button to become enabled (the page namespace may
    // still be defaulting when the modal opens).
    guardrailsPage.findDeploySubmitButton().should('be.enabled');
    guardrailsPage.findDeploySubmitButton().click();

    cy.wait('@createDeployment').then((interception) => {
      expect(interception.request.body).to.eql({
        name: 'my-new-guardrail',
        namespace: TEST_NAMESPACE,
        templateName: 'trustyai-service-operator-nemo-guardrails-default-pii',
        templateNamespace: 'redhat-ods-applications',
      });
    });

    // The page switches to the deployments tab and the new deployment appears
    // with a pending status while the operator reconciles it.
    guardrailsPage.findDeploymentsTable().should('exist');
    guardrailsPage.findDeploymentRow('my-new-guardrail').should('exist');
    guardrailsPage.findDeploymentStatus('my-new-guardrail').should('have.text', 'Pending');
  });

  it('should delete a deployment from the deployments tab', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findDeploymentsTabButton().click();
    guardrailsPage.findDeploymentRow('my-guardrails').should('exist');

    cy.findByTestId('guardrails-deployment-delete-my-guardrails').click();
    cy.findByTestId('delete-guardrail-modal').should('exist');

    // Reply with a true 204 No Content (empty body) like the real BFF, so the
    // frontend's response handling is exercised faithfully.
    cy.intercept(
      {
        method: 'DELETE',
        pathname: '/ai-safety/api/v1/guardrails/deployments/my-guardrails',
        query: { namespace: TEST_NAMESPACE },
      },
      { statusCode: 204, body: '' },
    ).as('deleteDeployment');

    // The refreshed list no longer includes the deleted deployment
    cy.interceptApi(
      'GET /api/:apiVersion/guardrails/deployments',
      {
        path: {
          apiVersion: CLIENT_API_VERSION,
        },
        query: {
          namespace: TEST_NAMESPACE,
        },
      },
      [],
    );

    cy.findByTestId('delete-guardrail-submit-button').click();
    cy.wait('@deleteDeployment');
    guardrailsPage.findDeploymentRow('my-guardrails').should('not.exist');
    guardrailsPage.findDeploymentsEmpty().should('exist');
  });

  it('should disable the deploy submit button for an invalid name', () => {
    initIntercepts({});
    guardrailsPage.visit();

    guardrailsPage.findDeployButton('default-pii').click();
    guardrailsPage.findDeployModal().should('exist');
    guardrailsPage.findDeployNameInput().type('Invalid Name');
    guardrailsPage.findDeploySubmitButton().should('be.disabled');
  });
});
