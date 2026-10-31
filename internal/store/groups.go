package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/gian-gg/metron/internal/store/db"
)

// Group is what a name refers to: every session sharing Tag, or one session addressed by ID.
type Group struct {
	Tag      string
	Sessions []db.Session
}

// FindGroup resolves name as a tag first, then as a session ID prefix.
func (s *Store) FindGroup(ctx context.Context, name string) (Group, error) {
	tagged, err := s.SessionsByTag(ctx, &name)
	if err != nil {
		return Group{}, fmt.Errorf("find sessions by tag: %w", err)
	}
	if len(tagged) > 0 {
		return Group{Tag: name, Sessions: tagged}, nil
	}

	session, err := s.FindSession(ctx, name)
	if err != nil {
		return Group{}, err
	}
	return Group{Sessions: []db.Session{session}}, nil
}

// LatestGroup returns the group of the most recently started session, or false when there are none.
func (s *Store) LatestGroup(ctx context.Context) (Group, bool, error) {
	latest, err := s.ListSessions(ctx, db.ListSessionsParams{Limit: 1})
	if err != nil {
		return Group{}, false, fmt.Errorf("list sessions: %w", err)
	}
	if len(latest) == 0 {
		return Group{}, false, nil
	}
	if tag := latest[0].Tag; tag != nil && *tag != "" {
		group, err := s.FindGroup(ctx, *tag)
		return group, err == nil, err
	}
	return Group{Sessions: latest}, true, nil
}

// GroupSpans returns the spans of every session in g, ordered by start time.
func (s *Store) GroupSpans(ctx context.Context, g Group) ([]db.ListSessionSpansRow, error) {
	var spans []db.ListSessionSpansRow
	for _, session := range g.Sessions {
		rows, err := s.ListSessionSpans(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("list spans of %s: %w", session.ID, err)
		}
		spans = append(spans, rows...)
	}
	slices.SortStableFunc(spans, func(a, b db.ListSessionSpansRow) int {
		return cmp.Or(cmp.Compare(a.StartTime, b.StartTime), cmp.Compare(a.SpanID, b.SpanID))
	})
	return spans, nil
}

// GroupTraceStats returns one row per trace across every session in g, merging a trace split between sessions.
func (s *Store) GroupTraceStats(ctx context.Context, g Group) ([]db.ListSessionTraceStatsRow, error) {
	var merged []db.ListSessionTraceStatsRow
	index := map[string]int{}
	for _, session := range g.Sessions {
		rows, err := s.ListSessionTraceStats(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("trace stats of %s: %w", session.ID, err)
		}
		for _, row := range rows {
			i, seen := index[row.TraceID]
			if !seen {
				index[row.TraceID] = len(merged)
				merged = append(merged, row)
				continue
			}
			merged[i] = mergeTraceStats(merged[i], row)
		}
	}
	return merged, nil
}

func mergeTraceStats(a, b db.ListSessionTraceStatsRow) db.ListSessionTraceStatsRow {
	return db.ListSessionTraceStatsRow{
		TraceID:      a.TraceID,
		Started:      min(a.Started, b.Started),
		Ended:        max(a.Ended, b.Ended),
		Errors:       a.Errors + b.Errors,
		ModelCalls:   a.ModelCalls + b.ModelCalls,
		ToolCalls:    a.ToolCalls + b.ToolCalls,
		InputTokens:  a.InputTokens + b.InputTokens,
		OutputTokens: a.OutputTokens + b.OutputTokens,
	}
}
