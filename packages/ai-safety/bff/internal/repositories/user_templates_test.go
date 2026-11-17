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

const validConfigYAML = "passthrough: true\nrails:\n  input:\n    flows:\n      - mask pii\n"

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
