package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials    = errors.New("invalid_credentials")
	ErrSessionExpired        = errors.New("session_expired")
	ErrSessionRevoked        = errors.New("session_revoked")
	ErrSessionNotFound       = errors.New("session_not_found")
	ErrNoOrganization        = errors.New("no_organization")
	ErrPasswordChangeNeeded  = errors.New("password_change_required")
	ErrInvalidOrganization   = errors.New("invalid_organization")
	ErrNotOrganizationMember = errors.New("not_organization_member")
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
	log *zap.Logger
}

type LoginResponse struct {
	Token              string    `json:"token"`
	UserID             uuid.UUID `json:"userId"`
	Email              string    `json:"email"`
	OrgID              uuid.UUID `json:"orgId"`
	OrgIDs             []string  `json:"orgIds"`
	MustChangePassword bool      `json:"mustChangePassword"`
	SessionExpiresAt   time.Time `json:"sessionExpiresAt"`
}

type AuthSession struct {
	UserID             uuid.UUID
	OrgID              uuid.UUID
	MustChangePassword bool
}

func NewService(db *gorm.DB, cfg *config.Config, log *zap.Logger) *Service {
	if log == nil {
		log = zap.L()
	}
	return &Service{db: db, cfg: cfg, log: log}
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (LoginResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return LoginResponse{}, ErrInvalidCredentials
	}

	user, err := s.getUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResponse{}, ErrInvalidCredentials
		}
		return LoginResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	orgIDs, activeOrgID, err := s.listUserOrgIDs(ctx, user.ID)
	if err != nil {
		return LoginResponse{}, err
	}
	if activeOrgID == uuid.Nil {
		return LoginResponse{}, ErrNoOrganization
	}

	secret := ""
	if s.cfg != nil {
		secret = strings.TrimSpace(s.cfg.SessionConfig.SessionSecret)
	}
	token, tokenHash, err := generateSessionToken(secret)
	if err != nil {
		return LoginResponse{}, err
	}

	expiresAt := time.Now().UTC().Add(s.sessionTTL())
	orgIDsJSON, _ := json.Marshal(orgIDs)

	if err := s.db.WithContext(ctx).Exec(
		`INSERT INTO sessions (id, user_id, session_token_hash, user_agent, ip_address, active_org_id, org_ids, expires_at, created_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), user.ID, tokenHash, userAgent, ip, activeOrgID, string(orgIDsJSON), expiresAt,
	).Error; err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		Token:              token,
		UserID:             user.ID,
		Email:              user.Email,
		OrgID:              activeOrgID,
		OrgIDs:             orgIDs,
		MustChangePassword: user.MustChangePassword,
		SessionExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (AuthSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AuthSession{}, ErrSessionNotFound
	}

	secret := ""
	if s.cfg != nil {
		secret = strings.TrimSpace(s.cfg.SessionConfig.SessionSecret)
	}
	hash := hashToken(secret, token)

	var row struct {
		UserID             uuid.UUID
		OrgID              uuid.UUID
		ExpiresAt          time.Time
		RevokedAt          *time.Time
		MustChangePassword bool
	}

	err := s.db.WithContext(ctx).Raw(
		`SELECT s.user_id, s.active_org_id, s.expires_at, s.revoked_at, u.must_change_password
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.session_token_hash = ? LIMIT 1`,
		hash,
	).Scan(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthSession{}, ErrSessionNotFound
		}
		return AuthSession{}, err
	}

	if row.RevokedAt != nil {
		return AuthSession{}, ErrSessionRevoked
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return AuthSession{}, ErrSessionExpired
	}

	if row.OrgID == uuid.Nil {
		orgIDs, activeOrgID, err := s.listUserOrgIDs(ctx, row.UserID)
		if err != nil {
			return AuthSession{}, err
		}
		if activeOrgID == uuid.Nil {
			return AuthSession{}, ErrNoOrganization
		}
		row.OrgID = activeOrgID
		orgIDsJSON, _ := json.Marshal(orgIDs)
		_ = s.db.WithContext(ctx).Exec(
			`UPDATE sessions SET active_org_id = ?, org_ids = ? WHERE session_token_hash = ?`,
			activeOrgID, string(orgIDsJSON), hash,
		).Error
	}

	_ = s.db.WithContext(ctx).Exec(
		`UPDATE sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE session_token_hash = ?`,
		hash,
	).Error

	return AuthSession{
		UserID:             row.UserID,
		OrgID:              row.OrgID,
		MustChangePassword: row.MustChangePassword,
	}, nil
}

func (s *Service) RevokeSession(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrSessionNotFound
	}
	secret := ""
	if s.cfg != nil {
		secret = strings.TrimSpace(s.cfg.SessionConfig.SessionSecret)
	}
	hash := hashToken(secret, token)

	result := s.db.WithContext(ctx).Exec(
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE session_token_hash = ? AND revoked_at IS NULL`,
		hash,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *Service) SwitchOrganization(ctx context.Context, token string, orgID uuid.UUID) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrSessionNotFound
	}
	if orgID == uuid.Nil {
		return ErrInvalidOrganization
	}
	secret := ""
	if s.cfg != nil {
		secret = strings.TrimSpace(s.cfg.SessionConfig.SessionSecret)
	}
	hash := hashToken(secret, token)

	var row struct {
		UserID uuid.UUID
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT user_id FROM sessions WHERE session_token_hash = ? AND revoked_at IS NULL LIMIT 1`,
		hash,
	).Scan(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return err
	}
	if row.UserID == uuid.Nil {
		return ErrSessionNotFound
	}

	var count int64
	if err := s.db.WithContext(ctx).
		Table("organization_members").
		Where("user_id = ? AND org_id = ?", row.UserID, orgID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotOrganizationMember
	}

	if err := s.db.WithContext(ctx).Exec(
		`UPDATE sessions SET active_org_id = ?, last_seen_at = CURRENT_TIMESTAMP WHERE session_token_hash = ?`,
		orgID, hash,
	).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if userID == uuid.Nil || currentPassword == "" || newPassword == "" {
		return ErrInvalidCredentials
	}

	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Exec(
		`UPDATE users
		 SET password_hash = ?, must_change_password = false, last_password_changed = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		string(hash), userID,
	).Error
}

func (s *Service) GetMemberRole(ctx context.Context, userID, orgID uuid.UUID) (string, error) {
	if userID == uuid.Nil || orgID == uuid.Nil {
		return "", ErrNotOrganizationMember
	}
	var role string
	if err := s.db.WithContext(ctx).Raw(
		`SELECT role FROM organization_members WHERE user_id = ? AND org_id = ? LIMIT 1`,
		userID, orgID,
	).Scan(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotOrganizationMember
		}
		return "", err
	}
	if strings.TrimSpace(role) == "" {
		return "", ErrNotOrganizationMember
	}
	return role, nil
}

func (s *Service) SkipPasswordChange(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrInvalidCredentials
	}
	return s.db.WithContext(ctx).Exec(
		`UPDATE users
		 SET must_change_password = false, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		userID,
	).Error
}

type dbUser struct {
	ID                 uuid.UUID
	Email              string
	PasswordHash       string
	MustChangePassword bool
}

func (s *Service) getUserByEmail(ctx context.Context, email string) (dbUser, error) {
	var user dbUser
	err := s.db.WithContext(ctx).Raw(
		`SELECT id, email, password_hash, must_change_password FROM users WHERE email = ? LIMIT 1`,
		email,
	).Scan(&user).Error
	return user, err
}

func (s *Service) getUserByID(ctx context.Context, userID uuid.UUID) (dbUser, error) {
	var user dbUser
	err := s.db.WithContext(ctx).Raw(
		`SELECT id, email, password_hash, must_change_password FROM users WHERE id = ? LIMIT 1`,
		userID,
	).Scan(&user).Error
	return user, err
}

func (s *Service) listUserOrgIDs(ctx context.Context, userID uuid.UUID) ([]string, uuid.UUID, error) {
	type orgRow struct {
		ID        uuid.UUID
		IsDefault bool
	}
	var orgs []orgRow
	if err := s.db.WithContext(ctx).Raw(
		`SELECT o.id, o.is_default
		 FROM organization_members m
		 JOIN organizations o ON o.id = m.org_id
		 WHERE m.user_id = ?
		 ORDER BY o.is_default DESC, o.created_at ASC`,
		userID,
	).Scan(&orgs).Error; err != nil {
		return nil, uuid.Nil, err
	}

	if len(orgs) == 0 {
		return nil, uuid.Nil, nil
	}

	orgIDs := make([]string, 0, len(orgs))
	for _, org := range orgs {
		orgIDs = append(orgIDs, org.ID.String())
	}
	return orgIDs, orgs[0].ID, nil
}

func (s *Service) sessionTTL() time.Duration {
	if s.cfg != nil && s.cfg.SessionConfig.SessionTTLHours > 0 {
		return time.Duration(s.cfg.SessionConfig.SessionTTLHours) * time.Hour
	}
	return 24 * time.Hour
}

func generateSessionToken(secret string) (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashToken(secret, token), nil
}

func hashToken(secret, token string) string {
	if secret != "" {
		token = secret + ":" + token
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
