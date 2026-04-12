package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/organization/domain"
	"gorm.io/gorm"
)

type stubRepo struct {
	formats []domain.OrganizationInvoiceNumberFormat
}

func (s *stubRepo) WithTx(tx *gorm.DB) domain.Repository { return s }

func (s *stubRepo) CreateOrganization(ctx context.Context, org domain.Organization) error { return nil }
func (s *stubRepo) Update(ctx context.Context, org domain.Organization) error             { return nil }
func (s *stubRepo) AddMember(ctx context.Context, member domain.OrganizationMember) error { return nil }
func (s *stubRepo) ListOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]domain.OrganizationListItem, error) {
	return nil, nil
}
func (s *stubRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.OrganizationMemberInfo, error) {
	return nil, nil
}
func (s *stubRepo) IsMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	return false, nil
}
func (s *stubRepo) CreateInvites(ctx context.Context, invites []domain.OrganizationInvite) error {
	return nil
}
func (s *stubRepo) GetInvite(ctx context.Context, inviteID uuid.UUID) (*domain.OrganizationInvite, error) {
	return nil, nil
}
func (s *stubRepo) UpdateInvite(ctx context.Context, invite domain.OrganizationInvite) error {
	return nil
}
func (s *stubRepo) UpsertBillingPreferences(ctx context.Context, prefs domain.OrganizationBillingPreferences) error {
	return nil
}

func (s *stubRepo) CreateInvoiceNumberFormat(ctx context.Context, format domain.OrganizationInvoiceNumberFormat) error {
	s.formats = append(s.formats, format)
	return nil
}

func (s *stubRepo) ListInvoiceNumberFormats(ctx context.Context, orgID uuid.UUID) ([]domain.OrganizationInvoiceNumberFormat, error) {
	out := make([]domain.OrganizationInvoiceNumberFormat, 0, len(s.formats))
	for _, item := range s.formats {
		if item.OrgID == orgID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *stubRepo) GetInvoiceNumberFormatByID(ctx context.Context, orgID, formatID uuid.UUID) (*domain.OrganizationInvoiceNumberFormat, error) {
	for _, item := range s.formats {
		if item.OrgID == orgID && item.ID == formatID {
			copied := item
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *stubRepo) UpdateInvoiceNumberFormat(ctx context.Context, format domain.OrganizationInvoiceNumberFormat) error {
	for i := range s.formats {
		if s.formats[i].ID == format.ID && s.formats[i].OrgID == format.OrgID {
			s.formats[i] = format
			return nil
		}
	}
	return nil
}

func (s *stubRepo) CreateLink(ctx context.Context, link domain.OrganizationLink) error { return nil }
func (s *stubRepo) ListChildLinks(ctx context.Context, parentOrgID uuid.UUID) ([]domain.OrganizationLink, error) {
	return nil, nil
}
func (s *stubRepo) ListParentLinks(ctx context.Context, childOrgID uuid.UUID) ([]domain.OrganizationLink, error) {
	return nil, nil
}

func TestCreateInvoiceNumberFormat_InvalidFormat(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(Params{Repo: repo})

	_, err := svc.CreateInvoiceNumberFormat(context.Background(), uuid.New(), uuid.New().String(), domain.InvoiceNumberFormatRequest{
		Format:        "INV-{YYYY}",
		SequenceScope: "org_month",
		EffectiveFrom: time.Now().UTC(),
	})
	if err != domain.ErrInvalidFormat {
		t.Fatalf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestCreateInvoiceNumberFormat_Overlap(t *testing.T) {
	orgID := uuid.New()
	repo := &stubRepo{
		formats: []domain.OrganizationInvoiceNumberFormat{
			{
				ID:            uuid.New(),
				OrgID:         orgID,
				Format:        "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
				SequenceScope: "org_month",
				EffectiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EffectiveTo:   nil,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}
	svc := NewService(Params{Repo: repo})

	_, err := svc.CreateInvoiceNumberFormat(context.Background(), uuid.New(), orgID.String(), domain.InvoiceNumberFormatRequest{
		Format:        "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
		SequenceScope: "org_month",
		EffectiveFrom: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != domain.ErrOverlappingFormat {
		t.Fatalf("expected ErrOverlappingFormat, got %v", err)
	}
}

func TestCloseInvoiceNumberFormat_Overlap(t *testing.T) {
	orgID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()

	repo := &stubRepo{
		formats: []domain.OrganizationInvoiceNumberFormat{
			{
				ID:            firstID,
				OrgID:         orgID,
				Format:        "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
				SequenceScope: "org_month",
				EffectiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EffectiveTo:   timePtr(time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)),
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
			{
				ID:            secondID,
				OrgID:         orgID,
				Format:        "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
				SequenceScope: "org_month",
				EffectiveFrom: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				EffectiveTo:   nil,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}
	svc := NewService(Params{Repo: repo})

	_, err := svc.CloseInvoiceNumberFormat(context.Background(), uuid.New(), orgID.String(), firstID.String(), time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC))
	if err != domain.ErrOverlappingFormat {
		t.Fatalf("expected ErrOverlappingFormat, got %v", err)
	}
}

func timePtr(value time.Time) *time.Time {
	v := value
	return &v
}
