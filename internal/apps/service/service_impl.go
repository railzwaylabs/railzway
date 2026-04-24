package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type Service struct {
	repo domain.Repository
	cfg  *config.Config
}

func NewService(repo domain.Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) ListCatalog(ctx context.Context) (domain.ListAppsResponse, error) {
	apps, err := s.repo.ListCatalog(ctx)
	if err != nil {
		return domain.ListAppsResponse{}, err
	}
	return domain.ListAppsResponse{Apps: apps, Warnings: s.warnings()}, nil
}

func (s *Service) ListInstallations(ctx context.Context) (domain.ListInstallationsResponse, error) {
	orgID, err := orgIDFromContext(ctx)
	if err != nil {
		return domain.ListInstallationsResponse{}, err
	}
	if s.repo == nil {
		return domain.ListInstallationsResponse{Installations: []domain.AppInstallation{}, Warnings: s.warnings()}, nil
	}
	items, err := s.repo.ListInstallations(ctx, orgID)
	if err != nil {
		return domain.ListInstallationsResponse{}, err
	}
	return domain.ListInstallationsResponse{Installations: items, Warnings: s.warnings()}, nil
}

func (s *Service) InstallApp(ctx context.Context, req domain.InstallAppRequest) (domain.AppInstallation, error) {
	orgID, err := orgIDFromContext(ctx)
	if err != nil {
		return domain.AppInstallation{}, err
	}
	appID := strings.TrimSpace(req.AppID)
	if appID == "" {
		return domain.AppInstallation{}, domain.ErrInvalidApp
	}
	authMethod := strings.TrimSpace(req.AuthMethod)
	allowedMethods := []string{}
	var catalog *domain.AppDefinition
	if s.repo != nil {
		existing, err := s.repo.FindInstallationByAppID(ctx, orgID, appID)
		if err != nil {
			return domain.AppInstallation{}, err
		}
		if existing != nil {
			return *existing, nil
		}
		catalog, err = s.repo.GetCatalogByID(ctx, appID)
		if err != nil {
			return domain.AppInstallation{}, err
		}
		if catalog == nil {
			return domain.AppInstallation{}, domain.ErrInvalidApp
		}
		allowedMethods = append(allowedMethods, catalog.AuthMethods...)
	}

	if authMethod == "" {
		if len(allowedMethods) > 0 {
			authMethod = allowedMethods[0]
		} else {
			authMethod = "api_keys"
		}
	}
	if len(allowedMethods) > 0 && !containsString(allowedMethods, authMethod) {
		return domain.AppInstallation{}, domain.ErrInvalidAuthMethod
	}

	if err := validateAppCredentials(catalog, authMethod, req.Credentials, false); err != nil {
		return domain.AppInstallation{}, err
	}

	now := time.Now().UTC()
	encryptedCreds, err := s.encryptCredentials(req.Credentials)
	if err != nil {
		return domain.AppInstallation{}, err
	}
	inst := domain.AppInstallation{
		ID:          uuid.New(),
		OrgID:       orgID,
		AppID:       appID,
		Status:      domain.InstallStatusActive,
		AuthMethod:  authMethod,
		Config:      defaultJSON(req.Config),
		Credentials: encryptedCreds,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if s.repo != nil {
		if err := s.repo.CreateInstallation(ctx, inst); err != nil {
			return domain.AppInstallation{}, err
		}
	}
	return inst, nil
}

func (s *Service) UpdateInstallation(ctx context.Context, id string, req domain.UpdateInstallationRequest) (domain.AppInstallation, error) {
	orgID, err := orgIDFromContext(ctx)
	if err != nil {
		return domain.AppInstallation{}, err
	}
	installID, err := parseID(id)
	if err != nil {
		return domain.AppInstallation{}, domain.ErrInvalidID
	}
	if s.repo == nil {
		return domain.AppInstallation{}, domain.ErrNotFound
	}
	inst, err := s.repo.FindInstallationByID(ctx, orgID, installID)
	if err != nil {
		return domain.AppInstallation{}, err
	}
	if inst == nil {
		return domain.AppInstallation{}, domain.ErrNotFound
	}

	nextMethod := inst.AuthMethod
	if req.AuthMethod != nil {
		nextMethod = strings.TrimSpace(*req.AuthMethod)
	}
	var catalog *domain.AppDefinition
	if s.repo != nil {
		catalog, err = s.repo.GetCatalogByID(ctx, inst.AppID)
		if err != nil {
			return domain.AppInstallation{}, err
		}
	}
	if req.AuthMethod != nil || len(req.Credentials) > 0 {
		if err := validateAppCredentials(catalog, nextMethod, req.Credentials, req.AuthMethod != nil && len(inst.Credentials) > 0); err != nil {
			return domain.AppInstallation{}, err
		}
	}

	updates := map[string]interface{}{"updated_at": time.Now().UTC()}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status != string(domain.InstallStatusActive) && status != string(domain.InstallStatusDisabled) {
			return domain.AppInstallation{}, domain.ErrInvalidStatus
		}
		updates["status"] = status
	}

	if len(req.Config) > 0 {
		updates["config"] = req.Config
	}

	if len(req.Credentials) > 0 {
		encryptedCreds, err := s.encryptCredentials(req.Credentials)
		if err != nil {
			return domain.AppInstallation{}, err
		}
		updates["credentials"] = encryptedCreds
	}

	if req.AuthMethod != nil {
		method := strings.TrimSpace(*req.AuthMethod)
		if method == "" {
			return domain.AppInstallation{}, domain.ErrInvalidAuthMethod
		}

		if s.repo != nil {
			if catalog == nil {
				return domain.AppInstallation{}, domain.ErrInvalidApp
			}
			if len(catalog.AuthMethods) > 0 && !containsString(catalog.AuthMethods, method) {
				return domain.AppInstallation{}, domain.ErrInvalidAuthMethod
			}
		}

		updates["auth_method"] = method
	}

	if err := s.repo.UpdateInstallation(ctx, orgID, installID, updates); err != nil {
		return domain.AppInstallation{}, err
	}

	updated, err := s.repo.FindInstallationByID(ctx, orgID, installID)
	if err != nil {
		return domain.AppInstallation{}, err
	}

	if updated == nil {
		return domain.AppInstallation{}, domain.ErrNotFound
	}

	return *updated, nil
}

func validateStripeCredentials(raw json.RawMessage) error {
	if len(raw) == 0 {
		return domain.ErrMissingCredentials
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.ErrMissingCredentials
	}
	_, hasPublishable := data["publishable_key"]
	_, hasSecret := data["secret_key"]
	_, hasWebhook := data["webhook_secret"]
	if !hasPublishable || !hasSecret || !hasWebhook {
		return domain.ErrMissingCredentials
	}
	return nil
}

func validateAppCredentials(catalog *domain.AppDefinition, method string, raw json.RawMessage, allowExisting bool) error {
	if catalog == nil || len(catalog.CredentialsSchema) == 0 {
		return nil
	}
	required := catalog.CredentialsSchema[method]
	if len(required) == 0 {
		return nil
	}
	if len(raw) == 0 {
		if allowExisting {
			return nil
		}
		return domain.ErrMissingCredentials
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.ErrMissingCredentials
	}
	for _, key := range required {
		value, ok := data[key]
		if !ok {
			return domain.ErrCredentialsKey
		}
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			return domain.ErrCredentialsKey
		}
	}
	return nil
}

func orgIDFromContext(ctx context.Context) (uuid.UUID, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidOrganization
	}
	return orgID, nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, err
	}
	return parsed, nil
}

func defaultJSON(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func containsString(values []string, target string) bool {
	for _, val := range values {
		if strings.EqualFold(strings.TrimSpace(val), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

type encryptedEnvelope struct {
	Alg        string `json:"alg"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	Version    int    `json:"v"`
}

func (s *Service) encryptCredentials(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	key, err := decodeKey(s.cfg)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, raw, nil)
	env := encryptedEnvelope{
		Alg:        "aes-256-gcm",
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Version:    1,
	}
	out, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func decodeKey(cfg *config.Config) ([]byte, error) {
	if cfg == nil {
		return nil, domain.ErrCredentialsKey
	}
	key := strings.TrimSpace(cfg.Integrations.AppsCredentialsKey)
	if key == "" {
		return nil, domain.ErrCredentialsKey
	}
	if len(key) == 32 {
		return []byte(key), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		return nil, domain.ErrCredentialsKey
	}
	return decoded, nil
}

func (s *Service) warnings() []domain.Warning {
	if s.cfg == nil {
		return []domain.Warning{{
			Module:  "apps",
			Code:    "apps_credentials_key_missing",
			Message: "APPS_CREDENTIALS_KEY is not set; app credentials cannot be encrypted.",
		}}
	}
	key := strings.TrimSpace(s.cfg.Integrations.AppsCredentialsKey)
	if key == "" {
		return []domain.Warning{{
			Module:  "apps",
			Code:    "apps_credentials_key_missing",
			Message: "APPS_CREDENTIALS_KEY is not set; app credentials cannot be encrypted.",
		}}
	}
	if len(key) == 32 {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		return []domain.Warning{{
			Module:  "apps",
			Code:    "apps_credentials_key_invalid",
			Message: "APPS_CREDENTIALS_KEY must be 32 bytes or base64-encoded 32 bytes.",
		}}
	}
	return nil
}
