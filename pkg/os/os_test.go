package os

import (
	"testing"

	"github.com/libtnb/assert/check"
)

func TestIsDebian(t *testing.T) {
	check.True(t, IsDebian(), check.Msgf("os-release: %v", readOSRelease()))
}

func TestIsRHEL(t *testing.T) {
	check.False(t, IsRHEL(), check.Msgf("os-release: %v", readOSRelease()))
}
