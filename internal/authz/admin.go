package authz

import (
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"gorm.io/gorm"
)

const adminModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && regexMatch(r.obj, p.obj) && regexMatch(r.act, p.act)
`

type AdminAuthorizer struct {
	enforcer *casbin.Enforcer
}

type Policy struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

func NewAdminAuthorizer(db *gorm.DB) (*AdminAuthorizer, error) {
	m, err := model.NewModelFromString(adminModel)
	if err != nil {
		return nil, err
	}
	var enforcer *casbin.Enforcer
	if db != nil {
		adapter, err := gormadapter.NewAdapterByDB(db)
		if err != nil {
			return nil, err
		}
		enforcer, err = casbin.NewEnforcer(m, adapter)
		if err != nil {
			return nil, err
		}
		if err := enforcer.LoadPolicy(); err != nil {
			return nil, err
		}
		if err := ensureAdminPolicies(enforcer); err != nil {
			return nil, err
		}
	} else {
		enforcer, err = casbin.NewEnforcer(m)
		if err != nil {
			return nil, err
		}
		if err := ensureAdminPolicies(enforcer); err != nil {
			return nil, err
		}
	}
	return &AdminAuthorizer{enforcer: enforcer}, nil
}

func (a *AdminAuthorizer) Enforce(role, path, method string) (bool, error) {
	if a == nil || a.enforcer == nil {
		return true, nil
	}
	role = normalizeRole(role)
	if role == "" {
		role = organizationdomain.RoleMember
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return false, nil
	}
	return a.enforcer.Enforce(role, path, method)
}

func normalizeRole(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}

func (a *AdminAuthorizer) ListPolicies() ([]Policy, error) {
	if a == nil || a.enforcer == nil {
		return []Policy{}, nil
	}
	rows, err := a.enforcer.GetPolicy()
	if err != nil {
		return nil, err
	}
	out := make([]Policy, 0, len(rows))
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		out = append(out, Policy{
			Subject: row[0],
			Object:  row[1],
			Action:  row[2],
		})
	}
	return out, nil
}

func (a *AdminAuthorizer) AddPolicy(policy Policy) (bool, error) {
	if a == nil || a.enforcer == nil {
		return false, nil
	}
	sub := strings.TrimSpace(policy.Subject)
	obj := strings.TrimSpace(policy.Object)
	act := strings.TrimSpace(policy.Action)
	if sub == "" || obj == "" || act == "" {
		return false, nil
	}
	exists, err := a.enforcer.HasPolicy(sub, obj, act)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	added, err := a.enforcer.AddPolicy(sub, obj, act)
	if err != nil {
		return false, err
	}
	if added {
		if err := a.enforcer.SavePolicy(); err != nil {
			return false, err
		}
	}
	return added, nil
}

func (a *AdminAuthorizer) RemovePolicy(policy Policy) (bool, error) {
	if a == nil || a.enforcer == nil {
		return false, nil
	}
	sub := strings.TrimSpace(policy.Subject)
	obj := strings.TrimSpace(policy.Object)
	act := strings.TrimSpace(policy.Action)
	if sub == "" || obj == "" || act == "" {
		return false, nil
	}
	removed, err := a.enforcer.RemovePolicy(sub, obj, act)
	if err != nil {
		return false, err
	}
	if removed {
		if err := a.enforcer.SavePolicy(); err != nil {
			return false, err
		}
	}
	return removed, nil
}

func ensureAdminPolicies(enforcer *casbin.Enforcer) error {
	if enforcer == nil {
		return nil
	}
	adminPatterns := []string{
		`^/admin(/v1)?/.*$`,
	}
	policies := [][]string{}
	for _, role := range []string{
		organizationdomain.RoleOwner,
		organizationdomain.RoleAdmin,
	} {
		for _, pattern := range adminPatterns {
			policies = append(policies, []string{role, pattern, ".*"})
		}
	}
	for _, role := range []string{
		organizationdomain.RoleMember,
		organizationdomain.RoleDeveloper,
		organizationdomain.RoleFinOps,
	} {
		for _, pattern := range adminPatterns {
			policies = append(policies, []string{role, pattern, "GET|HEAD|OPTIONS"})
		}
	}
	changed := false
	for _, policy := range policies {
		ok, err := enforcer.HasPolicy(policy)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if _, err := enforcer.AddPolicy(policy); err != nil {
			return err
		}
		changed = true
	}
	if changed {
		return enforcer.SavePolicy()
	}
	return nil
}
