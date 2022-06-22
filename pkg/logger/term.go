package logger

import (
	"io"
	"os"
	"runtime"

	"github.com/mattn/go-isatty"
)

// Code in this file is taken from 'kind' project at:
// https://github.com/kubernetes-sigs/kind/blob/main/pkg/internal/env/term.go

// IsTerminal returns true if the writer w is a terminal
func IsTerminal(w io.Writer) bool {
	if v, ok := (w).(*os.File); ok {
		return isatty.IsTerminal(v.Fd())
	}
	return false
}

// IsSmartTerminal returns true if the writer w is a terminal AND
// we think that the terminal is smart enough to use VT escape codes etc.
func IsSmartTerminal(w io.Writer) bool {
	return isSmartTerminal(w, runtime.GOOS, os.LookupEnv)
}

func isSmartTerminal(w io.Writer, GOOS string, lookupEnv func(string) (string, bool)) bool {
	// Not smart if it's not a tty
	if !IsTerminal(w) {
		return false
	}

	// getenv helper for when we only care about the value
	getenv := func(e string) string {
		v, _ := lookupEnv(e)
		return v
	}

	// Explicit request for no ANSI escape codes
	// https://no-color.org/
	if _, set := lookupEnv("NO_COLOR"); set {
		return false
	}

	// Explicitly dumb terminals are not smart
	// https://en.wikipedia.org/wiki/Computer_terminal#Dumb_terminals
	term := getenv("TERM")
	if term == "dumb" {
		return false
	}
	// st has some bug 🤷‍♂️
	// https://github.com/kubernetes-sigs/kind/issues/1892
	if term == "st-256color" {
		return false
	}

	// On Windows WT_SESSION is set by the modern terminal component.
	// Older terminals have poor support for UTF-8, VT escape codes, etc.
	if GOOS == "windows" && getenv("WT_SESSION") == "" {
		return false
	}

	/* CI Systems with bad Fake TTYs */
	// Travis CI
	// https://github.com/kubernetes-sigs/kind/issues/1478
	// We can detect it with documented magical environment variables
	// https://docs.travis-ci.com/user/environment-variables/#default-environment-variables
	if getenv("HAS_JOSH_K_SEAL_OF_APPROVAL") == "true" && getenv("TRAVIS") == "true" {
		return false
	}

	// OK, we'll assume it's smart now, given no evidence otherwise.
	return true
}
