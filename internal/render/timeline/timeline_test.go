package timeline

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/render/internal/ui"
	"github.com/gian-gg/metron/internal/store/db"
)

func span(trace, id, name, kind, status string, startMs, endMs int64) db.ListSessionSpansRow {
	return db.ListSessionSpansRow{
		TraceID: trace, SpanID: id, Name: name, Kind: kind, Status: status,
		StartTime: startMs * int64(time.Millisecond), EndTime: endMs * int64(time.Millisecond),
	}
}

func timelineFixture() (db.Session, []db.ListSessionSpansRow) {
	tag, end := "v1", (118 * time.Second).Nanoseconds()
	session := db.Session{ID: "a1b2c3d4e5f6", Tag: &tag, EndTime: &end}

	model, in, out := "deepseek-v4-flash", int64(23908), int64(48)
	chat := span("5b8efff798038103", "s2", "model deepseek", "model", "unset", 1_100, 3_200)
	chat.Model, chat.InputTokens, chat.OutputTokens = &model, &in, &out
	zero := int64(0)
	tool := span("5b8efff798038103", "s3", "tool zoho_projects_read", "tool", "error", 3_200, 4_900)
	tool.InputTokens = &zero

	return session, []db.ListSessionSpansRow{
		span("5b8efff798038103", "s1", "assistant.run", "other", "unset", 1_000, 8_000),
		chat,
		tool,
		span("c0ffee00c0ffee00", "s4", "assistant.run", "other", "unset", 9_000, 9_500),
	}
}

func TestTimeline(t *testing.T) {
	session, spans := timelineFixture()
	var buf bytes.Buffer

	if err := Render(&buf, "", []db.Session{session}, spans); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"session a1b2c3d4", "tag v1", "done", "1m58s",
		"trace 5b8efff7", "7s", "3 spans", "1 error",
		"trace c0ffee00", "1 span",
		"+0s", "+100ms", "+2.2s",
		"model deepseek", "deepseek-v4-flash", "2.1s", "23908 in · 48 out",
		"tool zoho_projects_read", "✗ error",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Count(out, "trace ") != 2 {
		t.Errorf("want two trace blocks:\n%s", out)
	}
	if strings.Contains(out, "0 in") {
		t.Errorf("zero tokens should be hidden:\n%s", out)
	}
	if strings.Index(out, "assistant.run") > strings.Index(out, "model deepseek") {
		t.Errorf("rows are not in start order:\n%s", out)
	}
}

func TestTimelineNoSpans(t *testing.T) {
	var buf bytes.Buffer

	if err := Render(&buf, "", []db.Session{{ID: "a1b2c3d4e5f6"}}, nil); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "session a1b2c3d4") || !strings.Contains(out, "No spans yet") {
		t.Errorf("want header and an empty hint:\n%s", out)
	}
}

func TestTimelineIsPlainWhenPiped(t *testing.T) {
	session, spans := timelineFixture()
	var buf bytes.Buffer

	if err := Render(&buf, "", []db.Session{session}, spans); err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output contains color codes:\n%q", buf.String())
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("trailing space: %q", line)
		}
	}
}

func TestTokens(t *testing.T) {
	in, out, zero := int64(120), int64(30), int64(0)
	tests := []struct {
		name    string
		in, out *int64
		want    string
	}{
		{"both", &in, &out, "120 in · 30 out"},
		{"input only", &in, nil, "120 in"},
		{"output only", nil, &out, "30 out"},
		{"none", nil, nil, ui.Placeholder},
		{"zeros are hidden", &zero, &zero, ui.Placeholder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokens(tt.in, tt.out); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{{1, "1 error"}, {2, "2 errors"}}
	for _, tt := range tests {
		if got := errorCount(tt.n); got != tt.want {
			t.Errorf("errorCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestTimelineForATagNamesTheTagAndItsSessions(t *testing.T) {
	_, spans := timelineFixture()
	end := int64(5)
	sessions := []db.Session{{ID: "aaaaaaaaaaaa", EndTime: &end}, {ID: "bbbbbbbbbbbb"}}
	var buf bytes.Buffer

	if err := Render(&buf, "v1", sessions, spans); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := strings.SplitN(buf.String(), "\n", 2)[0]
	for _, want := range []string{"tag v1", "2 sessions", "running"} {
		if !strings.Contains(out, want) {
			t.Errorf("header %q missing %q", out, want)
		}
	}
	if strings.Contains(out, "session aaaaaaaa") {
		t.Errorf("header should name the tag, not one session: %q", out)
	}
}
