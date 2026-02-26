package formatter

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

func TestTemplateEngine_Parse(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		want     []string
		wantErr  bool
	}{
		{
			name:     "empty template",
			template: "",
			want:     []string{},
			wantErr:  false,
		},
		{
			name:     "no variables",
			template: "Hello world",
			want:     []string{},
			wantErr:  false,
		},
		{
			name:     "single variable",
			template: "Count: ${unread-count}",
			want:     []string{"unread-count"},
			wantErr:  false,
		},
		{
			name:     "multiple different variables",
			template: "${unread-count} unread, ${read-count} read",
			want:     []string{"unread-count", "read-count"},
			wantErr:  false,
		},
		{
			name:     "duplicate variables",
			template: "${unread-count} and ${unread-count}",
			want:     []string{"unread-count"},
			wantErr:  false,
		},
		{
			name:     "hyphens in variable names",
			template: "${total-count} ${latest-message}",
			want:     []string{"total-count", "latest-message"},
			wantErr:  false,
		},
		{
			name:     "numbers in variable names",
			template: "${level1} ${count2}",
			want:     []string{"level1", "count2"},
			wantErr:  false,
		},
		{
			name:     "complex template",
			template: "[${unread-count}] ${latest-message} | Severity: ${highest-severity}",
			want:     []string{"unread-count", "latest-message", "highest-severity"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Parse(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Parse() got %d variables, want %d: %v", len(got), len(tt.want), got)
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("Parse() variable %d: got %s, want %s", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestTemplateEngine_Substitute(t *testing.T) {
	engine := NewTemplateEngine()

	ctx := VariableContext{
		UnreadCount:     5,
		ReadCount:       3,
		TotalCount:      8,
		ActiveCount:     4,
		DismissedCount:  1,
		LatestMessage:   "Test message",
		HasUnread:       true,
		HasActive:       true,
		HasDismissed:    false,
		HighestSeverity: domain.LevelError,
		SessionList:     "session1,session2",
		WindowList:      "window1,window2",
		PaneList:        "pane1,pane2",
	}

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "empty template",
			template: "",
			want:     "",
			wantErr:  false,
		},
		{
			name:     "no variables",
			template: "Hello world",
			want:     "Hello world",
			wantErr:  false,
		},
		{
			name:     "single variable",
			template: "Count: ${unread-count}",
			want:     "Count: 5",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "${unread-count} unread, ${read-count} read",
			want:     "5 unread, 3 read",
			wantErr:  false,
		},
		{
			name:     "total-count alias",
			template: "Total: ${total-count}",
			want:     "Total: 5",
			wantErr:  false,
		},
		{
			name:     "boolean variable true",
			template: "Has unread: ${has-unread}",
			want:     "Has unread: true",
			wantErr:  false,
		},
		{
			name:     "boolean variable false",
			template: "Has dismissed: ${has-dismissed}",
			want:     "Has dismissed: false",
			wantErr:  false,
		},
		{
			name:     "severity mapping",
			template: "Severity: ${highest-severity}",
			want:     "Severity: 2",
			wantErr:  false,
		},
		{
			name:     "latest message",
			template: "Message: ${latest-message}",
			want:     "Message: Test message",
			wantErr:  false,
		},
		{
			name:     "complex template",
			template: "[${unread-count}] ${latest-message}",
			want:     "[5] Test message",
			wantErr:  false,
		},
		{
			name:     "unknown variable replaced with empty",
			template: "Count: ${unknown-var}",
			want:     "Count: ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Substitute(tt.template, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Substitute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Substitute() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_ValidateTemplate(t *testing.T) {
	te := NewTemplateEngine()
	concrete, ok := te.(*templateEngine)
	if !ok {
		t.Fatal("failed to get concrete templateEngine type")
	}

	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "empty template",
			template: "",
			wantErr:  false,
		},
		{
			name:     "valid template",
			template: "Count: ${unread-count}",
			wantErr:  false,
		},
		{
			name:     "mismatched opens",
			template: "Count: ${ ${ unread-count}",
			wantErr:  true,
		},
		{
			name:     "mismatched closes",
			template: "Count: ${unread-count}}",
			wantErr:  true,
		},
		{
			name:     "multiple valid variables",
			template: "${unread-count} ${read-count}",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := concrete.ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateEngine_InvalidVariableSyntax(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "underscore not matched",
			template: "${unread_count}",
			want:     []string{},
		},
		{
			name:     "uppercase not matched",
			template: "${UNREAD-COUNT}",
			want:     []string{},
		},
		{
			name:     "spaces not matched",
			template: "${ unread-count }",
			want:     []string{},
		},
		{
			name:     "valid hyphenated names",
			template: "${unread-count} ${latest-message}",
			want:     []string{"unread-count", "latest-message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := engine.Parse(tt.template)

			if len(got) != len(tt.want) {
				t.Errorf("Parse() got %d variables, want %d", len(got), len(tt.want))
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("Parse() variable %d: got %s, want %s", i, v, tt.want[i])
				}
			}
		})
	}
}
