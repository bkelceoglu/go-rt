package ui

import (
	"testing"
)

func TestColorizeAndStripANSI(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	text := "gort engine"
	colored := BoldCyan(text)
	if colored == text {
		t.Errorf("expected ANSI escape codes in colored text, got %q", colored)
	}

	stripped := StripANSI(colored)
	if stripped != text {
		t.Errorf("expected %q, got %q", text, stripped)
	}

	if VisibleLen(colored) != len(text) {
		t.Errorf("expected visible length %d, got %d", len(text), VisibleLen(colored))
	}
}

func TestPadRightVisible(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	colored := Red("alert")
	padded := PadRightVisible(colored, 10)
	if VisibleLen(padded) != 10 {
		t.Errorf("expected visible length 10, got %d", VisibleLen(padded))
	}
}

func TestBadgeFormatting(t *testing.T) {
	prev := IsColorEnabled()
	SetColorEnabled(false)
	defer SetColorEnabled(prev)

	cases := []struct {
		name     string
		label    string
		expected string
	}{
		{name: "critical severity", label: "CRITICAL", expected: "[CRITICAL]"},
		{name: "pass status", label: "PASS", expected: " PASS "},
		{name: "custom label", label: "INFO", expected: "[INFO]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Badge(tc.label); got != tc.expected {
				t.Errorf("Badge(%q) = %q, want %q", tc.label, got, tc.expected)
			}
		})
	}
}
