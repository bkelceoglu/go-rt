package template

import (
	"regexp"
	"slices"
)

var placeholderRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

// ExtractVariables returns unique variable names referenced as {{var}} in order of appearance.
func ExtractVariables(tmpl string) []string {
	matches := placeholderRegex.FindAllStringSubmatch(tmpl, -1)
	var vars []string
	for _, match := range matches {
		if len(match) > 1 && !slices.Contains(vars, match[1]) {
			vars = append(vars, match[1])
		}
	}
	return vars
}

// ResolveInjectVar selects explicitVar if set, or the last variable in templates, defaulting to "prompt".
func ResolveInjectVar(explicitVar string, templates ...string) string {
	if explicitVar != "" {
		return explicitVar
	}
	var allVars []string
	for _, tmpl := range templates {
		allVars = appendUnique(allVars, ExtractVariables(tmpl))
	}
	if len(allVars) > 0 {
		return allVars[len(allVars)-1]
	}
	return "prompt"
}

func appendUnique(existing, incoming []string) []string {
	for _, item := range incoming {
		if !slices.Contains(existing, item) {
			existing = append(existing, item)
		}
	}
	return existing
}
