package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/opendatahub-io/ai-safety/bff/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const validConfigYAML = "passthrough: true\nrails:\n  input:\n    flows:\n      - mask sensitive data on input\n"

func TestCreateTemplate(t *testing.T) {
	client := &fakeGuardrailsClient{}
	repo := NewGuardrailsRepository(nil)

	template, err := repo.CreateTemplate(client, context.Background(), models.GuardrailTemplateUpsertRequest{
		Name:        "my-template",
		Namespace:   "test-ns",
		Description: "  My custom guardrail  ",
		ConfigYAML:  validConfigYAML,
	})
	require.NoError(t, err)

	assert.Equal(t, "my-template", template.Name)
	assert.Equal(t, "test-ns", template.Namespace)
	assert.Equal(t, "user", template.Source)
	assert.Equal(t, "My custom guardrail", template.Description)
	assert.Equal(t, []string{"config.yaml"}, template.Keys)

	require.Len(t, client.createdConfigMaps, 1)
	created := client.createdConfigMaps[0]
	assert.Equal(t, "true", created.Labels["nemo-guardrails-config"])
	assert.Equal(t, "true", created.Labels["opendatahub.io/dashboard"])
	assert.Equal(t, "My custom guardrail", created.Annotations["guardrail-description"])
	assert.Equal(t, validConfigYAML, created.Data["config.yaml"])
}

func TestCreateTemplateInvalidName(t *testing.T) {
	client := &fakeGuardrailsClient{}
	repo := NewGuardrailsRepository(nil)

	_, err := repo.CreateTemplate(client, context.Background(), models.GuardrailTemplateUpsertRequest{
		Name:       "Invalid Name",
		Namespace:  "test-ns",
		ConfigYAML: validConfigYAML,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template name")
}

func TestCreateTemplateInvalidYAML(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
	}{
		{name: "empty", configYAML: "  "},
		{name: "unparsable", configYAML: "\tinvalid: [yaml"},
		{name: "not a mapping", configYAML: "- just\n- a\n- list\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeGuardrailsClient{}
			repo := NewGuardrailsRepository(nil)

			_, err := repo.CreateTemplate(client, context.Background(), models.GuardrailTemplateUpsertRequest{
				Name:       "my-template",
				Namespace:  "test-ns",
				ConfigYAML: tt.configYAML,
			})
			require.Error(t, err)
		})
	}
}

func dashboardManagedUserTemplate() *fakeGuardrailsClient {
	return &fakeGuardrailsClient{
		configMaps: []corev1.ConfigMap{
			testConfigMap("test-ns", "my-template", map[string]string{
				"nemo-guardrails-config":   "true",
				"opendatahub.io/dashboard": "true",
			}, map[string]string{
				"config.yaml": validConfigYAML,
				"rails.co":    "define flow",
			}),
		},
	}
}

func TestUpdateTemplate(t *testing.T) {
	client := dashboardManagedUserTemplate()
	repo := NewGuardrailsRepository(nil)

	template, err := repo.UpdateTemplate(client, context.Background(), "test-ns", "my-template", models.GuardrailTemplateUpsertRequest{
		Description: "Updated description",
		ConfigYAML:  "passthrough: false\n",
	})
	require.NoError(t, err)
	assert.Equal(t, "user", template.Source)
	assert.Equal(t, "Updated description", template.Description)

	// The other ConfigMap keys survive the update.
	updated := client.configMaps[0]
	assert.Equal(t, "passthrough: false\n", updated.Data["config.yaml"])
	assert.Equal(t, "define flow", updated.Data["rails.co"])
}

func TestUpdateTemplateNotDashboardManaged(t *testing.T) {
	client := &fakeGuardrailsClient{
		configMaps: []corev1.ConfigMap{
			testConfigMap("test-ns", "hand-made", map[string]string{
				"nemo-guardrails-config": "true",
			}, map[string]string{"config.yaml": validConfigYAML}),
		},
	}
	repo := NewGuardrailsRepository(nil)

	_, err := repo.UpdateTemplate(client, context.Background(), "test-ns", "hand-made", models.GuardrailTemplateUpsertRequest{
		ConfigYAML: "passthrough: false\n",
	})
	require.Error(t, err)
	var notManaged *NotDashboardManagedError
	require.True(t, errors.As(err, &notManaged))
}

func TestUpdateTemplateNotFound(t *testing.T) {
	client := &fakeGuardrailsClient{}
	repo := NewGuardrailsRepository(nil)

	_, err := repo.UpdateTemplate(client, context.Background(), "test-ns", "missing", models.GuardrailTemplateUpsertRequest{
		ConfigYAML: "passthrough: false\n",
	})
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestValidateTemplateRails(t *testing.T) {
	tests := []struct {
		name          string
		configYAML    string
		expectedError string
	}{
		{
			name: "valid flows with parameters pass",
			configYAML: `passthrough: true
rails:
  input:
    flows:
      - hf classifier check input $classifier="prompt-injection"
  output:
    flows:
      - mask sensitive data on output
  config:
    hf_classifier:
      prompt-injection:
        engine: local
`,
		},
		{
			name:       "passthrough-only configs pass",
			configYAML: "passthrough: true\n",
		},
		{
			name: "unknown flow is rejected with the known list",
			configYAML: `rails:
  input:
    flows:
      - hf classifier check windowed input
`,
			expectedError: `unknown input rail flow "hf classifier check windowed input"`,
		},
		{
			name: "unknown output flow is rejected",
			configYAML: `rails:
  output:
    flows:
      - does not exist
`,
			expectedError: `unknown output rail flow "does not exist"`,
		},
		{
			name: "unknown rails.config key is rejected",
			configYAML: `rails:
  config:
    hf_classifiers:
      typo: true
`,
			expectedError: `unknown rails.config key "hf_classifiers"`,
		},
		{
			name: "self check input without a prompt is rejected",
			configYAML: `rails:
  input:
    flows:
      - self check input
`,
			expectedError: `rail flow "self check input" requires a prompts entry with task "self_check_input"`,
		},
		{
			name: "self check input with a prompt passes",
			configYAML: `rails:
  input:
    flows:
      - self check input
prompts:
  - task: self_check_input
    content: "Check {{ user_input }}"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, validateTemplateConfigYAML(tt.configYAML))
			err := validateTemplateRails(tt.configYAML)
			if tt.expectedError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestDeleteTemplate(t *testing.T) {
	userTemplate := testConfigMap("test-ns", "my-template",
		map[string]string{"nemo-guardrails-config": "true", "opendatahub.io/dashboard": "true"},
		map[string]string{"config.yaml": validConfigYAML})
	operatorTemplate := testConfigMap("redhat-ods-applications", "operator-template",
		map[string]string{"nemo-guardrails-config": "true", "platform.opendatahub.io/part-of": "trustyai-module-operator"},
		map[string]string{"config.yaml": validConfigYAML})
	handWritten := testConfigMap("test-ns", "hand-written",
		map[string]string{"nemo-guardrails-config": "true"},
		map[string]string{"config.yaml": validConfigYAML})
	deploymentCopy := testConfigMap("test-ns", "guardrail-my-deployment",
		map[string]string{"nemo-guardrails-config": "true", "opendatahub.io/dashboard": "true", "opendatahub.io/deployment-config": "true"},
		map[string]string{"config.yaml": validConfigYAML})
	deploymentCopy.Annotations = map[string]string{
		"opendatahub.io/template-source": "redhat-ods-applications/some-template",
	}

	repo := NewGuardrailsRepository(nil)

	t.Run("deletes a user template", func(t *testing.T) {
		client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{userTemplate}}
		require.NoError(t, repo.DeleteTemplate(client, context.Background(), "test-ns", "my-template"))
		require.Len(t, client.configMaps, 0)
	})

	t.Run("missing template returns not found", func(t *testing.T) {
		client := &fakeGuardrailsClient{}
		err := repo.DeleteTemplate(client, context.Background(), "test-ns", "missing")
		require.Error(t, err)
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("operator templates are rejected", func(t *testing.T) {
		client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{operatorTemplate}}
		err := repo.DeleteTemplate(client, context.Background(), "redhat-ods-applications", "operator-template")
		var notManaged *NotDashboardManagedError
		require.ErrorAs(t, err, &notManaged)
		require.Len(t, client.configMaps, 1)
	})

	t.Run("hand-written config maps are rejected", func(t *testing.T) {
		client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{handWritten}}
		err := repo.DeleteTemplate(client, context.Background(), "test-ns", "hand-written")
		var notManaged *NotDashboardManagedError
		require.ErrorAs(t, err, &notManaged)
		require.Len(t, client.configMaps, 1)
	})

	t.Run("deployment configuration copies are rejected", func(t *testing.T) {
		client := &fakeGuardrailsClient{configMaps: []corev1.ConfigMap{deploymentCopy}}
		err := repo.DeleteTemplate(client, context.Background(), "test-ns", "guardrail-my-deployment")
		var deploymentConfig *IsDeploymentConfigError
		require.ErrorAs(t, err, &deploymentConfig)
		assert.Contains(t, err.Error(), "delete its guardrail deployment instead")
		require.Len(t, client.configMaps, 1)
	})
}
