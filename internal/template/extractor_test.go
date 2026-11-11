package template_test

import (
	"testing"

	"github.com/caveman/go-rt/internal/template"
)

func TestExtractVariablesAndResolveInjectVar(t *testing.T) {
	t.Parallel()
	tmpl := "System: {{role}} for {{company}}. User query: {{ user_input }}"
	vars := template.ExtractVariables(tmpl)
	if len(vars) != 3 || vars[0] != "role" || vars[2] != "user_input" {
		t.Fatalf("unexpected extracted vars: %v", vars)
	}
	injectVar := template.ResolveInjectVar("", tmpl)
	if injectVar != "user_input" {
		t.Fatalf("expected last var 'user_input', got %q", injectVar)
	}
	if got := template.ResolveInjectVar("custom", tmpl); got != "custom" {
		t.Fatalf("expected explicit 'custom', got %q", got)
	}
}
