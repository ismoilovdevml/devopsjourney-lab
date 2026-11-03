package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devops-lab/backend/internal/models"
)

var day0 = time.Date(2026, 11, 3, 15, 30, 0, 0, time.UTC)

func days(offsets ...int) []string {
	out := make([]string, 0, len(offsets))
	for _, o := range offsets {
		out = append(out, dayKey(day0.AddDate(0, 0, -o)))
	}
	return out
}

func TestComputeStreak(t *testing.T) {
	cases := []struct {
		name string
		days []string
		want int
	}{
		{"none", nil, 0},
		{"today only", days(0), 1},
		{"yesterday only", days(1), 1},
		{"two days ago breaks", days(2, 3, 4), 0},
		{"three ending today", days(0, 1, 2), 3},
		{"three ending yesterday", days(1, 2, 3), 3},
		{"gap stops count", days(0, 1, 3, 4), 2},
		{"duplicates and order", append(days(2, 0, 1), days(0, 1)...), 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeStreak(tc.days, day0); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestStreakReachedAt(t *testing.T) {
	d := days(10, 9, 8, 5, 4, 3, 2, 1, 0) // run of 3 ending at -8, run of 6 ending today
	at := streakReachedAt(d, 3)
	if at == nil || dayKey(*at) != dayKey(day0.AddDate(0, 0, -8)) {
		t.Fatalf("streak-3 reached at %v", at)
	}
	if at := streakReachedAt(d, 7); at != nil {
		t.Fatalf("streak-7 should not be reached, got %v", at)
	}
	if at := streakReachedAt(days(6, 5, 4, 3, 2, 1, 0), 7); at == nil || dayKey(*at) != dayKey(day0) {
		t.Fatalf("streak-7 reached at %v", at)
	}
	if streakReachedAt([]string{"garbage"}, 1) != nil {
		t.Fatal("garbage day must be ignored")
	}
}

func testTasks() []models.Task {
	return []models.Task{
		{ID: "a", Category: "web", Points: 100, Difficulty: 1, Checks: []models.Check{{Name: "c1", Cmd: "secret-a"}}},
		{ID: "b", Category: "containers", Points: 200, Difficulty: 2, Checks: []models.Check{{Name: "c1", Cmd: "secret-b"}, {Name: "c2", Cmd: "x"}}},
		{ID: "c", Category: "web", Points: 300, Difficulty: 3},
		{ID: "d", Category: "users", Points: 100, Difficulty: 1},
	}
}

func TestComputeSkills(t *testing.T) {
	got := computeSkills(testTasks(), map[string]bool{"a": true, "b": true})
	want := []models.SkillProgress{{Category: "web", Percent: 25}, {Category: "containers", Percent: 100}, {Category: "users", Percent: 0}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skill %d: got %v want %v", i, got[i], want[i])
		}
	}
	if got := computeSkills(nil, nil); len(got) != 0 {
		t.Fatalf("expected no skills, got %v", got)
	}
}

func TestPercent(t *testing.T) {
	cases := []struct{ part, whole, want int }{
		{0, 0, 0}, {1, 0, 0}, {0, 10, 0}, {1, 3, 33}, {2, 3, 67}, {3, 3, 100}, {5, 3, 100},
	}
	for _, tc := range cases {
		if got := percent(tc.part, tc.whole); got != tc.want {
			t.Errorf("percent(%d,%d)=%d want %d", tc.part, tc.whole, got, tc.want)
		}
	}
}

func achievementMap(list []models.Achievement) map[string]models.Achievement {
	m := map[string]models.Achievement{}
	for _, a := range list {
		m[a.ID] = a
	}
	return m
}

func TestComputeAchievements(t *testing.T) {
	ts := func(min int) time.Time { return day0.Add(time.Duration(min) * time.Minute) }

	t.Run("nothing done", func(t *testing.T) {
		got := computeAchievements(achievementInput{TotalTasks: 6})
		if len(got) != len(achievementDefs) {
			t.Fatalf("expected %d achievements, got %d", len(achievementDefs), len(got))
		}
		for _, a := range got {
			if a.Earned || a.EarnedAt != nil {
				t.Fatalf("%s should not be earned", a.ID)
			}
			if a.Title == "" || a.Description == "" {
				t.Fatalf("%s missing text", a.ID)
			}
		}
	})

	t.Run("progress milestones", func(t *testing.T) {
		got := achievementMap(computeAchievements(achievementInput{
			TotalTasks:  5,
			Completions: []time.Time{ts(30), ts(10), ts(20)},
		}))
		if a := got["first-blood"]; !a.Earned || !a.EarnedAt.Equal(ts(10)) {
			t.Fatalf("first-blood %+v", a)
		}
		// half of 5 rounds up to 3 completions
		if a := got["half-way"]; !a.Earned || !a.EarnedAt.Equal(ts(30)) {
			t.Fatalf("half-way %+v", a)
		}
		if got["all-done"].Earned {
			t.Fatal("all-done must not be earned with 3/5")
		}
	})

	t.Run("all done", func(t *testing.T) {
		got := achievementMap(computeAchievements(achievementInput{
			TotalTasks: 2, Completions: []time.Time{ts(1), ts(2)},
		}))
		if a := got["all-done"]; !a.Earned || !a.EarnedAt.Equal(ts(2)) {
			t.Fatalf("all-done %+v", a)
		}
	})

	t.Run("zero tasks never all-done", func(t *testing.T) {
		got := achievementMap(computeAchievements(achievementInput{}))
		if got["all-done"].Earned || got["half-way"].Earned {
			t.Fatal("no tasks must not earn progress achievements")
		}
	})

	t.Run("speedrun and perfectionist", func(t *testing.T) {
		got := achievementMap(computeAchievements(achievementInput{
			TotalTasks: 6,
			FirstPass: []attemptRecord{
				{TaskID: "a", Passed: true, ElapsedSeconds: 900, CreatedAt: ts(1)},
				{TaskID: "b", Passed: true, ElapsedSeconds: 299, CreatedAt: ts(5)},
				{TaskID: "c", Passed: true, ElapsedSeconds: 300, CreatedAt: ts(3)},
			},
			FirstAttempt: []attemptRecord{
				{TaskID: "a", Passed: true, CreatedAt: ts(1)},
				{TaskID: "b", Passed: false, CreatedAt: ts(2)},
				{TaskID: "c", Passed: true, CreatedAt: ts(3)},
				{TaskID: "d", Passed: true, CreatedAt: ts(9)},
				{TaskID: "e", Passed: true, CreatedAt: ts(7)},
			},
		}))
		if a := got["speedrun"]; !a.Earned || !a.EarnedAt.Equal(ts(3)) {
			t.Fatalf("speedrun %+v", a)
		}
		if a := got["perfectionist"]; !a.Earned || !a.EarnedAt.Equal(ts(7)) {
			t.Fatalf("perfectionist %+v", a)
		}
	})

	t.Run("streaks", func(t *testing.T) {
		got := achievementMap(computeAchievements(achievementInput{ActivityDays: days(2, 1, 0)}))
		if !got["streak-3"].Earned || got["streak-7"].Earned {
			t.Fatalf("streaks %+v %+v", got["streak-3"], got["streak-7"])
		}
	})
}

func TestRankLeaderboard(t *testing.T) {
	t1 := day0
	t2 := day0.Add(time.Hour)
	rows := []leaderboardRow{
		{UserID: 1, Username: "late", Score: 300, Completed: 2, LastCompleted: &t2},
		{UserID: 2, Username: "early", Score: 300, Completed: 2, LastCompleted: &t1},
		{UserID: 3, Username: "top", Score: 600, Completed: 3, LastCompleted: &t2},
		{UserID: 4, Username: "low", Score: 100, Completed: 1, LastCompleted: &t1},
	}
	got := rankLeaderboard(rows)
	wantOrder := []uint{3, 2, 1, 4}
	wantRank := []int{1, 2, 2, 3}
	for i := range got {
		if got[i].UserID != wantOrder[i] || got[i].Rank != wantRank[i] {
			t.Fatalf("pos %d: got user %d rank %d, want user %d rank %d", i, got[i].UserID, got[i].Rank, wantOrder[i], wantRank[i])
		}
	}
}

func TestBuildLeaderboard(t *testing.T) {
	t1 := day0
	rows := []leaderboardRow{
		{UserID: 1, Username: "a", Score: 300, LastCompleted: &t1},
		{UserID: 2, Username: "b", Score: 200, LastCompleted: &t1},
		{UserID: 3, Username: "c", Score: 100, LastCompleted: &t1},
	}

	resp := buildLeaderboard(rows, 2, 3, "c")
	if len(resp.Entries) != 2 {
		t.Fatalf("limit not applied: %d", len(resp.Entries))
	}
	if resp.Me == nil || resp.Me.UserID != 3 || resp.Me.Rank != 3 {
		t.Fatalf("me outside limit must still be ranked: %+v", resp.Me)
	}

	resp = buildLeaderboard(rows, 50, 9, "nobody")
	if resp.Me == nil || resp.Me.Rank != 4 || resp.Me.Score != 0 || resp.Me.Username != "nobody" {
		t.Fatalf("zero-score me: %+v", resp.Me)
	}

	resp = buildLeaderboard(nil, 50, 9, "solo")
	if resp.Me == nil || resp.Me.Rank != 1 || resp.Entries == nil {
		t.Fatalf("empty board: %+v", resp)
	}
	b, _ := json.Marshal(buildLeaderboard(nil, 50, 0, ""))
	if string(b) != `{"entries":[],"me":null}` {
		t.Fatalf("anonymous JSON: %s", b)
	}
}

func TestPeriodStart(t *testing.T) {
	if _, ok := periodStart("year", day0); ok {
		t.Fatal("unknown period must be rejected")
	}
	for p, want := range map[string]time.Time{
		"week":  day0.AddDate(0, 0, -7),
		"month": day0.AddDate(0, -1, 0),
		"all":   time.Unix(0, 0).UTC(),
		"":      time.Unix(0, 0).UTC(),
	} {
		got, ok := periodStart(p, day0)
		if !ok || !got.Equal(want) {
			t.Errorf("period %q: got %v want %v", p, got, want)
		}
	}
}

func TestBucketSessions(t *testing.T) {
	cur := day0.Unix() / 3600
	got := bucketSessions([]hourCount{
		{Hour: cur, Count: 5},
		{Hour: cur - 1, Count: 2},
		{Hour: cur - 23, Count: 1},
		{Hour: cur - 24, Count: 99}, // too old
		{Hour: cur + 1, Count: 99},  // future (clock skew)
	}, day0)
	if len(got) != 24 || got[23] != 5 || got[22] != 2 || got[0] != 1 {
		t.Fatalf("buckets: %v", got)
	}
	var sum int64
	for _, v := range got {
		sum += v
	}
	if sum != 8 {
		t.Fatalf("sum %d", sum)
	}
}

func TestMergeProgress(t *testing.T) {
	now := day0
	res := func(score, total int) *models.CheckResponse {
		return &models.CheckResponse{Score: score, Total: total, Passed: score == total}
	}

	p, first := mergeProgress(models.Progress{}, res(2, 4), 60, now)
	if first || p.Attempts != 1 || p.Score != 2 || p.Total != 4 || p.Passed || p.CompletedAt != nil || p.TimeSpent != 60 {
		t.Fatalf("first attempt: %+v first=%v", p, first)
	}

	p, first = mergeProgress(p, res(1, 4), 30, now)
	if first || p.Attempts != 2 || p.Score != 2 || p.TimeSpent != 90 {
		t.Fatalf("worse attempt keeps best: %+v", p)
	}

	p, first = mergeProgress(p, res(4, 4), 10, now)
	if !first || !p.Passed || p.Score != 4 || p.CompletedAt == nil || !p.CompletedAt.Equal(now) {
		t.Fatalf("pass: %+v first=%v", p, first)
	}

	later := now.Add(time.Hour)
	p, first = mergeProgress(p, res(0, 4), 5, later)
	if first || !p.Passed || p.Score != 4 || !p.CompletedAt.Equal(now) || p.Attempts != 4 {
		t.Fatalf("passed is sticky and completed_at unchanged: %+v", p)
	}

	p, _ = mergeProgress(p, res(1, 2), -10, later)
	if p.Total != 2 || p.Score > p.Total || p.TimeSpent != 105 {
		t.Fatalf("total change clamps score, negative delta ignored: %+v", p)
	}
}

func TestUnlockAndSummaries(t *testing.T) {
	tasks := testTasks()
	st := emptyTaskState()
	st.completed["a"] = true
	st.inProgress["b"] = true
	st.inProgress["a"] = true // completed wins

	got := buildSummaries(tasks, st, false)
	wantUnlocked := []bool{true, true, false, false}
	for i, s := range got {
		if s.Unlocked != wantUnlocked[i] {
			t.Fatalf("%s unlocked=%v want %v", s.ID, s.Unlocked, wantUnlocked[i])
		}
	}
	if !got[0].Completed || got[0].InProgress || !got[1].InProgress || got[1].ChecksCount != 2 {
		t.Fatalf("state flags: %+v", got[:2])
	}

	for _, s := range buildSummaries(tasks, emptyTaskState(), true) {
		if !s.Unlocked {
			t.Fatalf("admin must see %s unlocked", s.ID)
		}
	}
	anon := buildSummaries(tasks, emptyTaskState(), false)
	if !anon[0].Unlocked || anon[1].Unlocked {
		t.Fatal("anonymous: only the first task is unlocked")
	}
}

func TestTaskDetailNeverLeaksCommands(t *testing.T) {
	tk := testTasks()[1]
	tk.Setup = []string{"apt-get install secret-setup"}
	b, err := json.Marshal(buildDetail(tk, true, emptyTaskState()))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, leak := range []string{"secret-b", "secret-setup", `"cmd"`, `"setup"`} {
		if strings.Contains(s, leak) {
			t.Fatalf("detail JSON leaks %q: %s", leak, s)
		}
	}
	for _, field := range []string{`"hints":[]`, `"checks":[{"name":"c1"},{"name":"c2"}]`, `"checks_count":2`, `"readme"`, `"image"`} {
		if !strings.Contains(s, field) {
			t.Fatalf("detail JSON missing %s: %s", field, s)
		}
	}
	// The raw model must not leak either (defense in depth).
	raw, _ := json.Marshal(tk)
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("models.Task JSON leaks: %s", raw)
	}
}

func TestValidateProfile(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := []struct {
		name    string
		req     models.UpdateMeRequest
		wantErr bool
	}{
		{"empty", models.UpdateMeRequest{}, false},
		{"github ok", models.UpdateMeRequest{Github: str("octo-cat")}, false},
		{"github cleared", models.UpdateMeRequest{Github: str("")}, false},
		{"github bad chars", models.UpdateMeRequest{Github: str("octo_cat")}, true},
		{"github url", models.UpdateMeRequest{Github: str("https://github.com/x")}, true},
		{"goal 200 runes", models.UpdateMeRequest{Goal: str(strings.Repeat("ö", 200))}, false},
		{"goal 201 runes", models.UpdateMeRequest{Goal: str(strings.Repeat("a", 201))}, true},
		{"email blank", models.UpdateMeRequest{Email: str("  ")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, msg := validateProfile(&tc.req)
			if (msg != "") != tc.wantErr {
				t.Fatalf("msg=%q wantErr=%v", msg, tc.wantErr)
			}
		})
	}
	u, _ := validateProfile(&models.UpdateMeRequest{Email: str(" Foo@Example.COM ")})
	if u["email"] != "foo@example.com" {
		t.Fatalf("email not normalized: %v", u)
	}
}

func TestPointsCTE(t *testing.T) {
	sql, args := pointsCTE(testTasks()[:2])
	if strings.Count(sql, "(?::text, ?::int)") != 2 || len(args) != 4 || args[0] != "a" || args[3] != 200 {
		t.Fatalf("cte %s %v", sql, args)
	}
	sql, args = pointsCTE(nil)
	if !strings.Contains(sql, "WHERE false") || args != nil {
		t.Fatalf("empty cte %s", sql)
	}
}
