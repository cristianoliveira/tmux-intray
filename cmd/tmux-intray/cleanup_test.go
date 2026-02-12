package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var _ = (*cobra.Command)(nil)

type fakeCleanupClient struct {
	ensureTmuxRunningResult bool
	ensureCalls             int
	cleanupCalled           bool
	cleanupErr              error
	captured                struct {
		days   int
		dryRun bool
	}
}

func (f *fakeCleanupClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeCleanupClient) CleanupOldNotifications(days int, dryRun bool) error {
	f.cleanupCalled = true
	f.captured.days = days
	f.captured.dryRun = dryRun
	return f.cleanupErr
}

func TestNewCleanupCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewCleanupCmd(nil)
}

func TestCleanupCmdRunETmuxNotRunningError(t *testing.T) {
	t.Setenv("TMUX_INTRAY_AUTO_CLEANUP_DAYS", "30")
	client := &fakeCleanupClient{ensureTmuxRunningResult: false}
	cmd := NewCleanupCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no tmux session running") {
		t.Fatalf("expected error about tmux, got %q", err.Error())
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
	}
	if client.cleanupCalled {
		t.Fatal("expected CleanupOldNotifications not to be called")
	}
}

func TestCleanupCmdRunESuccessWithDefaultDays(t *testing.T) {
	t.Setenv("TMUX_INTRAY_AUTO_CLEANUP_DAYS", "45")
	client := &fakeCleanupClient{ensureTmuxRunningResult: true}
	cmd := NewCleanupCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
	}
	if !client.cleanupCalled {
		t.Fatal("expected CleanupOldNotifications to be called")
	}
	if client.captured.days != 45 {
		t.Fatalf("expected days=45, got %d", client.captured.days)
	}
	if client.captured.dryRun {
		t.Fatal("expected dryRun=false")
	}
	// Check that output contains expected message
	if !strings.Contains(output.String(), "Starting cleanup of notifications dismissed more than 45 days ago") {
		t.Fatalf("expected output to contain cleanup message, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Cleanup completed") {
		t.Fatalf("expected output to contain completion message, got %q", output.String())
	}
}

func TestCleanupCmdRunESuccessWithDaysFlag(t *testing.T) {
	client := &fakeCleanupClient{ensureTmuxRunningResult: true}
	cmd := NewCleanupCmd(client)
	setFlag(t, cmd, "days", "7")

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.captured.days != 7 {
		t.Fatalf("expected days=7, got %d", client.captured.days)
	}
}

func TestCleanupCmdRunEWithDryRun(t *testing.T) {
	client := &fakeCleanupClient{ensureTmuxRunningResult: true}
	cmd := NewCleanupCmd(client)
	setFlag(t, cmd, "dryrun", "true")

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !client.captured.dryRun {
		t.Fatal("expected dryRun=true")
	}
}

func TestCleanupCmdRunEDaysZeroUsesConfig(t *testing.T) {
	t.Setenv("TMUX_INTRAY_AUTO_CLEANUP_DAYS", "99")
	client := &fakeCleanupClient{ensureTmuxRunningResult: true}
	cmd := NewCleanupCmd(client)
	setFlag(t, cmd, "days", "0")

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.captured.days != 99 {
		t.Fatalf("expected days=99 from config, got %d", client.captured.days)
	}
}

func TestCleanupCmdRunEDaysNegativeError(t *testing.T) {
	client := &fakeCleanupClient{ensureTmuxRunningResult: true}
	cmd := NewCleanupCmd(client)
	setFlag(t, cmd, "days", "-5")

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "days must be a positive integer") {
		t.Fatalf("expected error about positive integer, got %q", err.Error())
	}
	if client.cleanupCalled {
		t.Fatal("expected CleanupOldNotifications not to be called")
	}
}

func TestCleanupCmdRunECleanupError(t *testing.T) {
	client := &fakeCleanupClient{
		ensureTmuxRunningResult: true,
		cleanupErr:              errors.New("storage error"),
	}
	cmd := NewCleanupCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cleanup failed: storage error") {
		t.Fatalf("expected wrapped cleanup error, got %q", err.Error())
	}
	if !client.cleanupCalled {
		t.Fatal("expected CleanupOldNotifications to be called")
	}
}
