package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/opendatahub-io/ai-safety/bff/internal/constants"
	"github.com/opendatahub-io/ai-safety/bff/internal/integrations/kubernetes"
	"github.com/opendatahub-io/ai-safety/bff/internal/models"
	"github.com/opendatahub-io/ai-safety/bff/internal/repositories"

	"github.com/julienschmidt/httprouter"
)

type GuardrailTemplatesEnvelope Envelope[[]models.GuardrailTemplate, None]
type GuardrailTemplateEnvelope Envelope[models.GuardrailTemplateDetail, None]
type GuardrailDeploymentsEnvelope Envelope[[]models.GuardrailDeployment, None]
type GuardrailDeploymentEnvelope Envelope[models.GuardrailDeploymentDetail, None]
type CreatedGuardrailDeploymentEnvelope Envelope[models.GuardrailDeployment, None]
type ServingModelsEnvelope Envelope[[]models.ServingModel, None]

// GetServingModelsHandler lists the models served in the requested namespace
// that a guardrail deployment can be wired to. Access is gated by
// SelfSubjectAccessReviews for the KServe serving resources; when the user may
// list neither, the request is forbidden.
func (app *App) GetServingModelsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in request context"))
		return
	}

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	canListIS, err := client.CanListInferenceServices(ctx, identity, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	canListLLM, err := client.CanListLLMInferenceServices(ctx, identity, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !canListIS && !canListLLM {
		app.forbiddenResponse(w, r, fmt.Sprintf("user is not allowed to list serving models in namespace %s", namespace))
		return
	}

	servingModels, err := app.repositories.Guardrails.GetServingModels(client, ctx, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := ServingModelsEnvelope{Data: servingModels}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGuardrailTemplatesHandler lists the guardrail configuration templates:
// the platform catalog (read with the BFF's own credentials) plus, when a
// namespace query parameter is given and the user may list ConfigMaps there,
// the user-created templates in that namespace.
func (app *App) GetGuardrailTemplatesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	internalClient, err := app.kubernetesClientFactory.GetInternalClient()
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get internal Kubernetes client: %w", err))
		return
	}

	userNamespace := r.URL.Query().Get(string(constants.NamespaceHeaderParameterKey))
	var client kubernetes.KubernetesClientInterface
	if userNamespace != "" {
		identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
		if !ok || identity == nil {
			app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
			return
		}
		client, err = app.kubernetesClientFactory.GetClient(ctx)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
			return
		}
		allowed, err := client.CanListConfigMaps(ctx, identity, userNamespace)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		if !allowed {
			// The platform catalog remains available; user templates are not.
			userNamespace = ""
		}
	}

	templates, err := app.repositories.Guardrails.GetTemplates(
		internalClient, client, ctx, app.config.GuardrailsTemplateNamespace, userNamespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := GuardrailTemplatesEnvelope{Data: templates}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGuardrailTemplateHandler returns a single guardrail configuration
// template with its full file contents. Platform templates are read with the
// BFF's credentials; user-namespace templates with the requesting user's.
func (app *App) GetGuardrailTemplateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	name := ps.ByName("name")
	namespace := r.URL.Query().Get(string(constants.NamespaceHeaderParameterKey))

	var client kubernetes.KubernetesClientInterface
	var err error
	if namespace != "" && namespace != app.config.GuardrailsTemplateNamespace {
		client, err = app.kubernetesClientFactory.GetClient(ctx)
	} else {
		namespace = app.config.GuardrailsTemplateNamespace
		client, err = app.kubernetesClientFactory.GetInternalClient()
	}
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	template, err := app.repositories.Guardrails.GetTemplate(client, ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := GuardrailTemplateEnvelope{Data: *template}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGuardrailDeploymentsHandler lists the NemoGuardrails deployments in the
// requested namespace. Access is gated by a SelfSubjectAccessReview for the
// nemoguardrails resource in that namespace.
func (app *App) GetGuardrailDeploymentsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in request context"))
		return
	}

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	allowed, err := client.CanListNemoGuardrails(ctx, identity, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !allowed {
		app.forbiddenResponse(w, r, fmt.Sprintf("user is not allowed to list NemoGuardrails in namespace %s", namespace))
		return
	}

	deployments, err := app.repositories.Guardrails.GetDeployments(client, ctx, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := GuardrailDeploymentsEnvelope{Data: deployments}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGuardrailDeploymentHandler returns a single NemoGuardrails deployment
// with its full specification.
func (app *App) GetGuardrailDeploymentHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	name := ps.ByName("name")
	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in request context"))
		return
	}

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	allowed, err := client.CanListNemoGuardrails(ctx, identity, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !allowed {
		app.forbiddenResponse(w, r, fmt.Sprintf("user is not allowed to list NemoGuardrails in namespace %s", namespace))
		return
	}

	deployment, err := app.repositories.Guardrails.GetDeployment(client, ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := GuardrailDeploymentEnvelope{Data: *deployment}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateGuardrailDeploymentHandler deploys a guardrail template into a
// namespace. Access is gated by a SelfSubjectAccessReview covering creation of
// both the ConfigMap copy and the NemoGuardrails custom resource.
func (app *App) CreateGuardrailDeploymentHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	var request models.GuardrailDeploymentCreateRequest
	if err := json.Unmarshal(body, &request); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if request.Name == "" || request.Namespace == "" || request.TemplateName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("name, namespace, and templateName are required"))
		return
	}
	if err := repositories.ValidateDeploymentName(request.Name); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if request.Model != nil {
		if _, err := repositories.ValidateModelWiring(request.Model); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	allowed, err := client.CanCreateGuardrailDeployment(ctx, identity, request.Namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !allowed {
		app.forbiddenResponse(w, r, fmt.Sprintf("user is not allowed to create guardrail deployments in namespace %s", request.Namespace))
		return
	}

	internalClient, err := app.kubernetesClientFactory.GetInternalClient()
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get internal Kubernetes client: %w", err))
		return
	}

	deployment, err := app.repositories.Guardrails.CreateDeploymentFromTemplate(
		client, internalClient, ctx, app.config.GuardrailsTemplateNamespace, request)
	if err != nil {
		switch {
		case apierrors.IsNotFound(err):
			app.notFoundResponse(w, r)
		case apierrors.IsAlreadyExists(err):
			app.conflictResponse(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := CreatedGuardrailDeploymentEnvelope{Data: *deployment}
	if err := app.WriteJSON(w, http.StatusCreated, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DeleteGuardrailDeploymentHandler deletes a NemoGuardrails deployment and the
// dashboard-created ConfigMap copies for it. Access is gated by a
// SelfSubjectAccessReview for deleting NemoGuardrails in the namespace.
func (app *App) DeleteGuardrailDeploymentHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	name := ps.ByName("name")
	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in request context"))
		return
	}

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	allowed, err := client.CanDeleteNemoGuardrails(ctx, identity, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !allowed {
		app.forbiddenResponse(w, r, fmt.Sprintf("user is not allowed to delete NemoGuardrails in namespace %s", namespace))
		return
	}

	if err := app.repositories.Guardrails.DeleteDeployment(client, ctx, namespace, name); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
