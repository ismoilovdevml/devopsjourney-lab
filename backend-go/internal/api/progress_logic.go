package api

import (
	"math"
	"sort"
	"time"

	"github.com/devops-lab/backend/internal/models"
)

// Pure (DB-free) logic behind /api/progress, /api/leaderboard and /api/admin/stats.

const (
	speedrunSeconds     = 5 * 60
	perfectionistNeeded = 3
	dayLayout           = "2006-01-02"
)

// dayKey returns the UTC calendar day of t.
func dayKey(t time.Time) string {
	return t.UTC().Format(dayLayout)
}

// computeStreak counts consecutive UTC days with activity ending today, or yesterday
// when there is no activity yet today. days are "YYYY-MM-DD" strings in any order.
func computeStreak(days []string, now time.Time) int {
	set := make(map[string]bool, len(days))
	for _, d := range days {
		set[d] = true
	}
	cur := now.UTC()
	if !set[dayKey(cur)] {
		cur = cur.AddDate(0, 0, -1)
		if !set[dayKey(cur)] {
			return 0
		}
	}
	n := 0
	for set[dayKey(cur)] {
		n++
		cur = cur.AddDate(0, 0, -1)
	}
	return n
}

// streakReachedAt returns the first day on which a run of n consecutive active days was reached.
func streakReachedAt(days []string, n int) *time.Time {
	if n <= 0 {
		return nil
	}
	parsed := make([]time.Time, 0, len(days))
	seen := map[string]bool{}
	for _, d := range days {
		if seen[d] {
			continue
		}
		t, err := time.Parse(dayLayout, d)
		if err != nil {
			continue
		}
		seen[d] = true
		parsed = append(parsed, t)
	}
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].Before(parsed[j]) })
	run := 0
	for i, t := range parsed {
		if i > 0 && t.Sub(parsed[i-1]) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run >= n {
			r := t
			return &r
		}
	}
	return nil
}

// computeSkills returns, per category (in task order), completed points / total points as 0..100.
func computeSkills(tasks []models.Task, completed map[string]bool) []models.SkillProgress {
	type acc struct{ done, total int }
	order := []string{}
	sums := map[string]*acc{}
	for _, t := range tasks {
		a, ok := sums[t.Category]
		if !ok {
			a = &acc{}
			sums[t.Category] = a
			order = append(order, t.Category)
		}
		a.total += t.Points
		if completed[t.ID] {
			a.done += t.Points
		}
	}
	out := make([]models.SkillProgress, 0, len(order))
	for _, cat := range order {
		a := sums[cat]
		out = append(out, models.SkillProgress{Category: cat, Percent: percent(a.done, a.total)})
	}
	return out
}

// percent returns round(100*part/whole), 0 when whole is 0, clamped to 0..100.
func percent(part, whole int) int {
	if whole <= 0 || part <= 0 {
		return 0
	}
	p := int(math.Round(100 * float64(part) / float64(whole)))
	if p > 100 {
		return 100
	}
	return p
}

// attemptRecord is a check attempt reduced to what achievements need.
type attemptRecord struct {
	TaskID         string
	Passed         bool
	ElapsedSeconds int
	CreatedAt      time.Time
}

// achievementInput is everything needed to compute achievements.
type achievementInput struct {
	TotalTasks   int
	Completions  []time.Time     // completed_at of every passed task
	ActivityDays []string        // distinct UTC days with a check or completion
	FirstAttempt []attemptRecord // the first attempt per task
	FirstPass    []attemptRecord // the first passing attempt per task
}

type achievementDef struct {
	ID, Title, Description string
}

var achievementDefs = []achievementDef{
	{"first-blood", "First blood", "Complete your first task"},
	{"half-way", "Half way there", "Complete half of all tasks"},
	{"all-done", "All done", "Complete every task"},
	{"streak-3", "On a roll", "Practice 3 days in a row"},
	{"streak-7", "Week warrior", "Practice 7 days in a row"},
	{"speedrun", "Speedrun", "Pass a task within 5 minutes of starting the lab"},
	{"perfectionist", "Perfectionist", "Pass 3 tasks on the first attempt"},
}

// computeAchievements evaluates every achievement; earned_at is when the condition was first met.
func computeAchievements(in achievementInput) []models.Achievement {
	completions := append([]time.Time(nil), in.Completions...)
	sort.Slice(completions, func(i, j int) bool { return completions[i].Before(completions[j]) })

	nth := func(n int) *time.Time { // time of the n-th completion (1-based)
		if n <= 0 || len(completions) < n {
			return nil
		}
		t := completions[n-1]
		return &t
	}

	earnedAt := map[string]*time.Time{
		"first-blood": nth(1),
		"streak-3":    streakReachedAt(in.ActivityDays, 3),
		"streak-7":    streakReachedAt(in.ActivityDays, 7),
	}
	if in.TotalTasks > 0 {
		earnedAt["half-way"] = nth((in.TotalTasks + 1) / 2)
		earnedAt["all-done"] = nth(in.TotalTasks)
	}

	var speedrun *time.Time
	for _, a := range in.FirstPass {
		if a.Passed && a.ElapsedSeconds <= speedrunSeconds && (speedrun == nil || a.CreatedAt.Before(*speedrun)) {
			t := a.CreatedAt
			speedrun = &t
		}
	}
	earnedAt["speedrun"] = speedrun

	var perfect []time.Time
	for _, a := range in.FirstAttempt {
		if a.Passed {
			perfect = append(perfect, a.CreatedAt)
		}
	}
	if len(perfect) >= perfectionistNeeded {
		sort.Slice(perfect, func(i, j int) bool { return perfect[i].Before(perfect[j]) })
		t := perfect[perfectionistNeeded-1]
		earnedAt["perfectionist"] = &t
	}

	out := make([]models.Achievement, 0, len(achievementDefs))
	for _, d := range achievementDefs {
		at := earnedAt[d.ID]
		out = append(out, models.Achievement{
			ID: d.ID, Title: d.Title, Description: d.Description,
			Earned: at != nil, EarnedAt: at,
		})
	}
	return out
}

// leaderboardRow is a user's aggregate for the selected period.
type leaderboardRow struct {
	UserID        uint
	Username      string
	Score         int
	Completed     int
	LastCompleted *time.Time
}

// rankLeaderboard orders by score desc, then earliest last completion, then user id,
// and assigns dense ranks by score (equal scores share a rank).
func rankLeaderboard(rows []leaderboardRow) []models.LeaderboardEntry {
	sorted := append([]leaderboardRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		switch {
		case a.LastCompleted != nil && b.LastCompleted != nil && !a.LastCompleted.Equal(*b.LastCompleted):
			return a.LastCompleted.Before(*b.LastCompleted)
		case a.LastCompleted != nil && b.LastCompleted == nil:
			return true
		case a.LastCompleted == nil && b.LastCompleted != nil:
			return false
		}
		return a.UserID < b.UserID
	})
	out := make([]models.LeaderboardEntry, 0, len(sorted))
	rank := 0
	for i, r := range sorted {
		if i == 0 || r.Score != sorted[i-1].Score {
			rank++
		}
		out = append(out, models.LeaderboardEntry{
			UserID: r.UserID, Username: r.Username, Score: r.Score, Completed: r.Completed, Rank: rank,
		})
	}
	return out
}

// buildLeaderboard ranks rows, keeps the top limit entries and always computes "me".
func buildLeaderboard(rows []leaderboardRow, limit int, meID uint, meName string) models.LeaderboardResponse {
	ranked := rankLeaderboard(rows)
	resp := models.LeaderboardResponse{Entries: ranked}
	if len(ranked) > limit {
		resp.Entries = ranked[:limit]
	}
	if meID == 0 {
		return resp
	}
	for i := range ranked {
		if ranked[i].UserID == meID {
			me := ranked[i]
			resp.Me = &me
			return resp
		}
	}
	// No points in this period: rank right after everyone who scored.
	rank := 1
	if n := len(ranked); n > 0 {
		rank = ranked[n-1].Rank
		if ranked[n-1].Score > 0 {
			rank++
		}
	}
	resp.Me = &models.LeaderboardEntry{UserID: meID, Username: meName, Rank: rank}
	return resp
}

// periodStart returns the lower bound for completed_at, or false for an unknown period.
func periodStart(period string, now time.Time) (time.Time, bool) {
	switch period {
	case "", "all":
		return time.Unix(0, 0).UTC(), true
	case "week":
		return now.AddDate(0, 0, -7), true
	case "month":
		return now.AddDate(0, -1, 0), true
	}
	return time.Time{}, false
}

// hourCount is the number of sessions created in one hour (hour = unix seconds / 3600).
type hourCount struct {
	Hour  int64
	Count int64
}

// bucketSessions maps hourly counts to 24 buckets; index 23 is the current hour.
func bucketSessions(counts []hourCount, now time.Time) []int64 {
	out := make([]int64, 24)
	cur := now.Unix() / 3600
	for _, hc := range counts {
		diff := cur - hc.Hour
		if diff >= 0 && diff < 24 {
			out[23-diff] += hc.Count
		}
	}
	return out
}
