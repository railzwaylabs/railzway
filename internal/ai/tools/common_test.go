package tools

import "testing"

func TestOrganizationIDInputIdentifier(t *testing.T) {
	t.Run("prefers org id", func(t *testing.T) {
		input := organizationIDInput{
			ID:    " org_fallback ",
			OrgID: " org_primary ",
		}

		if got := input.identifier(); got != "org_primary" {
			t.Fatalf("identifier() = %q, want %q", got, "org_primary")
		}
	})

	t.Run("falls back to id", func(t *testing.T) {
		input := organizationIDInput{ID: " org_from_id "}

		if got := input.identifier(); got != "org_from_id" {
			t.Fatalf("identifier() = %q, want %q", got, "org_from_id")
		}
	})
}
