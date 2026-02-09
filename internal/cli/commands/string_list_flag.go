package commands

import "strings"

type stringListFlag []string

func (s *stringListFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	*s = append(*s, trimmed)
	return nil
}

func (s *stringListFlag) Values() []string {
	if s == nil || len(*s) == 0 {
		return nil
	}
	out := make([]string, len(*s))
	copy(out, *s)
	return out
}
