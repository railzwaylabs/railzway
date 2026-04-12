package authz

import (
	_ "embed"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

//go:embed model/apikey_model.conf
var apiKeyModel string

type APIKeyAuthorizer struct {
	enforcer *casbin.Enforcer
}

func NewAPIKeyAuthorizer(scopes []string) (*APIKeyAuthorizer, error) {
	m, err := model.NewModelFromString(apiKeyModel)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		resource, action := splitScope(scope)
		if resource == "" {
			continue
		}
		if action == "" {
			action = "*"
		}
		if _, err := enforcer.AddPolicy(resource, normalizeScopeAction(action)); err != nil {
			return nil, err
		}
	}
	return &APIKeyAuthorizer{enforcer: enforcer}, nil
}

func (a *APIKeyAuthorizer) Enforce(resource, action string) (bool, error) {
	if a == nil || a.enforcer == nil {
		return false, nil
	}
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" || action == "" {
		return false, nil
	}
	return a.enforcer.Enforce(resource, action)
}

func splitScope(scope string) (string, string) {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func normalizeScopeAction(action string) string {
	trimmed := strings.TrimSpace(action)
	if trimmed == "" || trimmed == "*" {
		return ".*"
	}
	switch strings.ToLower(trimmed) {
	case "read":
		return "GET|HEAD|OPTIONS"
	case "write":
		return "POST|PUT|PATCH|DELETE"
	default:
		return strings.ToUpper(trimmed)
	}
}
