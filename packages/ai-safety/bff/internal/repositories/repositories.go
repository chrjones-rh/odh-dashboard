package repositories

import "crypto/x509"

// Repositories struct is a single convenient container to hold and represent all our repositories.
type Repositories struct {
	HealthCheck *HealthCheckRepository
	User        *UserRepository
	Namespace   *NamespaceRepository
	Guardrails  *GuardrailsRepository
}

// GuardrailsLocalConfig carries the local-development overrides for reaching
// guardrail services from a BFF running outside the cluster. Both are unset
// (nil receiver / empty fields) in cluster deployments.
type GuardrailsLocalConfig struct {
	// DialMap redirects guardrail service connections to alternate dial
	// addresses, typically `oc port-forward` listeners; the request URL is
	// left untouched so service certificate validation keeps working. See
	// config.GuardrailsDialMap.
	DialMap map[string]string
	// RouteSuffix rewrites guardrail service URLs to the deployments'
	// OpenShift routes ("<name>-<namespace>.<suffix>"), which are reachable
	// from outside the cluster. Takes precedence over DialMap. See
	// config.GuardrailsRouteSuffix.
	RouteSuffix string
	// RouteCAs validates route certificates when RouteSuffix is set; nil
	// uses the system pool. Route certificates are signed by the ingress
	// operator CA, not the service serving CA.
	RouteCAs *x509.CertPool
}

func NewRepositories(local *GuardrailsLocalConfig) *Repositories {
	return &Repositories{
		HealthCheck: NewHealthCheckRepository(),
		User:        NewUserRepository(),
		Namespace:   NewNamespaceRepository(),
		Guardrails:  NewGuardrailsRepository(local),
	}
}
