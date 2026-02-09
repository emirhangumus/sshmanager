package model

import "testing"

func TestValidateProxyJump(t *testing.T) {
	valid := []string{
		"",
		"bastion",
		"user@bastion",
		"bastion:2222",
		"user@bastion:2222,backup:2200",
	}
	for _, value := range valid {
		if err := ValidateProxyJump(value); err != nil {
			t.Fatalf("ValidateProxyJump(%q) unexpected error: %v", value, err)
		}
	}

	invalid := []string{
		"bad hop",
		",bastion",
		"bastion,",
		"bastion:99999",
		"user@@bastion",
	}
	for _, value := range invalid {
		if err := ValidateProxyJump(value); err == nil {
			t.Fatalf("ValidateProxyJump(%q) expected error, got nil", value)
		}
	}
}

func TestValidateForwardSpec(t *testing.T) {
	valid := []string{
		"8080:127.0.0.1:80",
		"127.0.0.1:8080:127.0.0.1:80",
		"9000:[::1]:9001",
	}
	for _, value := range valid {
		if err := ValidateForwardSpec(value); err != nil {
			t.Fatalf("ValidateForwardSpec(%q) unexpected error: %v", value, err)
		}
	}

	invalid := []string{
		"",
		"8080:localhost",
		"bad:localhost:80",
		"8080:localhost:70000",
		"80 80:localhost:80",
	}
	for _, value := range invalid {
		if err := ValidateForwardSpec(value); err == nil {
			t.Fatalf("ValidateForwardSpec(%q) expected error, got nil", value)
		}
	}
}

func TestValidateExtraSSHArgs(t *testing.T) {
	valid := [][]string{
		nil,
		{"-v"},
		{"-vvv", "-C"},
		{"-o", "ServerAliveInterval=30"},
		{"-oStrictHostKeyChecking=no"},
	}
	for _, args := range valid {
		if err := ValidateExtraSSHArgs(args); err != nil {
			t.Fatalf("ValidateExtraSSHArgs(%v) unexpected error: %v", args, err)
		}
	}

	invalid := [][]string{
		{"-L", "8080:127.0.0.1:80"},
		{"-o"},
		{"-o", "NoEquals"},
		{"-oProxyCommand=nc %h %p"},
		{"hostname"},
	}
	for _, args := range invalid {
		if err := ValidateExtraSSHArgs(args); err == nil {
			t.Fatalf("ValidateExtraSSHArgs(%v) expected error, got nil", args)
		}
	}
}

func TestNormalizeStringList(t *testing.T) {
	got := NormalizeStringList([]string{" a ", "", "  ", "b"})
	if len(got) != 2 {
		t.Fatalf("NormalizeStringList length = %d, want 2", len(got))
	}
	if got[0] != "a" || got[1] != "b" {
		t.Fatalf("NormalizeStringList unexpected value: %v", got)
	}

	if NormalizeStringList([]string{"", " "}) != nil {
		t.Fatal("NormalizeStringList should return nil for empty input")
	}
}
