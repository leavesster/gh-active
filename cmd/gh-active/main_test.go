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
