package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

var (
	ansiEscapeRegex  = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	hasColorOverride atomic.Bool
	isColorEnabled   atomic.Bool
	defaultColorOnce = sync.OnceValue(detectColorSupport)
	badgeFormatters  = map[string]func(string) string{
		"CRITICAL": func(_ string) string { return BoldRed("[CRITICAL]") },
		"HIGH":     func(_ string) string { return Red("[HIGH]") },
		"MEDIUM":   func(_ string) string { return Yellow("[MEDIUM]") },
		"LOW":      func(_ string) string { return Cyan("[LOW]") },
		"PASS":     func(_ string) string { return BoldGreen(" PASS ") },
		"FAIL":     func(_ string) string { return BoldRed(" FAIL ") },
	}
)

func detectColorSupport() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// SetColorEnabled manually overrides automatic color detection.
func SetColorEnabled(enabled bool) {
	isColorEnabled.Store(enabled)
	hasColorOverride.Store(true)
}

// IsColorEnabled reports whether ANSI color output is currently active.
func IsColorEnabled() bool {
	if hasColorOverride.Load() {
		return isColorEnabled.Load()
	}
	return defaultColorOnce()
}

// Colorize wraps text with ANSI escape codes if color is enabled.
func Colorize(code, text string) string {
	if !IsColorEnabled() || text == "" {
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", code, text)
}

// Text formatting helpers.
func Bold(text string) string    { return Colorize("1", text) }
func Dim(text string) string     { return Colorize("2", text) }
func Red(text string) string     { return Colorize("31", text) }
func Green(text string) string   { return Colorize("32", text) }
func Yellow(text string) string  { return Colorize("33", text) }
func Blue(text string) string    { return Colorize("34", text) }
func Magenta(text string) string { return Colorize("35", text) }
func Cyan(text string) string    { return Colorize("36", text) }
func Gray(text string) string    { return Colorize("90", text) }

// Bold color helpers.
func BoldRed(text string) string    { return Colorize("1;31", text) }
func BoldGreen(text string) string  { return Colorize("1;32", text) }
func BoldYellow(text string) string { return Colorize("1;33", text) }
func BoldCyan(text string) string   { return Colorize("1;36", text) }

// Badge returns a stylized severity or status badge.
func Badge(label string) string {
	if fn, ok := badgeFormatters[strings.ToUpper(label)]; ok {
		return fn(label)
	}
	return Gray("[" + label + "]")
}

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiEscapeRegex.ReplaceAllString(s, "")
}

// VisibleLen returns the printable rune length of a string, ignoring ANSI codes.
func VisibleLen(s string) int {
	return utf8.RuneCountInString(StripANSI(s))
}

// PadRightVisible pads string with spaces up to width based on visible rune length.
func PadRightVisible(s string, width int) string {
	vlen := VisibleLen(s)
	if vlen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vlen)
}

// PrefixGort returns the standard formatted [gort] CLI prefix.
func PrefixGort() string {
	return BoldCyan("[gort]")
}
