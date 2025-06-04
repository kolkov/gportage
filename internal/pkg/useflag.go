package pkg

import (
	"regexp"
	"strings"
)

type UseFlag struct {
	Name      string
	Default   bool
	Enabled   bool
	Condition string // Условие USE-флага
}

func ParseUseFlag(flag string) UseFlag {
	// Пример: ssl? ( >=dev-libs/openssl-1.1.0 )
	re := regexp.MustCompile(`([a-zA-Z0-9_]+)\?\s*\((.*)\)`)
	matches := re.FindStringSubmatch(flag)

	if len(matches) == 3 {
		return UseFlag{
			Name:      matches[1],
			Condition: strings.TrimSpace(matches[2]),
		}
	}

	return UseFlag{
		Name: strings.TrimPrefix(flag, "-"),
	}
}

func (u *UseFlag) IsEnabled(flags map[string]bool) bool {
	if enabled, exists := flags[u.Name]; exists {
		return enabled
	}
	return u.Default
}
