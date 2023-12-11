package gocron

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRange(t *testing.T) {
	zero := uint64(0)
	ranges := []struct {
		expr     string
		min, max uint
		expected uint64
		err      string
	}{
		{"5", 0, 7, 1 << 5, ""},
		{"0", 0, 7, 1 << 0, ""},
		{"7", 0, 7, 1 << 7, ""},

		{"5-5", 0, 7, 1 << 5, ""},
		{"5-6", 0, 7, 1<<5 | 1<<6, ""},
		{"5-7", 0, 7, 1<<5 | 1<<6 | 1<<7, ""},

		{"5-6/2", 0, 7, 1 << 5, ""},
		{"5-7/2", 0, 7, 1<<5 | 1<<7, ""},
		{"5-7/1", 0, 7, 1<<5 | 1<<6 | 1<<7, ""},

		{"*", 1, 3, 1<<1 | 1<<2 | 1<<3 | starBit, ""},
		{"*/2", 1, 3, 1<<1 | 1<<3, ""},

		{"5--5", 0, 0, zero, "too many hyphens"},
		{"jan-x", 0, 0, zero, "failed to parse int from"},
		{"2-x", 1, 5, zero, "failed to parse int from"},
		{"*/-12", 0, 0, zero, "negative number"},
		{"*//2", 0, 0, zero, "too many slashes"},
		{"1", 3, 5, zero, "below minimum"},
		{"6", 3, 5, zero, "above maximum"},
		{"5-3", 3, 5, zero, "beyond end of range"},
		{"*/0", 0, 0, zero, "should be a positive number"},
	}

	for _, c := range ranges {
		actual, err := getRange(c.expr, bounds{c.min, c.max, nil})
		if len(c.err) != 0 && (err == nil || !strings.Contains(err.Error(), c.err)) {
			t.Errorf("%s => expected %v, got %v", c.expr, c.err, err)
		}
		if len(c.err) == 0 && err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if actual != c.expected {
			t.Errorf("%s => expected %d, got %d", c.expr, c.expected, actual)
		}
	}
}

func TestField(t *testing.T) {
	fields := []struct {
		expr     string
		min, max uint
		expected uint64
	}{
		{"5", 1, 7, 1 << 5},
		{"5,6", 1, 7, 1<<5 | 1<<6},
		{"5,6,7", 1, 7, 1<<5 | 1<<6 | 1<<7},
		{"1,5-7/2,3", 1, 7, 1<<1 | 1<<5 | 1<<7 | 1<<3},
	}

	for _, c := range fields {
		actual, _ := getField(c.expr, bounds{c.min, c.max, nil})
		if actual != c.expected {
			t.Errorf("%s => expected %d, got %d", c.expr, c.expected, actual)
		}
	}
}

func TestAll(t *testing.T) {
	all := []struct {
		r        bounds
		expected uint64
	}{
		{minutes, 0xfffffffffffffff}, // 0-59: 60 ones
		{hours, 0xffffff},            // 0-23: 24 ones
		{doms, 0xfffffffe},           // 1-31: 31 ones, 1 zero
		{months, 0x1ffe},             // 1-12: 12 ones, 1 zero
		{dows, 0x7f},                 // 0-6: 7 ones
	}

	for _, c := range all {
		actual := allBits(c.r) // allBits() adds the starBit, so compensate for that..
		if c.expected|starBit != actual {
			t.Errorf("%d-%d/%d => expected %b, got %b",
				c.r.min, c.r.max, 1, c.expected|starBit, actual)
		}
	}
}

func TestBits(t *testing.T) {
	bits := []struct {
		min, max, step uint
		expected       uint64
	}{
		{0, 0, 1, 0x1},
		{1, 1, 1, 0x2},
		{1, 5, 2, 0x2a}, // 101010
		{1, 4, 2, 0xa},  // 1010
	}

	for _, c := range bits {
		actual := getBits(c.min, c.max, c.step)
		if c.expected != actual {
			t.Errorf("%d-%d/%d => expected %b, got %b",
				c.min, c.max, c.step, c.expected, actual)
		}
	}
}

func TestParseScheduleErrors(t *testing.T) {
	var tests = []struct{ expr, err string }{
		{"* 5 j * * *", "failed to parse int from"},
		{"@every Xm", "failed to parse duration"},
		{"@unrecognized", "unrecognized descriptor"},
		{"* * * *", "expected exactly 6 fields, found 4: [* * * *]"},
		{"", "empty spec string"},
	}
	for _, c := range tests {
		actual, err := Parse(c.expr)
		if err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s => expected %v, got %v", c.expr, c.err, err)
		}
		if actual != nil {
			t.Errorf("expected nil schedule on error, got %v", actual)
		}
	}
}

func TestParseSchedule(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	utc, _ := time.LoadLocation("UTC")
	entries := []struct {
		expr     string
		loc      *time.Location
		expected Schedule
	}{
		{"0 5 * * * *", time.Local, every5min(time.Local)},
		{"0 5 * * * *", utc, every5min(time.UTC)},
		{"0 5 * * * *", tokyo, every5min(tokyo)},
		{"@every 5m", time.Local, everySchedule{5 * time.Minute}},
		{"@midnight", time.Local, midnight(time.Local)},
		{"@midnight", utc, midnight(time.UTC)},
		{"@midnight", tokyo, midnight(tokyo)},
		{"@yearly", time.Local, annual(time.Local)},
		{"@annually", time.Local, annual(time.Local)},
		{
			expr: "* 5 * * * *",
			loc:  time.Local,
			expected: &specSchedule{
				second:   allBits(seconds),
				minute:   1 << 5,
				hour:     allBits(hours),
				dom:      allBits(doms),
				month:    allBits(months),
				dow:      allBits(dows),
				location: time.Local,
			},
		},
	}

	for _, c := range entries {
		actual, err := ParseWithLocation(c.expr, c.loc)
		if err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => expected %b, got %b", c.expr, c.expected, actual)
		}
	}
}

func TestStandardSpecSchedule(t *testing.T) {
	entries := []struct {
		expr     string
		expected Schedule
		err      string
	}{
		{
			expr:     "0 5 * * * *",
			expected: &specSchedule{1 << seconds.min, 1 << 5, allBits(hours), allBits(doms), allBits(months), allBits(dows), time.Local},
		},
		{
			expr:     "@every 5m",
			expected: everySchedule{time.Duration(5) * time.Minute},
		},
		{
			expr: "* 5 j * * *",
			err:  "failed to parse int from",
		},
		{
			expr: "* * * *",
			err:  "expected exactly 6 fields, found 4: [* * * *]",
		},
	}

	for _, c := range entries {
		actual, err := Parse(c.expr)
		if len(c.err) != 0 && (err == nil || !strings.Contains(err.Error(), c.err)) {
			t.Errorf("%s => expected %v, got %v", c.expr, c.err, err)
		}
		if len(c.err) == 0 && err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => expected %b, got %b", c.expr, c.expected, actual)
		}
	}
}

func every5min(loc *time.Location) *specSchedule {
	return &specSchedule{1 << 0, 1 << 5, allBits(hours), allBits(doms), allBits(months), allBits(dows), loc}
}

func midnight(loc *time.Location) *specSchedule {
	return &specSchedule{1, 1, 1, allBits(doms), allBits(months), allBits(dows), loc}
}

func annual(loc *time.Location) *specSchedule {
	return &specSchedule{
		second:   1 << seconds.min,
		minute:   1 << minutes.min,
		hour:     1 << hours.min,
		dom:      1 << doms.min,
		month:    1 << months.min,
		dow:      allBits(dows),
		location: loc,
	}
}
