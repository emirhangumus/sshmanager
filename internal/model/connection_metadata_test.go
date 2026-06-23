package model

import "testing"

func TestNormalizeTags(t *testing.T) {
	got := NormalizeTags([]string{" prod ", "Prod", "", "db"})
	if len(got) != 2 {
		t.Fatalf("NormalizeTags length = %d, want 2", len(got))
	}
	if got[0] != "prod" || got[1] != "db" {
		t.Fatalf("NormalizeTags unexpected result: %v", got)
	}
}

func TestValidateGroup(t *testing.T) {
	valid := []string{"", "prod", "team/core", "prod-api_1"}
	for _, group := range valid {
		if err := ValidateGroup(group); err != nil {
			t.Fatalf("ValidateGroup(%q) unexpected error: %v", group, err)
		}
	}

	invalid := []string{"bad group", "team$core"}
	for _, group := range invalid {
		if err := ValidateGroup(group); err == nil {
			t.Fatalf("ValidateGroup(%q) expected error, got nil", group)
		}
	}
}

func TestValidateTags(t *testing.T) {
	if err := ValidateTags([]string{"prod", "team/core", "api-1"}); err != nil {
		t.Fatalf("ValidateTags unexpected error: %v", err)
	}
	if err := ValidateTags([]string{"bad tag"}); err == nil {
		t.Fatal("ValidateTags expected error for invalid tag, got nil")
	}
}
