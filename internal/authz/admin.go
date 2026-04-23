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
		role = organizationdomain.RoleCustomerSupport
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
	policies := [][]string{}
	// OWNER and ADMIN get full access.
	for _, role := range []string{
		organizationdomain.RoleOwner,
		organizationdomain.RoleAdmin,
	} {
		policies = append(policies, []string{role, `^/admin(/v1)?/.*$`, ".*"})
	}

	// FINANCE: Write access to money-related modules.
	policies = append(policies, []string{organizationdomain.RoleFinance, `^/admin(/v1)?/(invoices|ledger|taxes|payments)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleFinance, `^/admin(/v1)?/(reconciliation|coupons|promotion-codes|segments)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleFinance, `^/admin(/v1)?/.*$`, "GET|HEAD|OPTIONS"})

	// OPERATIONS: Write access to catalog and customer lifecycle.
	policies = append(policies, []string{organizationdomain.RoleOperations, `^/admin(/v1)?/(products|plans|prices|features)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleOperations, `^/admin(/v1)?/(customers|subscriptions)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleOperations, `^/admin(/v1)?/(coupons|promotion-codes|segments)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleOperations, `^/admin(/v1)?/.*$`, "GET|HEAD|OPTIONS"})

	// DEVELOPER: Write access to technical integration and tools.
	policies = append(policies, []string{organizationdomain.RoleDeveloper, `^/admin(/v1)?/(meters|apikeys|webhooks|ai|test-clock|feature-flags|warnings)(/.*)?$`, ".*"})
	policies = append(policies, []string{organizationdomain.RoleDeveloper, `^/admin(/v1)?/.*$`, "GET|HEAD|OPTIONS"})

	// CUSTOMER_SUPPORT: Targeted Read-only access to customer-facing data.
	policies = append(policies, []string{organizationdomain.RoleCustomerSupport, `^/admin(/v1)?/(customers|subscriptions|invoices|products|plans|features|usage|rating)(/.*)?$`, "GET|HEAD|OPTIONS"})

	// AUDITOR: Universal Read-only access for compliance.
	policies = append(policies, []string{organizationdomain.RoleAuditor, `^/admin(/v1)?/.*$`, "GET|HEAD|OPTIONS"})

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
