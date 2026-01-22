package signup

import (
	"context"
	"strings"

	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
	orgdomain "github.com/smallbiznis/railzway/internal/organization/domain"
	"github.com/smallbiznis/railzway/internal/signup/domain"
)

type service struct {
	authsvc     authdomain.Service
	orgsvc      orgdomain.Service
	provisioner domain.Provisioner
}

const (
	defaultCountryCode  = "ID"
	defaultTimezoneName = "Asia/Jakarta"
)

func NewService(authsvc authdomain.Service, orgsvc orgdomain.Service, provisioner domain.Provisioner) domain.Service {
	return &service{
		authsvc:     authsvc,
		orgsvc:      orgsvc,
		provisioner: provisioner,
	}
}

func (s *service) Signup(ctx context.Context, req domain.Request) (*domain.Result, error) {
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, domain.ErrInvalidRequest
	}

	orgName := strings.TrimSpace(req.OrgName)
	if orgName == "" {
		orgName = strings.TrimSpace(req.Username)
	}

	if orgName == "" {
		return nil, domain.ErrInvalidRequest
	}

	user, err := s.authsvc.CreateUser(ctx, authdomain.CreateUserRequest{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.Username,
	})
	if err != nil {
		return nil, err
	}

	org, err := s.orgsvc.Create(ctx, user.ID, orgdomain.CreateOrganizationRequest{
		Name:         orgName,
		CountryCode:  defaultCountryCode,
		TimezoneName: defaultTimezoneName,
	})
	if err != nil {
		return nil, err
	}

	// Delegate provisioning to a dedicated service.
	if err := s.provisioner.Provision(ctx, org.ID); err != nil {
		return nil, err
	}

	session, err := s.authsvc.Login(ctx, authdomain.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
	})
	if err != nil {
		return nil, err
	}

	return &domain.Result{
		Session:   session.Session,
		RawToken:  session.RawToken,
		ExpiresAt: session.ExpiresAt,
		OrgID:     org.ID,
		UserID:    user.ID.String(),
	}, nil
}
