package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func resetMigrateFlags() {
	migrateTSVPathFlag = ""
	migrateSQLitePathFlag = ""
	migrateBackupPathFlag = ""
	migrateDryRunFlag = false
	migrateRollbackFlag = false
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

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := migrateCmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, out.String(), "migration completed")
	require.Contains(t, out.String(), "total=1 migrated=1 skipped=0 failed=0")

	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupPath)
	require.True(t, os.IsNotExist(err))
}

func TestMigrateCommandRejectsDryRunRollbackCombination(t *testing.T) {
	defer resetMigrateFlags()

	migrateDryRunFlag = true
	migrateRollbackFlag = true

	err := migrateCmd.RunE(&cobra.Command{}, nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "--dry-run cannot be combined with --rollback"))
}
