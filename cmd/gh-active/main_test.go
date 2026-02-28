package main

import (
	"testing"
	"time"
)

func TestWeekMonday(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-02-16", "2026-02-16"}, // Monday → itself
		{"2026-02-17", "2026-02-16"}, // Tuesday
		{"2026-02-18", "2026-02-16"}, // Wednesday
		{"2026-02-19", "2026-02-16"}, // Thursday
		{"2026-02-20", "2026-02-16"}, // Friday
		{"2026-02-21", "2026-02-16"}, // Saturday
		{"2026-02-22", "2026-02-16"}, // Sunday → same week's Monday
	}

	for _, tt := range tests {
		d, _ := time.Parse("2006-01-02", tt.input)
		got := weekMonday(d).Format("2006-01-02")
		if got != tt.want {
			t.Errorf("weekMonday(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestParseTimeRange_Week(t *testing.T) {
	start, end, err := parseTimeRange("", "", "2026-02-19")
	if err != nil {
		t.Fatal(err)
	}
	if got := start.Format("2006-01-02"); got != "2026-02-16" {
		t.Errorf("start = %s, want 2026-02-16", got)
	}
	if got := end.Format("2006-01-02"); got != "2026-02-22" {
		t.Errorf("end date = %s, want 2026-02-22", got)
	}
}

func TestParseTimeRange_Previous(t *testing.T) {
	start, end, err := parseTimeRange("", "", "previous")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	wantStart := weekMonday(now).AddDate(0, 0, -7)
	wantEnd := wantStart.AddDate(0, 0, 7).Add(-time.Second)

	if got := start.Format("2006-01-02"); got != wantStart.Format("2006-01-02") {
		t.Errorf("start = %s, want %s", got, wantStart.Format("2006-01-02"))
	}
	if got := end.Format("2006-01-02"); got != wantEnd.Format("2006-01-02") {
		t.Errorf("end = %s, want %s", got, wantEnd.Format("2006-01-02"))
	}
}

func TestParseTimeRange_Default_CurrentWeek(t *testing.T) {
	start, end, err := parseTimeRange("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	wantMonday := weekMonday(now)

	if got := start.Format("2006-01-02"); got != wantMonday.Format("2006-01-02") {
		t.Errorf("start = %s, want %s (this Monday)", got, wantMonday.Format("2006-01-02"))
	}
	// end should be close to now (within a few seconds)
	diff := now.Sub(end)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("end = %v, want close to now (%v), diff = %v", end, now, diff)
	}
}
