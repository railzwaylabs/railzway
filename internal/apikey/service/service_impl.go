package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/apikey/domain"
	"github.com/railzwaylabs/railzway/internal/authz"
)

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListKeys(ctx context.Context, orgID string) (domain.ListAPIKeysResponse, error) {
	if strings.TrimSpace(orgID) == "" {
		return domain.ListAPIKeysResponse{}, domain.ErrInvalidOrganization
	}
	keys, err := s.repo.ListKeys(ctx, orgID)
	if err != nil {
		return domain.ListAPIKeysResponse{}, err
	}
	resp := make([]domain.APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, mapKeyResponse(key, ""))
	}
	return domain.ListAPIKeysResponse{Keys: resp}, nil
}

func (s *Service) CreateKey(ctx context.Context, orgID string, req domain.CreateAPIKeyRequest) (domain.APIKeyResponse, error) {
	if strings.TrimSpace(orgID) == "" {
		return domain.APIKeyResponse{}, domain.ErrInvalidOrganization
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.APIKeyResponse{}, domain.ErrInvalidName
	}
	keyType := strings.TrimSpace(req.KeyType)
	prefix := domain.KeyPrefixForType(domain.KeyType(keyType))
	if prefix == "" {
		return domain.APIKeyResponse{}, domain.ErrInvalidKeyType
	}

	secret, err := generateSecret()
	if err != nil {
		return domain.APIKeyResponse{}, err
	}
	fullKey := prefix + secret
	keyPrefix := prefix + secret[:6]

	now := time.Now().UTC()
	key := domain.APIKey{
		ID:            uuid.New().String(),
		OrgID:         orgID,
		Name:          name,
		KeyPrefix:     keyPrefix,
		KeyType:       domain.KeyType(keyType),
		Scopes:        cleanList(req.Scopes),
		AllowedIPs:    cleanList(req.AllowedIPs),
		AllowedDomain: cleanList(req.AllowedDomains),
		Status:        domain.KeyStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	hash := hashKey(fullKey)

	if err := s.repo.CreateKey(ctx, key, hash); err != nil {
		return domain.APIKeyResponse{}, err
	}
	return mapKeyResponse(key, fullKey), nil
}

func (s *Service) RevokeKey(ctx context.Context, orgID string, id string) (domain.APIKeyResponse, error) {
	if strings.TrimSpace(orgID) == "" {
		return domain.APIKeyResponse{}, domain.ErrInvalidOrganization
	}
	if strings.TrimSpace(id) == "" {
		return domain.APIKeyResponse{}, domain.ErrInvalidID
	}
	updated, err := s.repo.RevokeKey(ctx, orgID, id)
	if err != nil {
		return domain.APIKeyResponse{}, err
	}
	if updated == nil {
		return domain.APIKeyResponse{}, domain.ErrNotFound
	}
	return mapKeyResponse(*updated, ""), nil
}

func (s *Service) AuthorizeKey(ctx context.Context, rawKey string, req domain.AuthorizeAPIKeyRequest) (domain.APIKey, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return domain.APIKey{}, domain.ErrInvalidKey
	}
	if s.repo == nil {
		return domain.APIKey{}, domain.ErrInvalidKey
	}
	key, err := s.repo.FindByHash(ctx, hashKey(rawKey))
	if err != nil {
		return domain.APIKey{}, err
	}
	if key == nil || key.ID == "" {
		return domain.APIKey{}, domain.ErrInvalidKey
	}
	switch key.Status {
	case domain.KeyStatusRevoked, domain.KeyStatusDisabled:
		return domain.APIKey{}, domain.ErrKeyRevoked
	}
	if !allowedIP(req.IP, key.AllowedIPs) || !allowedDomain(req.Domain, key.AllowedDomain) {
		return domain.APIKey{}, domain.ErrKeyNotAllowed
	}
	if len(key.Scopes) == 0 {
		return domain.APIKey{}, domain.ErrScopeForbidden
	}
	authorizer, err := authz.NewAPIKeyAuthorizer(key.Scopes)
	if err != nil {
		return domain.APIKey{}, err
	}
	allowed, err := authorizer.Enforce(req.Resource, req.Action)
	if err != nil {
		return domain.APIKey{}, err
	}
	if !allowed {
		return domain.APIKey{}, domain.ErrScopeForbidden
	}
	_ = s.repo.TouchLastUsed(ctx, key.ID)
	return *key, nil
}

func generateSecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func allowedIP(candidate string, allowed []string) bool {
	candidate = strings.TrimSpace(candidate)
	if len(allowed) == 0 {
		return true
	}
	if candidate == "" {
		return false
	}
	ip := net.ParseIP(candidate)
	for _, item := range allowed {
		entry := strings.TrimSpace(item)
		if entry == "" {
			continue
		}
		if strings.EqualFold(entry, candidate) {
			return true
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil && ip != nil && cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func allowedDomain(candidate string, allowed []string) bool {
	candidate = normalizeDomain(candidate)
	if len(allowed) == 0 {
		return true
	}
	if candidate == "" {
		return false
	}
	for _, item := range allowed {
		entry := normalizeDomain(item)
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, "*.") {
			base := strings.TrimPrefix(entry, "*.")
			if base != "" && strings.HasSuffix(candidate, "."+base) {
				return true
			}
			continue
		}
		if strings.EqualFold(entry, candidate) {
			return true
		}
	}
	return false
}

func normalizeDomain(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		trimmed = strings.TrimPrefix(trimmed, "http://")
		trimmed = strings.TrimPrefix(trimmed, "https://")
	}
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		return host
	}
	return trimmed
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func mapKeyResponse(key domain.APIKey, fullKey string) domain.APIKeyResponse {
	return domain.APIKeyResponse{
		ID:             key.ID,
		OrgID:          key.OrgID,
		Name:           key.Name,
		KeyPrefix:      key.KeyPrefix,
		KeyType:        string(key.KeyType),
		Scopes:         key.Scopes,
		AllowedIPs:     key.AllowedIPs,
		AllowedDomains: key.AllowedDomain,
		Status:         string(key.Status),
		LastUsedAt:     key.LastUsedAt,
		RevokedAt:      key.RevokedAt,
		CreatedAt:      key.CreatedAt,
		UpdatedAt:      key.UpdatedAt,
		Key:            fullKey,
	}
}
