package main

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func TestRandomizeSchedule_Same(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// invalid and special cases
	tests := []string{
		"*/5 * * * *",
		"invalid",
		"0 abc * * *",
		"100 10 * * *",
		"0 0 *",
		"0 0 * * * *",
	}

	for _, tc := range tests {
		v := randomizeSchedule(tc, rng)
		if v != tc {
			t.Error(tc, v)
		}
	}
}

func TestRandomizeSchedule_Change(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := []string{
		"30 2 * * *",
		"0 * * * *",
		"15 10 * * *",
		"0 0 1 * *",
		"0 0 1 6 *",
		"0 0 * * 1",
		"0 9-17 * * *",
		"0 0 1,15 * *",
		"@daily",
		"@weekly",
		"@monthly",
		"@hourly",
		"@annually",
		"@yearly",
	}

	for _, tc := range tests {
		v := randomizeSchedule(tc, rng)
		if v == tc {
			t.Error(v, tc)
		}

		// Verify result is valid cron format
		fields := strings.Fields(v)
		if len(fields) != 5 {
			t.Error(fields, tc)
		}
	}
}

func TestRandomizeScheduleSpecialFormats(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	t.Run("@daily becomes a random time with wildcards preserved", func(t *testing.T) {
		v := randomizeSchedule("@daily", rng)
		fields := strings.Fields(v)
		if len(fields) != 5 {
			t.Fatalf("@daily should produce 5-field cron, got %q", v)
		}

		if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
			t.Errorf("@daily should preserve wildcards in day-of-month, month, day-of-week, got %q", v)
		}

		// Minute and hour should be specific numbers
		if _, err := strconv.Atoi(fields[0]); err != nil {
			t.Errorf("@daily minute should be a number, got %q", fields[0])
		}
		if _, err := strconv.Atoi(fields[1]); err != nil {
			t.Errorf("@daily hour should be a number, got %q", fields[1])
		}
	})

	t.Run("@weekly preserves day-of-week", func(t *testing.T) {
		v := randomizeSchedule("@weekly", rng)
		fields := strings.Fields(v)
		if fields[4] != "0" {
			t.Errorf("@weekly should preserve day-of-week=0, got %q", fields[4])
		}
	})

	t.Run("@monthly preserves day-of-month", func(t *testing.T) {
		v := randomizeSchedule("@monthly", rng)
		fields := strings.Fields(v)
		if fields[2] != "1" {
			t.Errorf("@monthly should preserve day-of-month=1, got %q", fields[2])
		}
	})

	t.Run("@hourly keeps hour as wildcard", func(t *testing.T) {
		v := randomizeSchedule("@hourly", rng)
		fields := strings.Fields(v)
		if fields[1] != "*" {
			t.Errorf("@hourly should keep hour as *, got %q", fields[1])
		}
	})
}
