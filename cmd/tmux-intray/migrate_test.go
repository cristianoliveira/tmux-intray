package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/stretchr/testify/require"
)

type fakeMigrateClient struct {
	stateDir    string
	migrateOpts sqlite.MigrationOptions
	migrateErr  error
	migrateRes  sqlite.MigrationStats
	rollbackErr error
}

func (f *fakeMigrateClient) GetStateDir() string {
	return f.stateDir
}

func (f *fakeMigrateClient) MigrateToSQLite(opts sqlite.MigrationOptions) (sqlite.MigrationStats, error) {
	f.migrateOpts = opts
	return f.migrateRes, f.migrateErr
}

func (f *fakeMigrateClient) RollbackMigration(tsvPath, sqlitePath, backupPath string) error {
	return f.rollbackErr
}

func resetMigrateFlags() {
	migrateTSVPathFlag = ""
	migrateSQLitePathFlag = ""
	migrateBackupPathFlag = ""
	migrateDryRunFlag = false
	migrateRollbackFlag = false
}

func TestNewMigrateCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewMigrateCmd(nil)
}

func TestMigrateCmdDryRunWithClient(t *testing.T) {
	defer resetMigrateFlags()

	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backupPath := filepath.Join(tmpDir, "notifications.tsv.sqlite-migration.bak")

	line := "1\t2026-01-01T01:00:00Z\tactive\ts1\tw1\tp1\tmsg\t\tinfo\t\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(line), 0o644))

	client := &fakeMigrateClient{
		stateDir: tmpDir,
		migrateRes: sqlite.MigrationStats{
			TotalRows:     1,
			MigratedRows:  1,
			SkippedRows:   0,
			FailedRows:    0,
			DuplicateRows: 0,
		},
	}

	cmd := NewMigrateCmd(client)
	var out bytes.Buffer
	cmd.SetOut(&out)

	// Set flags
	migrateTSVPathFlag = tsvPath
	migrateSQLitePathFlag = dbPath
	migrateBackupPathFlag = backupPath
	migrateDryRunFlag = true

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, out.String(), "migration completed")
	require.Contains(t, out.String(), "total=1 migrated=1 skipped=0 failed=0")

	// Verify DryRun was passed
	require.True(t, client.migrateOpts.DryRun)

	// Verify files were not created (dry run)
	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupPath)
	require.True(t, os.IsNotExist(err))
}

func TestMigrateCmdRejectsDryRunRollbackCombination(t *testing.T) {
	defer resetMigrateFlags()

	client := &fakeMigrateClient{stateDir: "/tmp"}
	cmd := NewMigrateCmd(client)

	migrateDryRunFlag = true
	migrateRollbackFlag = true

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "--dry-run cannot be combined with --rollback"))
}

func TestMigrateCmdRollbackWithClient(t *testing.T) {
	defer resetMigrateFlags()

	client := &fakeMigrateClient{
		stateDir: "/tmp",
	}
	cmd := NewMigrateCmd(client)

	migrateRollbackFlag = true
	migrateTSVPathFlag = "/tmp/test.tsv"
	migrateSQLitePathFlag = "/tmp/test.db"
	migrateBackupPathFlag = "/tmp/test.tsv.bak"

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, out.String(), "rollback completed")
}

func TestMigrateCmdRollbackError(t *testing.T) {
	defer resetMigrateFlags()

	expectedErr := errors.New("rollback failed")
	client := &fakeMigrateClient{
		stateDir:    "/tmp",
		rollbackErr: expectedErr,
	}
	cmd := NewMigrateCmd(client)

	migrateRollbackFlag = true
	migrateTSVPathFlag = "/tmp/test.tsv"

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestMigrateCmdNoStateDir(t *testing.T) {
	defer resetMigrateFlags()

	client := &fakeMigrateClient{
		stateDir: "",
	}
	cmd := NewMigrateCmd(client)

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "state directory is not configured")
}

func TestMigrateCmdMigrateError(t *testing.T) {
	defer resetMigrateFlags()

	expectedErr := errors.New("migration failed")
	client := &fakeMigrateClient{
		stateDir:   "/tmp",
		migrateErr: expectedErr,
	}
	cmd := NewMigrateCmd(client)

	migrateTSVPathFlag = "/tmp/test.tsv"

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestMigrateCommandDryRun(t *testing.T) {
	defer resetMigrateFlags()

	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backupPath := filepath.Join(tmpDir, "notifications.tsv.sqlite-migration.bak")

	line := "1\t2026-01-01T01:00:00Z\tactive\ts1\tw1\tp1\tmsg\t\tinfo\t\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(line), 0o644))

	migrateTSVPathFlag = tsvPath
	migrateSQLitePathFlag = dbPath
	migrateBackupPathFlag = backupPath
	migrateDryRunFlag = true

	// Use the actual migrateCmd to test real behavior
	cmd := migrateCmd
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, out.String(), "migration completed")
	require.Contains(t, out.String(), "total=1 migrated=1 skipped=0 failed=0")

	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupPath)
	require.True(t, os.IsNotExist(err))
}

func TestMigrateCommandRejectsDryRunRollbackCombinationLegacy(t *testing.T) {
	defer resetMigrateFlags()

	migrateDryRunFlag = true
	migrateRollbackFlag = true

	err := migrateCmd.RunE(migrateCmd, nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "--dry-run cannot be combined with --rollback"))
}
