package license

import (
	"encoding/base64"
	"errors"

	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/railzwaylabs/railzway/internal/config"
)

// Capabilities represents the features enabled by the active license.
type Capabilities struct {
	SSO           bool
	RBAC          bool
	AuditExport   bool
	AppStore      bool
	DataWarehouse bool
	Analytics     bool
	// Add other capabilities as needed
}

// Service manages license verification and capability enforcement.
type Service struct {
	verifier     *Verifier
	logger       *zap.Logger
	capabilities Capabilities
	license      *Payload
	mu           sync.RWMutex
}

type LicenseMetadata struct {
	OrgID     string    `json:"org_id"`
	IssuedBy  string    `json:"issued_by"`
	ExpiresAt time.Time `json:"expires_at"`
	Plan      string    `json:"plan"` // Derived, e.g. "plus" if valid, "oss" otherwise
}

// NewService creates a new license service.
func NewService(logger *zap.Logger, cfg config.Config) (*Service, error) {
	svc := &Service{
		logger: logger,
		// Default capabilities enabled in OSS mode
		capabilities: Capabilities{
			AppStore: true,
		},
	}

	publicKeyBase64 := cfg.License.PublicKey
	if publicKeyBase64 != "" {
		pubBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
		if err != nil {
			return nil, fmt.Errorf("invalid public key encoding: %w", err)
		}
		if len(pubBytes) != 32 { // Ed25519 public key size
			return nil, errors.New("invalid public key size")
		}
		verifier := NewVerifier(pubBytes)
		svc.verifier = verifier
	}

	// Try loading from file defined in config
	if cfg.License.FilePath != "" {
		if err := svc.LoadFromFile(cfg.License.FilePath); err != nil {
			logger.Warn("Failed to load license from file", zap.String("path", cfg.License.FilePath), zap.Error(err))
		}
	} else if key := os.Getenv("RAILZWAY_LICENSE_KEY"); key != "" {
		// Fallback to env var (legacy/dev convenience)
		// Note: The prompt prefers "Load license.json from configurable path"
		// but supporting env var doesn't hurt for container simplicity unless strictly forbidden.
		// The prompt says "Load license.json from configurable path", implies file.
		if err := svc.LoadLicense(key); err != nil {
			logger.Warn("Failed to load license from env", zap.Error(err))
		}
	}

	return svc, nil
}

// LoadFromFile loads the license from a file path.
func (s *Service) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return s.LoadLicense(string(data))
}

// LoadFromEnv checks RAILZWAY_LICENSE_KEY and loads the license.
// Deprecated: prefer file loading via config.
func (s *Service) LoadFromEnv() error {
	// Logic moved to NewService to support unified config loading
	return nil
}

// LoadLicense verifies and loads a license string (JSON).
func (s *Service) LoadLicense(licenseJSON string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.verifier == nil {
		return errors.New("license verifier not initialized (missing public key)")
	}

	payload, err := s.verifier.ParseAndVerify([]byte(licenseJSON))
	if err != nil {
		s.logger.Warn("Failed to verify license", zap.Error(err))
		// Fail open principle: DO NOT crash, just don't enable features.
		// Use default capabilities (OSS).
		// However, returning error helps startup diagnostics.
		return err
	}

	s.license = payload
	s.capabilities = s.deriveCapabilities(payload)

	s.logger.Info("License loaded successfully",
		zap.String("org_id", payload.OrgID),
		zap.Time("expires_at", payload.ExpiresAt),
		zap.Strings("features", payload.Features),
	)

	return nil
}

// Metadata returns safe metadata about the active license.
func (s *Service) Metadata() LicenseMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.license == nil {
		return LicenseMetadata{
			Plan: "oss",
		}
	}

	return LicenseMetadata{
		OrgID:     s.license.OrgID,
		IssuedBy:  s.license.IssuedBy,
		ExpiresAt: s.license.ExpiresAt,
		Plan:      "plus", // If we have a valid license payload, it's Plus
	}
}

// Capabilities returns the current active capabilities.
func (s *Service) Capabilities() Capabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities
}

// deriveCapabilities maps license features to the Capabilities struct.
func (s *Service) deriveCapabilities(p *Payload) Capabilities {
	caps := Capabilities{}
	for _, f := range p.Features {
		switch f {
		case "sso":
			caps.SSO = true
		case "rbac":
			caps.RBAC = true
		case "audit_export":
			caps.AuditExport = true
		case "app_store":
			caps.AppStore = true
		case "data_warehouse":
			caps.DataWarehouse = true
		case "analytics":
			caps.Analytics = true
		}
	}
	return caps
}

// HasCapability checks if a specific feature is enabled in the current license.
func (s *Service) HasCapability(feature string) bool {
	caps := s.Capabilities()
	switch feature {
	case "sso":
		return caps.SSO
	case "rbac":
		return caps.RBAC
	case "audit_export":
		return caps.AuditExport
	case "app_store":
		return caps.AppStore
	case "data_warehouse":
		return caps.DataWarehouse
	case "analytics":
		return caps.Analytics
	default:
		return false
	}
}
