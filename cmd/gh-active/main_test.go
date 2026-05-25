package main

import (
	"testing"
	"time"

	"github.com/leavesster/gh-active/internal/output"
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
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	start, end, err := parseTimeRangeAt(now, "", "", "2026-02-19")
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
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	start, end, err := parseTimeRangeAt(now, "", "", "previous")
	if err != nil {
		t.Fatal(err)
	}

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
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	start, end, err := parseTimeRangeAt(now, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	wantMonday := weekMonday(now)

	if got := start.Format("2006-01-02"); got != wantMonday.Format("2006-01-02") {
		t.Errorf("start = %s, want %s (this Monday)", got, wantMonday.Format("2006-01-02"))
	}
	if !end.Equal(now) {
		t.Errorf("end = %v, want %v", end, now)
	}
}

func TestParseTimeRange_StartOnly(t *testing.T) {
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	start, end, err := parseTimeRangeAt(now, "2026-05-18", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := start.Format("2006-01-02 15:04:05"); got != "2026-05-18 00:00:00" {
		t.Errorf("start = %s, want 2026-05-18 00:00:00", got)
	}
	if !end.Equal(now) {
		t.Errorf("end = %v, want %v", end, now)
	}
}

func TestParseTimeRange_EndOnly(t *testing.T) {
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	start, end, err := parseTimeRangeAt(now, "", "2026-05-25", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := start.Format("2006-01-02 15:04:05"); got != "2026-05-25 00:00:00" {
		t.Errorf("start = %s, want 2026-05-25 00:00:00", got)
	}
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-05-25 23:59:59" {
		t.Errorf("end = %s, want 2026-05-25 23:59:59", got)
	}
}

func TestParseTimeRange_RejectsEndBeforeStart(t *testing.T) {
	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	_, _, err := parseTimeRangeAt(now, "2026-05-25", "2026-05-24", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOutputWriterPrefersFlagPath(t *testing.T) {
	writer, err := outputWriter("flag.md", "config.md")
	if err != nil {
		t.Fatal(err)
	}

	if got := writer.Target(); got != "flag.md" {
		t.Fatalf("target = %q, want %q", got, "flag.md")
	}
}

func TestOutputWriterUsesConfigPath(t *testing.T) {
	writer, err := outputWriter("", "config.md")
	if err != nil {
		t.Fatal(err)
	}

	fileWriter, ok := writer.(*output.FileWriter)
	if !ok {
		t.Fatalf("writer type = %T, want *output.FileWriter", writer)
	}
	if got := fileWriter.Target(); got != "config.md" {
		t.Fatalf("target = %q, want %q", got, "config.md")
	}
}
