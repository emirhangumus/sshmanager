package model

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	maxConnectionGroupLength = 64
	maxConnectionTagLength   = 64
)

var (
	connectionGroupPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	connectionTagPattern   = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

func NormalizeTags(tags []string) []string {
	normalized := NormalizeStringList(tags)
	if len(normalized) == 0 {
		return nil
	}

	result := make([]string, 0, len(normalized))
	seen := make(map[string]struct{}, len(normalized))
	for _, tag := range normalized {
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func ValidateGroup(group string) error {
	trimmed := strings.TrimSpace(group)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > maxConnectionGroupLength {
		return fmt.Errorf("group must be %d characters or fewer", maxConnectionGroupLength)
	}
	if !connectionGroupPattern.MatchString(trimmed) {
		return fmt.Errorf("group may only contain letters, numbers, '.', '_', '/', or '-'")
	}
	return nil
}

func ValidateTags(tags []string) error {
	for _, rawTag := range tags {
		tag := strings.TrimSpace(rawTag)
		if tag == "" {
			continue
		}
		if len(tag) > maxConnectionTagLength {
			return fmt.Errorf("tag %q exceeds %d characters", tag, maxConnectionTagLength)
		}
		if !connectionTagPattern.MatchString(tag) {
			return fmt.Errorf("tag %q may only contain letters, numbers, '.', '_', '/', or '-'", tag)
		}
	}
	return nil
}
