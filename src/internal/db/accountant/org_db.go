package accountant

import (
	"context"
	"fmt"

	"accountant/src/internal/db"
)

// UpsertOrganization ensures the organization row exists in this DB.
// Uses org_id as a placeholder name when no real name is available.
// If the row already has a real name (i.e., not the placeholder), it is left unchanged.
func UpsertOrganization(ctx context.Context, orgID string) error {
	_, err := db.GetPool().Exec(ctx, `
		INSERT INTO public.organizations (organization_id, organization_name)
		VALUES ($1, $1)
		ON CONFLICT (organization_id) DO NOTHING
	`, orgID)
	if err != nil {
		return fmt.Errorf("upsert organization: %w", err)
	}
	return nil
}
