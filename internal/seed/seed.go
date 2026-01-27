package seed

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gosimple/slug"
	authdomain "github.com/railzwaylabs/railzway/internal/auth/domain"
	"github.com/railzwaylabs/railzway/internal/auth/password"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	invoicetemplatedomain "github.com/railzwaylabs/railzway/internal/invoicetemplate/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"gorm.io/gorm"
)

const (
	defaultOrgName       = "Main"
	defaultOrgSlug       = "main"
	defaultAdminEmail    = "admin@railzway.com"
	defaultAdminPassword = "admin"
	defaultAdminDisplay  = "Railzway Admin"
)

// EnsureMainOrg seeds the default organization for startup bootstrap.
func EnsureMainOrg(db *gorm.DB) error {
	if db == nil {
		return errors.New("seed database handle is required")
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org, err := ensureMainOrgTx(ctx, tx, node, nil)
		if err != nil {
			return err
		}
		if _, err := ensureInvoiceSequenceTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		if _, err := ensureInvoiceTemplateTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		return ensureLedgerAccounts(ctx, tx, node, org.ID)
	})
}

// EnsureMainOrgWithID seeds the default organization using a fixed ID.
func EnsureMainOrgWithID(db *gorm.DB, orgID int64) error {
	if db == nil {
		return errors.New("seed database handle is required")
	}
	if orgID == 0 {
		return errors.New("seed org id is required")
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}

	targetID := snowflake.ID(orgID)
	ctx := context.Background()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org, err := ensureMainOrgTx(ctx, tx, node, &targetID)
		if err != nil {
			return err
		}
		if _, err := ensureInvoiceSequenceTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		if _, err := ensureInvoiceTemplateTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		return ensureLedgerAccounts(ctx, tx, node, org.ID)
	})
}

// EnsureMainOrgAndAdmin seeds the default organization and admin user for OSS mode.
func EnsureMainOrgAndAdmin(db *gorm.DB) error {
	return EnsureOrgAndAdminWithOptions(db, OrgSeedOptions{})
}

// OrgSeedOptions allows customizing org identity for bootstrap.
type OrgSeedOptions struct {
	OrgID int64
	Name  string
	Slug  string
	// AdminEmail and AdminPassword override the default admin credentials.
	AdminEmail    string
	AdminPassword string
	// CreateAdminUser controls whether the admin user is created.
	// Defaults to true when nil.
	CreateAdminUser *bool
}

// EnsureOrgAndAdminWithOptions seeds a default organization and admin user using provided overrides.
func EnsureOrgAndAdminWithOptions(db *gorm.DB, opts OrgSeedOptions) error {
	if db == nil {
		return errors.New("seed database handle is required")
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}

	ctx := context.Background()
	overrideID := resolveOrgID(opts.OrgID)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org, err := ensureOrgTx(ctx, tx, node, overrideID, opts.Name, opts.Slug)
		if err != nil {
			return err
		}

		if shouldCreateAdmin(opts.CreateAdminUser) {
			adminEmail, adminPassword := resolveAdminCredentials(opts)
			user, err := ensureAdminUser(ctx, tx, node, adminEmail, adminPassword)
			if err != nil {
				return err
			}
			if err := ensureOrgMembership(ctx, tx, node, org.ID, user.ID); err != nil {
				return err
			}
		}

		if _, err := ensureInvoiceSequenceTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		if _, err := ensureInvoiceTemplateTx(ctx, tx, node, org.ID); err != nil {
			return err
		}
		return ensureLedgerAccounts(ctx, tx, node, org.ID)
	})
}

func resolveOrgID(raw int64) *snowflake.ID {
	if raw == 0 {
		return nil
	}
	id := snowflake.ID(raw)
	return &id
}

func shouldCreateAdmin(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
}

func resolveAdminCredentials(opts OrgSeedOptions) (string, string) {
	email := strings.TrimSpace(opts.AdminEmail)
	if email == "" {
		email = defaultAdminEmail
	}
	email = strings.ToLower(email)

	password := opts.AdminPassword
	if strings.TrimSpace(password) == "" {
		password = defaultAdminPassword
	}

	return email, password
}

func ensureAdminUser(ctx context.Context, tx *gorm.DB, node *snowflake.Node, email string, passwordRaw string) (*authdomain.User, error) {
	var user authdomain.User
	err := tx.WithContext(ctx).
		Where("provider = ? AND external_id = ?", "local", email).
		First(&user).Error
	if err == nil {
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashed, err := password.Hash(passwordRaw)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user = authdomain.User{
		ID:                  node.Generate(),
		ExternalID:          email,
		Provider:            "local",
		DisplayName:         defaultAdminDisplay,
		Email:               email,
		PasswordHash:        &hashed,
		LastPasswordChanged: nil,
		IsDefault:           true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ensureOrgMembership(ctx context.Context, tx *gorm.DB, node *snowflake.Node, orgID snowflake.ID, userID snowflake.ID) error {
	var member organizationdomain.OrganizationMember
	err := tx.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&member).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := time.Now().UTC()
	member = organizationdomain.OrganizationMember{
		ID:        node.Generate(),
		OrgID:     orgID,
		UserID:    userID,
		Role:      organizationdomain.RoleOwner,
		CreatedAt: now,
	}
	return tx.WithContext(ctx).Create(&member).Error
}

func ensureMainOrgTx(ctx context.Context, tx *gorm.DB, node *snowflake.Node, orgID *snowflake.ID) (organizationdomain.Organization, error) {
	return ensureOrgTx(ctx, tx, node, orgID, defaultOrgName, defaultOrgSlug)
}

func ensureOrgTx(ctx context.Context, tx *gorm.DB, node *snowflake.Node, orgID *snowflake.ID, name string, slugValue string) (organizationdomain.Organization, error) {
	var org organizationdomain.Organization
	if orgID != nil && *orgID != 0 {
		if err := tx.WithContext(ctx).First(&org, "id = ?", *orgID).Error; err == nil {
			return org, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return org, err
		}
	}
	if strings.TrimSpace(name) == "" {
		name = defaultOrgName
	}
	if strings.TrimSpace(slugValue) == "" {
		slugValue = slug.Make(name)
	}

	err := tx.WithContext(ctx).Where("slug = ?", slugValue).First(&org).Error
	if err == nil {
		if orgID != nil && *orgID != 0 && org.ID != *orgID {
			return org, errors.New("default org id does not match existing org")
		}
		return org, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return org, err
	}
	now := time.Now().UTC()
	id := node.Generate()
	if orgID != nil && *orgID != 0 {
		id = *orgID
	}
	org = organizationdomain.Organization{
		ID:        id,
		Name:      name,
		Slug:      slugValue,
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.WithContext(ctx).Create(&org).Error; err != nil {
		return org, err
	}
	return org, nil
}

func ensureInvoiceSequenceTx(ctx context.Context, tx *gorm.DB, node *snowflake.Node, orgID snowflake.ID) (invoicedomain.InvoiceSequence, error) {
	var seq invoicedomain.InvoiceSequence
	err := tx.WithContext(ctx).Where("org_id = ?", orgID).First(&seq).Error
	if err == nil {
		return seq, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return seq, err
	}
	now := time.Now().UTC()
	seq = invoicedomain.InvoiceSequence{
		OrgID:      orgID,
		NextNumber: 1,
		UpdatedAt:  now,
	}
	if err := tx.WithContext(ctx).Create(&seq).Error; err != nil {
		return seq, err
	}
	return seq, nil
}

func ensureInvoiceTemplateTx(ctx context.Context, tx *gorm.DB, node *snowflake.Node, orgID snowflake.ID) (invoicetemplatedomain.InvoiceTemplate, error) {
	var seq invoicetemplatedomain.InvoiceTemplate
	err := tx.WithContext(ctx).Where("org_id = ?", orgID).First(&seq).Error
	if err == nil {
		return seq, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return seq, err
	}

	header := map[string]any{
		"title":           "Invoice",
		"logo_url":        "",
		"company_name":    "",
		"company_email":   "",
		"company_address": "",
		"bill_to_label":   "Bill to",
		"ship_to_label":   "Ship to",
	}

	footer := map[string]any{
		"note":  "Thank you for your business.",
		"legal": "This invoice is generated electronically and is valid without a signature.",
	}

	style := map[string]any{
		"table": map[string]any{
			"header_bg":       "#f1f5f9",
			"row_border":      "#e5e7eb",
			"font_size":       "12px",
			"font_family":     "Inter, system-ui, sans-serif",
			"accent_color":    "#22c55e",
			"primary_color":   "#0f172a",
			"secondary_color": "#64748b",
		},
	}

	now := time.Now().UTC()
	seq = invoicetemplatedomain.InvoiceTemplate{
		ID:        node.Generate(),
		OrgID:     orgID,
		Name:      "Default Invoice",
		Locale:    "en",
		Currency:  "USD",
		Header:    header,
		Footer:    footer,
		Style:     style,
		IsDefault: true,
		IsLocked:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.WithContext(ctx).Create(&seq).Error; err != nil {
		return seq, err
	}

	return seq, nil
}

func ensureLedgerAccounts(ctx context.Context, db *gorm.DB, node *snowflake.Node, orgID snowflake.ID) error {
	type account struct {
		Code string
		Type string
		Name string
	}

	accounts := []account{
		{"accounts_receivable", "asset", "Accounts Receivable"},
		{"cash", "asset", "Cash / Bank"},

		{"revenue_usage", "income", "Usage Revenue"},
		{"revenue_flat", "income", "Subscription Revenue"},

		{"tax_payable", "liability", "Tax Payable"},
		{"credit_balance", "liability", "Customer Credit Balance"},
		{"refund_liability", "liability", "Refund Liability"},

		{"payment_fee_expense", "expense", "Payment Gateway Fees"},
		{"adjustment", "expense", "Billing Adjustment"},
	}

	for _, a := range accounts {
		err := db.WithContext(ctx).
			Exec(`
				INSERT INTO ledger_accounts (id, org_id, code, type, name)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT (org_id, code) DO NOTHING
			`,
				node.Generate(),
				orgID,
				a.Code,
				a.Type,
				a.Name,
			).Error

		if err != nil {
			return err
		}
	}

	return nil
}
