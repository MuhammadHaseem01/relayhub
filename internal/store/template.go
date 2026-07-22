package store

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrMissingVars = errors.New("template: missing required variables")

var placeholderRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

func SubstituteVars(body string, vars map[string]string) (string, []string, error) {
	matches := placeholderRe.FindAllStringSubmatch(body, -1)

	seen := make(map[string]struct{})
	var missing []string

	for _, m := range matches {
		key := m[1]
		if _, already := seen[key]; already {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := vars[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return "", missing, fmt.Errorf("%w: %s", ErrMissingVars, strings.Join(missing, ", "))
	}

	result := placeholderRe.ReplaceAllStringFunc(body, func(match string) string {
		key := match[2 : len(match)-2]
		return vars[key]
	})

	return result, nil, nil
}
