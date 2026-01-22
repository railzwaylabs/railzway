package oauth2provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
	"github.com/smallbiznis/railzway/pkg/db"
)

type staticClock struct {
	now time.Time
}

func (c staticClock) Now() time.Time {
	return c.now
}

type staticTokenGen struct {
	values []string
	idx    int
}

func (g *staticTokenGen) NewToken() (string, error) {
	if g.idx >= len(g.values) {
		return "token", nil
	}
	val := g.values[g.idx]
	g.idx++
	return val, nil
}

type fakeUserRepo struct{}

func (fakeUserRepo) Count(context.Context) (int64, error)           { return 0, nil }
func (fakeUserRepo) Create(context.Context, *authdomain.User) error { return nil }
func (fakeUserRepo) FindOne(context.Context, authdomain.User) (*authdomain.User, error) {
	return nil, authdomain.ErrUserNotFound
}
func (fakeUserRepo) FindByExternalID(context.Context, string) (*authdomain.User, error) {
	return nil, authdomain.ErrUserNotFound
}
func (fakeUserRepo) FindByID(context.Context, snowflake.ID) (*authdomain.User, error) {
	return nil, authdomain.ErrUserNotFound
}
func (fakeUserRepo) UpdateFields(context.Context, snowflake.ID, map[string]any) error {
	return authdomain.ErrUserNotFound
}

func newTestService(t *testing.T) (*Service, Store) {
	t.Helper()

	dbConn, err := db.NewTest()
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := dbConn.AutoMigrate(&AuthorizationCode{}, &AccessToken{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	store := NewStore(dbConn)
	cfg := Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		CodeTTL:      5 * time.Minute,
		AccessTTL:    time.Hour,
	}

	svc := &Service{
		cfg:      cfg,
		store:    store,
		users:    fakeUserRepo{},
		clock:    staticClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		tokenGen: &staticTokenGen{values: []string{"access-token-1"}},
	}

	return svc, store
}

func TestExchangeTokenSuccess(t *testing.T) {
	svc, store := newTestService(t)

	code := "auth-code-1"
	codeHash := hashToken(code)
	authCode := &AuthorizationCode{
		CodeHash:    codeHash,
		ClientID:    svc.cfg.ClientID,
		RedirectURI: "https://acme.usevalora.net/login/usevalora_cloud",
		UserID:      snowflake.ID(1),
		Scopes:      []string{"user:email"},
		ExpiresAt:   svc.clock.Now().Add(svc.cfg.CodeTTL),
	}

	if err := store.CreateAuthorizationCode(context.Background(), authCode); err != nil {
		t.Fatalf("failed to create auth code: %v", err)
	}

	resp, err := svc.ExchangeToken(context.Background(), TokenRequest{
		ClientID:     svc.cfg.ClientID,
		ClientSecret: svc.cfg.ClientSecret,
		Code:         code,
		RedirectURI:  authCode.RedirectURI,
	})
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}

	stored, err := store.GetAccessToken(context.Background(), hashToken(resp.AccessToken))
	if err != nil {
		t.Fatalf("expected stored access token: %v", err)
	}
	if stored.UserID != authCode.UserID {
		t.Fatalf("expected user id %v, got %v", authCode.UserID, stored.UserID)
	}
}

func TestExchangeTokenCodeReuseFails(t *testing.T) {
	svc, store := newTestService(t)

	code := "auth-code-reuse"
	codeHash := hashToken(code)
	authCode := &AuthorizationCode{
		CodeHash:    codeHash,
		ClientID:    svc.cfg.ClientID,
		RedirectURI: "https://acme.usevalora.net/login/usevalora_cloud",
		UserID:      snowflake.ID(2),
		Scopes:      []string{"user:email"},
		ExpiresAt:   svc.clock.Now().Add(svc.cfg.CodeTTL),
	}

	if err := store.CreateAuthorizationCode(context.Background(), authCode); err != nil {
		t.Fatalf("failed to create auth code: %v", err)
	}

	_, err := svc.ExchangeToken(context.Background(), TokenRequest{
		ClientID:     svc.cfg.ClientID,
		ClientSecret: svc.cfg.ClientSecret,
		Code:         code,
		RedirectURI:  authCode.RedirectURI,
	})
	if err != nil {
		t.Fatalf("expected first exchange success, got %v", err)
	}

	_, err = svc.ExchangeToken(context.Background(), TokenRequest{
		ClientID:     svc.cfg.ClientID,
		ClientSecret: svc.cfg.ClientSecret,
		Code:         code,
		RedirectURI:  authCode.RedirectURI,
	})
	if err != ErrCodeUsed {
		t.Fatalf("expected ErrCodeUsed, got %v", err)
	}
}

func TestExchangeTokenRedirectMismatchFails(t *testing.T) {
	svc, store := newTestService(t)

	code := "auth-code-redirect"
	authCode := &AuthorizationCode{
		CodeHash:    hashToken(code),
		ClientID:    svc.cfg.ClientID,
		RedirectURI: "https://acme.usevalora.net/login/usevalora_cloud",
		UserID:      snowflake.ID(3),
		Scopes:      []string{"user:email"},
		ExpiresAt:   svc.clock.Now().Add(svc.cfg.CodeTTL),
	}

	if err := store.CreateAuthorizationCode(context.Background(), authCode); err != nil {
		t.Fatalf("failed to create auth code: %v", err)
	}

	_, err := svc.ExchangeToken(context.Background(), TokenRequest{
		ClientID:     svc.cfg.ClientID,
		ClientSecret: svc.cfg.ClientSecret,
		Code:         code,
		RedirectURI:  "https://acme.usevalora.net/internal/auth/oauth/callback/usevalora_cloud",
	})
	if err != ErrInvalidRedirectURI {
		t.Fatalf("expected ErrInvalidRedirectURI, got %v", err)
	}
}

func TestExchangeTokenPKCEVerification(t *testing.T) {
	svc, store := newTestService(t)

	verifier := "verifier-value"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	challengeHash := hashString(challenge)

	code := "auth-code-pkce"
	authCode := &AuthorizationCode{
		CodeHash:          hashToken(code),
		ClientID:          svc.cfg.ClientID,
		RedirectURI:       "https://acme.usevalora.net/login/usevalora_cloud",
		UserID:            snowflake.ID(4),
		Scopes:            []string{"user:email"},
		CodeChallengeHash: &challengeHash,
		ExpiresAt:         svc.clock.Now().Add(svc.cfg.CodeTTL),
	}

	if err := store.CreateAuthorizationCode(context.Background(), authCode); err != nil {
		t.Fatalf("failed to create auth code: %v", err)
	}

	_, err := svc.ExchangeToken(context.Background(), TokenRequest{
		ClientID:     svc.cfg.ClientID,
		ClientSecret: svc.cfg.ClientSecret,
		Code:         code,
		RedirectURI:  authCode.RedirectURI,
		CodeVerifier: "wrong-verifier",
	})
	if err != ErrPKCEMismatch {
		t.Fatalf("expected ErrPKCEMismatch, got %v", err)
	}
}
