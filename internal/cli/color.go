package cli

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
)

// ANSI escape codes for terminal colors.
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiCyan      = "\033[36m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiBoldCyan  = "\033[1;36m"
	ansiBoldGreen = "\033[1;32m"
)

// colorEnabled returns true if stderr is a terminal and color should be used.
// Respects the NO_COLOR environment variable (https://no-color.org/).
func colorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

// colorEnabledFd returns true if the given file descriptor supports color.
// Respects the NO_COLOR environment variable (https://no-color.org/).
func colorEnabledFd(fd uintptr) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// styled wraps text with ANSI codes if color is enabled.
// Returns plain text if color is false.
func styled(text, code string, color bool) string {
	if !color {
		return text
	}
	return fmt.Sprintf("%s%s%s", code, text, ansiReset)
}

// bold returns text in bold if color is enabled.
func bold(text string, color bool) string {
	return styled(text, ansiBold, color)
}

// dim returns text in dim if color is enabled.
func dim(text string, color bool) string {
	return styled(text, ansiDim, color)
}

// cyan returns text in cyan if color is enabled.
func cyan(text string, color bool) string {
	return styled(text, ansiCyan, color)
}

// green returns text in green if color is enabled.
func green(text string, color bool) string {
	return styled(text, ansiGreen, color)
}

// yellow returns text in yellow if color is enabled.
func yellow(text string, color bool) string {
	return styled(text, ansiYellow, color)
}

// boldCyan returns text in bold cyan if color is enabled.
func boldCyan(text string, color bool) string {
	return styled(text, ansiBoldCyan, color)
}

// boldGreen returns text in bold green if color is enabled.
func boldGreen(text string, color bool) string {
	return styled(text, ansiBoldGreen, color)
}
