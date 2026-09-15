package tui

import (
	"testing"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/assert/check"
)

func TestInteger(t *testing.T) {
	p := newPrompt(gotext.NewLocale("", "en"))
	unsigned, signed := p.integer(true, false), p.integer(false, true)

	check.NoError(t, unsigned("8080"))
	check.NoError(t, unsigned("0"))
	check.Error(t, unsigned(""))
	check.Error(t, unsigned("08"))
	check.Error(t, unsigned("-1"))
	check.Error(t, unsigned("1a"))

	check.NoError(t, signed(""))
	check.NoError(t, signed("-1"))
	check.Error(t, signed("-"))
	check.Error(t, signed("-01"))
}

func TestDisplay(t *testing.T) {
	check.Equal(t, display("plain", false), "plain")
	check.Equal(t, display("hi u", false), "'hi u'")
	check.Equal(t, display("it's", false), `'it'\''s'`)
	check.Equal(t, display("secret", true), "******")
}
