package app

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/stretchr/testify/assert"
)

type fakeSearchProviderFactory struct {
	calls int
	regex []bool
}

func (f *fakeSearchProviderFactory) Build(regex bool) search.Provider {
	f.calls++
	f.regex = append(f.regex, regex)
	return search.NewSubstringProvider(search.WithFields([]string{"level"}))
}

type fakeListClient struct {
	result string
	err    error
	calls  []struct {
		state      string
		level      string
		session    string
		window     string
		pane       string
		olderThan  string
		newerThan  string
		readFilter string
	}
}

func (f *fakeListClient) ListNotifications(state, level, session, window, pane, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	f.calls = append(f.calls, struct {
		state      string
		level      string
		session    string
		window     string
		pane       string
		olderThan  string
		newerThan  string
		readFilter string
	}{
		state:      state,
		level:      level,
		session:    session,
		window:     window,
		pane:       pane,
		olderThan:  olderThanCutoff,
		newerThan:  newerThanCutoff,
		readFilter: readFilter,
	})
	return f.result, f.err
}

func testLines() string {
	return `1	2025-01-01T10:00:00Z	active	sess1	win1	pane1	message one	123	info	2025-01-01T10:05:00Z
2	2025-01-01T11:00:00Z	active	sess1	win1	pane2	message two	124	warning	
3	2025-01-01T12:00:00Z	dismissed	sess2	win2	pane3	message three	125	error	2025-01-01T12:05:00Z
4	2025-01-01T13:00:00Z	active	sess2	win2	pane4	message four	126	info	
5	2025-01-01T14:00:00Z	active	sess3	win3	pane5	message five	127	info	2025-01-01T14:05:00Z`
}

func TestListUseCaseExecuteEmpty(t *testing.T) {
	client := &fakeListClient{result: ""}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{Format: "simple"}, &buf)

	assert.Equal(t, "\033[0;34mNo notifications found\033[0m\n", buf.String())
}

func TestListUseCaseExecuteClientError(t *testing.T) {
	client := &fakeListClient{err: errors.New("storage error")}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{}, &buf)

	assert.Contains(t, buf.String(), "list: failed to list notifications: storage error")
}

func TestListUseCaseExecuteUnreadFirstOrdering(t *testing.T) {
	client := &fakeListClient{result: testLines()}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{Format: "simple"}, &buf)

	output := strings.TrimSpace(buf.String())
	var ids []int
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("failed to parse ID from line %q: %v", line, err)
		}
		ids = append(ids, id)
	}

	assert.Equal(t, []int{2, 4, 1, 3, 5}, ids)
}

func TestListUseCaseExecuteWithCustomSearchProvider(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\terror message\t123\terror\n" +
		"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\twarning message\t124\twarning\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	provider := search.NewSubstringProvider(search.WithFields([]string{"level"}))
	useCase.Execute(ListOptions{
		Search:         "error",
		SearchProvider: provider,
		Format:         "legacy",
	}, &buf)

	assert.Contains(t, buf.String(), "error message")
	assert.NotContains(t, buf.String(), "warning message")
}

func TestListUseCaseFetchesThroughClient(t *testing.T) {
	client := &fakeListClient{result: testLines()}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{
		State:      "active",
		Level:      "warning",
		Session:    "sess1",
		Window:     "win1",
		Pane:       "pane2",
		OlderThan:  "2026-01-01T00:00:00Z",
		NewerThan:  "2026-01-02T00:00:00Z",
		ReadFilter: "unread",
		Format:     "simple",
	}, &buf)

	if assert.Len(t, client.calls, 1) {
		call := client.calls[0]
		assert.Equal(t, "active", call.state)
		assert.Equal(t, "warning", call.level)
		assert.Equal(t, "sess1", call.session)
		assert.Equal(t, "win1", call.window)
		assert.Equal(t, "pane2", call.pane)
		assert.Equal(t, "2026-01-01T00:00:00Z", call.olderThan)
		assert.Equal(t, "2026-01-02T00:00:00Z", call.newerThan)
		assert.Equal(t, "unread", call.readFilter)
	}
}

func TestListUseCaseExecuteBuildsSearchProviderFromInjectedFactory(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\terror message\t123\terror\n" +
		"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\twarning message\t124\twarning\n"}
	factory := &fakeSearchProviderFactory{}
	useCase := NewListUseCase(client, factory.Build)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{
		Search: "error",
		Format: "legacy",
	}, &buf)

	assert.Equal(t, 1, factory.calls)
	assert.Equal(t, []bool{false}, factory.regex)
	assert.Contains(t, buf.String(), "error message")
	assert.NotContains(t, buf.String(), "warning message")
}

func TestListUseCaseHidesStaleTmuxRowsByDefault(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\t$1\t@1\t%1\tlive target\t123\tinfo\t\n" +
		"2\t2025-01-01T10:01:00Z\tactive\t$180\t@329\t%703\tstale target\t124\tinfo\t\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{Format: "simple", DisplayNames: DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{"@1": "editor"},
		Panes:    map[string]string{"%1": "shell"},
	}}, &buf)

	output := buf.String()
	assert.Contains(t, output, "live target")
	assert.NotContains(t, output, "stale target")
	assert.NotContains(t, output, "stale-session:$180")
}

func TestListUseCaseShowsStaleTmuxRowsWhenRequested(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\t$180\t@329\t%703\tstale target\t123\tinfo\t\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{Format: "simple", DisplayNames: DisplayNames{}, ShowStale: true}, &buf)

	output := buf.String()
	assert.Contains(t, output, "stale-session:$180")
	assert.Contains(t, output, "stale-window:@329")
	assert.Contains(t, output, "stale-pane:%703")
	assert.NotContains(t, output, "\t$180\t@329\t%703\t")
}

func TestListUseCaseGroupedOutputUsesReadableFallbackForStaleTmuxIDs(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\t$180\t@329\t%703\tstale target\t123\tinfo\t\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{Format: "simple", GroupBy: "session", GroupCount: true, DisplayNames: DisplayNames{}, ShowStale: true}, &buf)

	assert.Contains(t, buf.String(), "Group: stale-session:$180 (1)")
}

func TestListUseCaseGroupedOutputKeepsRawSessionIdentityWhenDisplayNamesCollide(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\t$1\t@1\t%1\tmessage one\t123\tinfo\t\n" +
		"2\t2025-01-01T11:00:00Z\tactive\t$2\t@2\t%2\tmessage two\t124\twarning\t\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{
		Format:     "simple",
		GroupBy:    "session",
		GroupCount: true,
		DisplayNames: DisplayNames{
			Sessions: map[string]string{"$1": "work", "$2": "work"},
		},
		ShowStale: true,
	}, &buf)

	output := buf.String()
	assert.Contains(t, output, "Group: work (1)")
	assert.Equal(t, 2, strings.Count(output, "Group: work (1)"))
}

func TestListUseCaseGroupedOutputKeepsRawWindowIdentityWhenDisplayNamesCollide(t *testing.T) {
	client := &fakeListClient{result: "1\t2025-01-01T10:00:00Z\tactive\t$1\t@1\t%1\tmessage one\t123\tinfo\t\n" +
		"2\t2025-01-01T11:00:00Z\tactive\t$2\t@2\t%2\tmessage two\t124\twarning\t\n"}
	useCase := NewListUseCase(client, nil)

	var buf bytes.Buffer
	useCase.Execute(ListOptions{
		Format:     "simple",
		GroupBy:    "window",
		GroupCount: true,
		DisplayNames: DisplayNames{
			Windows: map[string]string{"@1": "editor", "@2": "editor"},
		},
		ShowStale: true,
	}, &buf)

	output := buf.String()
	assert.Contains(t, output, "Group: editor (1)")
	assert.Equal(t, 2, strings.Count(output, "Group: editor (1)"))
}

func TestOrderUnreadFirstPreservesRelativeOrder(t *testing.T) {
	notifs := []*domain.Notification{
		{ID: 1, ReadTimestamp: "2025-01-01T10:00:00Z", State: domain.StateActive, Level: domain.LevelInfo, Message: "test1", Timestamp: "2025-01-01T10:00:00Z"},
		{ID: 2, ReadTimestamp: "", State: domain.StateActive, Level: domain.LevelInfo, Message: "test2", Timestamp: "2025-01-01T11:00:00Z"},
		{ID: 3, ReadTimestamp: "2025-01-01T11:00:00Z", State: domain.StateActive, Level: domain.LevelInfo, Message: "test3", Timestamp: "2025-01-01T12:00:00Z"},
		{ID: 4, ReadTimestamp: "", State: domain.StateActive, Level: domain.LevelInfo, Message: "test4", Timestamp: "2025-01-01T13:00:00Z"},
	}

	result := OrderUnreadFirst(notifs)
	assert.Equal(t, []*domain.Notification{notifs[1], notifs[3], notifs[0], notifs[2]}, result)
}
