package rls

import (
	"fmt"

	"gorm.io/gorm"
)

func WithTenant(tx *gorm.DB, tenantID int64) error {
	return tx.Exec(
		"SET LOCAL app.current_org_id = ?",
		fmt.Sprintf("%d", tenantID),
	).Error
}
