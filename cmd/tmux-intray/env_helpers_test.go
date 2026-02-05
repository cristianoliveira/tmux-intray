package main

import "testing"

func TestAllowTmuxlessMode(t *testing.T) {
	t.Run("explicit override", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
		if !allowTmuxlessMode() {
			t.Fatalf("expected override to allow tmuxless mode")
		}
	})

	t.Run("bats", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
		t.Setenv("BATS_TMPDIR", "/tmp/bats")
		if !allowTmuxlessMode() {
			t.Fatalf("expected BATS env to allow tmuxless mode")
		}
	})

	t.Run("ci", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
		t.Setenv("BATS_TMPDIR", "")
		t.Setenv("CI", "true")
		if !allowTmuxlessMode() {
			t.Fatalf("expected CI env to allow tmuxless mode")
		}
	})

	t.Run("tmux_available_zero", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
		t.Setenv("BATS_TMPDIR", "")
		t.Setenv("CI", "")
		t.Setenv("TMUX_AVAILABLE", "0")
		if !allowTmuxlessMode() {
			t.Fatalf("expected TMUX_AVAILABLE=0 to allow tmuxless mode")
		}
	})

	t.Run("default_disallows", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
		t.Setenv("BATS_TMPDIR", "")
		t.Setenv("CI", "")
		t.Setenv("TMUX_AVAILABLE", "")
		if allowTmuxlessMode() {
			t.Fatalf("expected tmux requirement when no env is set")
		}
	})
}
