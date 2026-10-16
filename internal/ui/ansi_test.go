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
	SetColorEnabled(false)
	badge := Badge("CRITICAL")
	if badge != "[CRITICAL]" {
		t.Errorf("expected [CRITICAL], got %q", badge)
	}
}
