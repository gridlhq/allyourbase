package cli

import (
	"os"
	"testing"

	"github.com/allyourbase/ayb/internal/testutil"
)

func TestStyledWithColor(t *testing.T) {
	result := styled("hello", ansiBold, true)
	testutil.Equal(t, "\033[1mhello\033[0m", result)
}

func TestStyledWithoutColor(t *testing.T) {
	result := styled("hello", ansiBold, false)
	testutil.Equal(t, "hello", result)
}

func TestBold(t *testing.T) {
	testutil.Equal(t, "\033[1mtest\033[0m", bold("test", true))
	testutil.Equal(t, "test", bold("test", false))
}

func TestDim(t *testing.T) {
	testutil.Equal(t, "\033[2mtest\033[0m", dim("test", true))
	testutil.Equal(t, "test", dim("test", false))
}

func TestCyan(t *testing.T) {
	testutil.Equal(t, "\033[36mtest\033[0m", cyan("test", true))
	testutil.Equal(t, "test", cyan("test", false))
}

func TestGreen(t *testing.T) {
	testutil.Equal(t, "\033[32mtest\033[0m", green("test", true))
	testutil.Equal(t, "test", green("test", false))
}

func TestBoldCyan(t *testing.T) {
	testutil.Equal(t, "\033[1;36mtest\033[0m", boldCyan("test", true))
	testutil.Equal(t, "test", boldCyan("test", false))
}

func TestBoldGreen(t *testing.T) {
	testutil.Equal(t, "\033[1;32mtest\033[0m", boldGreen("test", true))
	testutil.Equal(t, "test", boldGreen("test", false))
}

func TestStyledEmptyString(t *testing.T) {
	testutil.Equal(t, "\033[1m\033[0m", styled("", ansiBold, true))
	testutil.Equal(t, "", styled("", ansiBold, false))
}

func TestColorEnabledRespectsNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	testutil.False(t, colorEnabled())
}

func TestColorEnabledFdRespectsNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	// NO_COLOR is set (even to empty string), so color should be disabled.
	testutil.False(t, colorEnabledFd(os.Stderr.Fd()))
}

func TestColorEnabledNO_COLORNotSet(t *testing.T) {
	// When NO_COLOR is not set, colorEnabled depends on terminal detection.
	// In tests, stderr is not a terminal, so it returns false regardless.
	// But we verify the function doesn't panic and returns a bool.
	os.Unsetenv("NO_COLOR")
	_ = colorEnabled()
	_ = colorEnabledFd(os.Stderr.Fd())
}
