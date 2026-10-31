package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

// seedGroups creates two sessions tagged v1 (one still running) and one untagged session.
func seedGroups(t *testing.T) *store.Store {
	t.Helper()
	s, _ := openTemp(t)
	ctx := context.Background()
	v1 := "v1"
	end5, end9 := int64(5), int64(9)
	for _, p := range []db.CreateSessionParams{
		{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
		{ID: "cccccccccccc", StartTime: 3},
	} {
		if err := s.CreateSession(ctx, p); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	for id, end := range map[string]*int64{"aaaaaaaaaaaa": &end5, "cccccccccccc": &end9} {
		if err := s.FinishSession(ctx, db.FinishSessionParams{ID: id, EndTime: end}); err != nil {
			t.Fatalf("finish session: %v", err)
		}
	}

	first, second, third := spanRow("aaaaaaaaaaaa", "s1"), spanRow("aaaaaaaaaaaa", "s2"), spanRow("bbbbbbbbbbbb", "s3")
	first.StartTime, first.EndTime = 10, 20
	second.StartTime, second.EndTime = 12, 30
	third.TraceID, third.StartTime, third.EndTime = "t2", 11, 25
	if err := s.InsertSpans(ctx, []db.InsertSpanParams{first, second, third}); err != nil {
		t.Fatalf("insert spans: %v", err)
	}
	return s
}

func TestListGroupsMergesSessionsSharingATag(t *testing.T) {
	s := seedGroups(t)

	groups, err := s.ListGroups(context.Background(), db.ListGroupsParams{Limit: 10})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("got %+v, want the untagged session then the v1 group", groups)
	}
	untagged, v1 := groups[0], groups[1]
	if untagged.Name != "cccccccccccc" || untagged.Tagged != 0 || untagged.Sessions != 1 || untagged.Traces != 0 || untagged.Running != 0 || untagged.LastActive != 9 {
		t.Errorf("untagged = %+v", untagged)
	}
	if v1.Name != "v1" || v1.Tagged != 1 || v1.Sessions != 2 || v1.Traces != 2 || v1.Started != 1 || v1.Running != 1 || v1.LastActive != 30 {
		t.Errorf("v1 = %+v", v1)
	}
}

func TestListGroupsFiltersByTag(t *testing.T) {
	s := seedGroups(t)
	v1 := "v1"

	groups, err := s.ListGroups(context.Background(), db.ListGroupsParams{Tag: &v1, Limit: 10})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "v1" {
		t.Fatalf("got %+v, want only v1", groups)
	}
}

func TestFindGroup(t *testing.T) {
	s := seedGroups(t)
	ctx := context.Background()
	tests := []struct {
		name         string
		arg          string
		wantTag      string
		wantSessions int
		wantErr      error
	}{
		{"tag names every session with it", "v1", "v1", 2, nil},
		{"id prefix names one session", "cccc", "", 1, nil},
		{"id prefix of a tagged session stays one session", "aaaa", "", 1, nil},
		{"unknown name", "zzz", "", 0, store.ErrSessionNotFound},
		{"ambiguous id prefix", "", "", 0, store.ErrSessionAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.FindGroup(ctx, tt.arg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got.Tag != tt.wantTag || len(got.Sessions) != tt.wantSessions {
				t.Errorf("got tag %q with %d sessions, want %q with %d", got.Tag, len(got.Sessions), tt.wantTag, tt.wantSessions)
			}
		})
	}
}

func TestLatestGroup(t *testing.T) {
	s := seedGroups(t)
	ctx := context.Background()

	latest, ok, err := s.LatestGroup(ctx)
	if err != nil || !ok {
		t.Fatalf("latest = %v, %v", ok, err)
	}
	if latest.Tag != "" || len(latest.Sessions) != 1 || latest.Sessions[0].ID != "cccccccccccc" {
		t.Errorf("got %+v, want the untagged session", latest)
	}

	v1 := "v1"
	if err := s.CreateSession(ctx, db.CreateSessionParams{ID: "dddddddddddd", Tag: &v1, StartTime: 4}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	latest, _, err = s.LatestGroup(ctx)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.Tag != "v1" || len(latest.Sessions) != 3 {
		t.Errorf("got tag %q with %d sessions, want all three v1 sessions", latest.Tag, len(latest.Sessions))
	}
}

func TestLatestGroupWithoutSessions(t *testing.T) {
	s, _ := openTemp(t)

	_, ok, err := s.LatestGroup(context.Background())

	if err != nil || ok {
		t.Errorf("got ok=%v err=%v, want no group and no error", ok, err)
	}
}

func TestGroupSpansMergesSessionsByStartTime(t *testing.T) {
	s := seedGroups(t)
	group, err := s.FindGroup(context.Background(), "v1")
	if err != nil {
		t.Fatalf("find group: %v", err)
	}

	spans, err := s.GroupSpans(context.Background(), group)
	if err != nil {
		t.Fatalf("group spans: %v", err)
	}
	var got []string
	for _, sp := range spans {
		got = append(got, sp.SpanID)
	}
	if len(got) != 3 || got[0] != "s1" || got[1] != "s3" || got[2] != "s2" {
		t.Errorf("got %v, want s1, s3, s2 in start order across both sessions", got)
	}
}
