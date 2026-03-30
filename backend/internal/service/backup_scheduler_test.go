package service

import (
	"testing"
	"time"
)

func TestValidateCron_Valid(t *testing.T) {
	cases := []string{
		"0 3 * * *",      // daily at 3am
		"*/15 * * * *",   // every 15 minutes
		"0 0 1 * *",      // monthly on 1st
		"30 2 * * 1",     // weekly Monday 2:30am
		"0 0 * * 0",      // weekly Sunday midnight
		"0,30 * * * *",   // every 30 minutes
		"0 3-6 * * *",    // hourly 3am-6am
		"0 3 1,15 * *",   // 1st and 15th of month
	}
	for _, c := range cases {
		if err := validateCron(c); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", c, err)
		}
	}
}

func TestValidateCron_Invalid(t *testing.T) {
	cases := []struct {
		expr string
		desc string
	}{
		{"", "empty"},
		{"0 3 * *", "4 fields"},
		{"0 3 * * * *", "6 fields"},
		{"0 3 * * MON", "letters"},
		{"hello world", "words"},
	}
	for _, c := range cases {
		if err := validateCron(c.expr); err == nil {
			t.Errorf("expected %q (%s) to be invalid", c.expr, c.desc)
		}
	}
}

func TestCronMatches_DailyAt3AM(t *testing.T) {
	// "0 3 * * *" should match 3:00 AM on any day
	t3am := time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)
	if !cronMatches("0 3 * * *", t3am) {
		t.Error("expected 3:00 AM to match '0 3 * * *'")
	}

	// Should NOT match at 3:01 AM
	t3_01 := time.Date(2025, 6, 15, 3, 1, 0, 0, time.UTC)
	if cronMatches("0 3 * * *", t3_01) {
		t.Error("3:01 AM should not match '0 3 * * *'")
	}

	// Should NOT match at 4:00 AM
	t4am := time.Date(2025, 6, 15, 4, 0, 0, 0, time.UTC)
	if cronMatches("0 3 * * *", t4am) {
		t.Error("4:00 AM should not match '0 3 * * *'")
	}
}

func TestCronMatches_Every15Min(t *testing.T) {
	base := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	for _, min := range []int{0, 15, 30, 45} {
		tm := base.Add(time.Duration(min) * time.Minute)
		if !cronMatches("*/15 * * * *", tm) {
			t.Errorf("expected minute %d to match '*/15 * * * *'", min)
		}
	}

	for _, min := range []int{1, 7, 14, 16, 31, 44} {
		tm := base.Add(time.Duration(min) * time.Minute)
		if cronMatches("*/15 * * * *", tm) {
			t.Errorf("minute %d should not match '*/15 * * * *'", min)
		}
	}
}

func TestCronMatches_MondaysAt230(t *testing.T) {
	// "30 2 * * 1" — Monday at 2:30am
	monday := time.Date(2025, 6, 16, 2, 30, 0, 0, time.UTC) // June 16 2025 is a Monday
	if !cronMatches("30 2 * * 1", monday) {
		t.Error("expected Monday 2:30 to match '30 2 * * 1'")
	}

	// Same time on Tuesday should not match
	tuesday := time.Date(2025, 6, 17, 2, 30, 0, 0, time.UTC)
	if cronMatches("30 2 * * 1", tuesday) {
		t.Error("Tuesday should not match '30 2 * * 1'")
	}
}

func TestCronMatches_MonthlyFirst(t *testing.T) {
	// "0 0 1 * *" — midnight on the 1st
	first := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if !cronMatches("0 0 1 * *", first) {
		t.Error("expected 1st at midnight to match '0 0 1 * *'")
	}

	second := time.Date(2025, 7, 2, 0, 0, 0, 0, time.UTC)
	if cronMatches("0 0 1 * *", second) {
		t.Error("2nd should not match '0 0 1 * *'")
	}
}

func TestCronMatches_CommaList(t *testing.T) {
	// "0 3 1,15 * *" — 3am on 1st and 15th
	d1 := time.Date(2025, 6, 1, 3, 0, 0, 0, time.UTC)
	d15 := time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)
	d10 := time.Date(2025, 6, 10, 3, 0, 0, 0, time.UTC)

	if !cronMatches("0 3 1,15 * *", d1) {
		t.Error("1st at 3am should match")
	}
	if !cronMatches("0 3 1,15 * *", d15) {
		t.Error("15th at 3am should match")
	}
	if cronMatches("0 3 1,15 * *", d10) {
		t.Error("10th should not match")
	}
}

func TestCronMatches_Range(t *testing.T) {
	// "0 3-6 * * *" — 3am, 4am, 5am, 6am
	for _, hr := range []int{3, 4, 5, 6} {
		tm := time.Date(2025, 6, 15, hr, 0, 0, 0, time.UTC)
		if !cronMatches("0 3-6 * * *", tm) {
			t.Errorf("hour %d should match '0 3-6 * * *'", hr)
		}
	}

	tm2am := time.Date(2025, 6, 15, 2, 0, 0, 0, time.UTC)
	if cronMatches("0 3-6 * * *", tm2am) {
		t.Error("2am should not match '0 3-6 * * *'")
	}

	tm7am := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	if cronMatches("0 3-6 * * *", tm7am) {
		t.Error("7am should not match '0 3-6 * * *'")
	}
}

func TestCronMatches_InvalidExpr(t *testing.T) {
	tm := time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)
	if cronMatches("invalid", tm) {
		t.Error("invalid cron should not match")
	}
	if cronMatches("", tm) {
		t.Error("empty cron should not match")
	}
}

func TestFieldMatches_Wildcard(t *testing.T) {
	for _, v := range []int{0, 1, 15, 59} {
		if !fieldMatches("*", v, 0, 59) {
			t.Errorf("wildcard should match %d", v)
		}
	}
}

func TestFieldMatches_WildcardStep(t *testing.T) {
	// */10 should match 0, 10, 20, 30, 40, 50
	for _, v := range []int{0, 10, 20, 30, 40, 50} {
		if !fieldMatches("*/10", v, 0, 59) {
			t.Errorf("*/10 should match %d", v)
		}
	}
	for _, v := range []int{1, 5, 11, 25, 59} {
		if fieldMatches("*/10", v, 0, 59) {
			t.Errorf("*/10 should not match %d", v)
		}
	}
}

func TestFormatNextRun(t *testing.T) {
	after := time.Date(2025, 6, 15, 2, 59, 0, 0, time.UTC)
	next, err := FormatNextRun("0 3 * * *", after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}
