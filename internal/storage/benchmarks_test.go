package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/mock"
)

var (
	addSequentialCount      = benchmarkEnvInt("TMUX_INTRAY_BENCH_ADD_SEQUENTIAL_COUNT", 1000)
	listActiveCount         = benchmarkEnvInt("TMUX_INTRAY_BENCH_LIST_ACTIVE_COUNT", 10000)
	markReadOps             = benchmarkEnvInt("TMUX_INTRAY_BENCH_MARK_READ_OPS", 100)
	dismissOps              = benchmarkEnvInt("TMUX_INTRAY_BENCH_DISMISS_OPS", 100)
	cleanupOldCount         = benchmarkEnvInt("TMUX_INTRAY_BENCH_CLEANUP_OLD_COUNT", 10000)
	concurrentGoroutines    = benchmarkEnvInt("TMUX_INTRAY_BENCH_CONCURRENT_GOROUTINES", 10)
	concurrentOpsPerRoutine = benchmarkEnvInt("TMUX_INTRAY_BENCH_CONCURRENT_OPS_PER_ROUTINE", 100)
	largeDatasetCount       = benchmarkEnvInt("TMUX_INTRAY_BENCH_LARGE_DATASET_COUNT", 100000)
)

type benchmarkBackend struct {
	name     string
	newStore func(tb testing.TB) (Storage, func())
	seed     func(tb testing.TB, store Storage, cfg seedConfig) []string
}

type seedConfig struct {
	count      int
	dismissed  int
	read       int
	oldTS      bool
	sessionMod int
	levelMod   int
}

var benchmarkBackends = []benchmarkBackend{
	{
		name:     "tsv",
		newStore: newTSVBenchmarkStore,
		seed:     seedTSVDirect,
	},
	{
		name:     "sqlite_sqlc",
		newStore: newSQLiteBenchmarkStore,
		seed:     seedSQLiteDirect,
	},
}

func BenchmarkStorage_AddSequential1000(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		for i := 0; i < addSequentialCount; i++ {
			_, err := store.AddNotification(
				fmt.Sprintf("message-%d", i),
				benchmarkTimestamp(i, false),
				fmt.Sprintf("session-%d", i%10),
				fmt.Sprintf("window-%d", i%8),
				fmt.Sprintf("pane-%d", i%6),
				"",
				benchmarkLevel(i, 4),
			)
			if err != nil {
				b.Fatalf("%s add notification: %v", backend.name, err)
			}
		}
	})
}

func BenchmarkStorage_ListActive10000(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: listActiveCount, sessionMod: 10, levelMod: 4})
		b.ResetTimer()
		_, err := store.ListNotifications("active", "", "", "", "", "", "")
		if err != nil {
			b.Fatalf("%s list active: %v", backend.name, err)
		}
	})
}

func BenchmarkStorage_FilterByState(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: listActiveCount, dismissed: listActiveCount / 2, sessionMod: 10, levelMod: 4})
		b.ResetTimer()
		_, err := store.ListNotifications("dismissed", "", "", "", "", "", "")
		if err != nil {
			b.Fatalf("%s filter by state: %v", backend.name, err)
		}
	})
}

func BenchmarkStorage_FilterByLevel(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: listActiveCount, sessionMod: 10, levelMod: 4})
		b.ResetTimer()
		_, err := store.ListNotifications("all", "error", "", "", "", "", "")
		if err != nil {
			b.Fatalf("%s filter by level: %v", backend.name, err)
		}
	})
}

func BenchmarkStorage_FilterBySession(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: listActiveCount, sessionMod: 20, levelMod: 4})
		b.ResetTimer()
		_, err := store.ListNotifications("all", "", "session-7", "", "", "", "")
		if err != nil {
			b.Fatalf("%s filter by session: %v", backend.name, err)
		}
	})
}

func BenchmarkStorage_MarkRead100(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		ids := backend.seed(b, store, seedConfig{count: 1000, sessionMod: 10, levelMod: 4})
		b.ResetTimer()
		for i := 0; i < markReadOps; i++ {
			if err := store.MarkNotificationRead(ids[i]); err != nil {
				b.Fatalf("%s mark read: %v", backend.name, err)
			}
		}
	})
}

func BenchmarkStorage_Dismiss100(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		ids := backend.seed(b, store, seedConfig{count: 1000, sessionMod: 10, levelMod: 4})
		b.ResetTimer()
		for i := 0; i < dismissOps; i++ {
			if err := store.DismissNotification(ids[i]); err != nil {
				b.Fatalf("%s dismiss notification: %v", backend.name, err)
			}
		}
	})
}

func BenchmarkStorage_CleanupOld10000(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{
			count:      cleanupOldCount,
			dismissed:  cleanupOldCount,
			oldTS:      true,
			sessionMod: 10,
			levelMod:   4,
		})
		b.ResetTimer()
		if err := store.CleanupOldNotifications(30, false); err != nil {
			b.Fatalf("%s cleanup old notifications: %v", backend.name, err)
		}
	})
}

func BenchmarkStorage_ConcurrentList10x100(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: listActiveCount, sessionMod: 20, levelMod: 4})
		b.ResetTimer()
		var wg sync.WaitGroup
		for g := 0; g < concurrentGoroutines; g++ {
			g := g
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < concurrentOpsPerRoutine; i++ {
					session := fmt.Sprintf("session-%d", (g+i)%20)
					_, err := store.ListNotifications("active", "", session, "", "", "", "")
					if err != nil {
						b.Errorf("%s concurrent list: %v", backend.name, err)
						return
					}
				}
			}()
		}
		wg.Wait()
	})
}

func BenchmarkStorage_LargeDatasetQuery100000(b *testing.B) {
	benchmarkBothBackends(b, func(b *testing.B, backend benchmarkBackend, store Storage) {
		backend.seed(b, store, seedConfig{count: largeDatasetCount, sessionMod: 25, levelMod: 4})
		b.ResetTimer()
		_, err := store.ListNotifications("active", "", "session-7", "", "", "", "")
		if err != nil {
			b.Fatalf("%s large dataset query: %v", backend.name, err)
		}
	})
}

func benchmarkBothBackends(b *testing.B, run func(b *testing.B, backend benchmarkBackend, store Storage)) {
	b.ReportAllocs()
	for _, backend := range benchmarkBackends {
		backend := backend
		b.Run(backend.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				store, cleanup := backend.newStore(b)
				b.StartTimer()
				run(b, backend, store)
				b.StopTimer()
				cleanup()
			}
		})
	}
}

func newTSVBenchmarkStore(tb testing.TB) (Storage, func()) {
	tb.Helper()
	stateDir := tb.TempDir()
	tb.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	tb.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	Reset()
	mockClient := new(tmux.MockClient)
	mockClient.On("HasSession").Return(true, nil)
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(mockClient)

	store, err := NewFileStorage()
	if err != nil {
		tb.Fatalf("new tsv benchmark storage: %v", err)
	}

	cleanup := func() {
		Reset()
	}

	return store, cleanup
}

func newSQLiteBenchmarkStore(tb testing.TB) (Storage, func()) {
	tb.Helper()
	tb.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	dbPath := filepath.Join(tb.TempDir(), "notifications.db")
	mockClient := new(tmux.MockClient)
	mockClient.On("HasSession").Return(true, nil)
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	sqlite.SetTmuxClient(mockClient)

	store, err := sqlite.NewSQLiteStorage(dbPath)
	if err != nil {
		tb.Fatalf("new sqlite benchmark storage: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
	}

	return store, cleanup
}

func seedTSVDirect(tb testing.TB, store Storage, cfg seedConfig) []string {
	tb.Helper()

	_ = store
	if cfg.count == 0 {
		return nil
	}
	if cfg.sessionMod <= 0 {
		cfg.sessionMod = 1
	}
	if cfg.levelMod <= 0 {
		cfg.levelMod = 4
	}

	var builder strings.Builder
	ids := make([]string, 0, cfg.count)
	for i := 0; i < cfg.count; i++ {
		id := i + 1
		state := "active"
		if i < cfg.dismissed {
			state = "dismissed"
		}
		readTimestamp := ""
		if i < cfg.read {
			readTimestamp = benchmarkTimestamp(i, cfg.oldTS)
		}
		builder.WriteString(fmt.Sprintf(
			"%d\t%s\t%s\tsession-%d\twindow-%d\tpane-%d\tmessage-%d\t\t%s\t%s\n",
			id,
			benchmarkTimestamp(i, cfg.oldTS),
			state,
			i%cfg.sessionMod,
			i%8,
			i%6,
			i,
			benchmarkLevel(i, cfg.levelMod),
			readTimestamp,
		))
		ids = append(ids, strconv.Itoa(id))
	}

	if err := os.WriteFile(notificationsFile, []byte(builder.String()), FileModeFile); err != nil {
		tb.Fatalf("seed tsv fixture: %v", err)
	}

	return ids
}

func seedSQLiteDirect(tb testing.TB, store Storage, cfg seedConfig) []string {
	tb.Helper()

	if cfg.count == 0 {
		return nil
	}
	if cfg.sessionMod <= 0 {
		cfg.sessionMod = 1
	}
	if cfg.levelMod <= 0 {
		cfg.levelMod = 4
	}

	ids := make([]string, 0, cfg.count)
	for i := 0; i < cfg.count; i++ {
		id, err := store.AddNotification(
			fmt.Sprintf("message-%d", i),
			benchmarkTimestamp(i, cfg.oldTS),
			fmt.Sprintf("session-%d", i%cfg.sessionMod),
			fmt.Sprintf("window-%d", i%8),
			fmt.Sprintf("pane-%d", i%6),
			"",
			benchmarkLevel(i, cfg.levelMod),
		)
		if err != nil {
			tb.Fatalf("seed sqlite add notification: %v", err)
		}
		ids = append(ids, id)
	}

	if cfg.dismissed == cfg.count {
		if err := store.DismissAll(); err != nil {
			tb.Fatalf("seed sqlite dismiss all: %v", err)
		}
	} else {
		for i := 0; i < cfg.dismissed && i < len(ids); i++ {
			if err := store.DismissNotification(ids[i]); err != nil {
				tb.Fatalf("seed sqlite dismiss notification: %v", err)
			}
		}
	}

	for i := 0; i < cfg.read && i < len(ids); i++ {
		if err := store.MarkNotificationRead(ids[i]); err != nil {
			tb.Fatalf("seed sqlite mark read: %v", err)
		}
	}

	return ids
}

func benchmarkTimestamp(i int, old bool) string {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if old {
		base = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return base.Add(time.Second * time.Duration(i)).Format(time.RFC3339)
}

func benchmarkLevel(i, levelMod int) string {
	levels := []string{"info", "warning", "error", "critical"}
	return levels[i%minInt(levelMod, len(levels))]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func benchmarkEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
