package service

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/billingoperations/domain"
)

func timePtr(t sql.NullTime) *time.Time {
	if t.Valid {
		val := t.Time.UTC()
		return &val
	}
	return nil
}

func parseSnowflakeID(id string) (snowflake.ID, error) {
	return snowflake.ParseString(id)
}

func normalizeIdempotencyKey(key string) string {
	return strings.TrimSpace(key)
}

func buildAuditAction(actionType string) string {
	return "billing_operations.action." + strings.ToLower(actionType)
}

func computeAgingBucket(days int) string {
	switch {
	case days <= 30:
		return "0-30"
	case days <= 60:
		return "31-60"
	case days <= 90:
		return "61-90"
	default:
		return "90+"
	}
}

func computeRiskLevel(amount int64, days int) string {
	score := int(amount/10000) + days
	if score > 100 {
		return "high"
	}
	if score > 50 {
		return "medium"
	}
	return "low"
}

func decryptToken(key []byte, ciphertextB64 string) string {
	ciphertextB64 = strings.TrimSpace(ciphertextB64)
	if len(key) == 0 || ciphertextB64 == "" {
		return ""
	}

	ciphertext, err := base64.RawStdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return ""
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return ""
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}

	return string(plaintext)
}

func assignmentFields(
	assignedTo sql.NullString,
	assignedAt time.Time,
	expiresAt sql.NullTime,
	status string,
	releasedAt sql.NullTime,
	releasedBy sql.NullString,
	releaseReason sql.NullString,
	breachedAt sql.NullTime,
	breachLevel sql.NullString,
	lastActionAt sql.NullTime,
	now time.Time,
) domain.Assignment {
	assigned := strings.TrimSpace(assignedTo.String)
	if !assignedTo.Valid {
		assigned = ""
	}
	expiresVal := time.Time{}
	if expiresAt.Valid {
		expiresVal = expiresAt.Time.UTC()
	}

	releaseVal := (*time.Time)(nil)
	if releasedAt.Valid {
		t := releasedAt.Time.UTC()
		releaseVal = &t
	}

	breachVal := (*time.Time)(nil)
	if breachedAt.Valid {
		t := breachedAt.Time.UTC()
		breachVal = &t
	}

	finalStatus := status
	if finalStatus == "" {
		if assigned != "" {
			finalStatus = domain.AssignmentStatusAssigned
		}
	}

	slaStatus := ""
	timeSinceAssigned := ""

	if assigned != "" && finalStatus != domain.AssignmentStatusReleased {
		referenceTime := assignedAt
		if lastActionAt.Valid {
			referenceTime = lastActionAt.Time
		}

		minutesSince := int(now.Sub(referenceTime).Minutes())
		if minutesSince < 0 {
			minutesSince = 0
		}

		switch {
		case minutesSince < 30:
			slaStatus = domain.SLAFresh
		case minutesSince < 90:
			slaStatus = domain.SLAActive
		case minutesSince < 240:
			slaStatus = domain.SLAAging
		default:
			slaStatus = domain.SLAStale
		}

		duration := now.Sub(assignedAt)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours > 0 {
			timeSinceAssigned = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			timeSinceAssigned = fmt.Sprintf("%dm", minutes)
		}
	}

	return domain.Assignment{
		Status:              finalStatus,
		AssignedTo:          assigned,
		AssignedAt:          assignedAt,
		AssignmentExpiresAt: expiresVal,
		ReleasedAt:          releaseVal,
		ReleasedBy:          strings.TrimSpace(releasedBy.String),
		ReleaseReason:       strings.TrimSpace(releaseReason.String),
		BreachedAt:          breachVal,
		BreachLevel:         strings.TrimSpace(breachLevel.String),
		LastActionAt:        timePtr(lastActionAt),
		SLAStatus:           slaStatus,
		TimeSinceAssigned:   timeSinceAssigned,
	}
}

func toJson(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
