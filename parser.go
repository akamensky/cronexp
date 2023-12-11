package gocron

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func Parse(spec string) (Schedule, error) {
	return ParseWithLocation(spec, time.Local)
}

func ParseWithLocation(spec string, loc *time.Location) (Schedule, error) {
	// Check if spec is an empty string
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

	// Handle descriptors if present
	if strings.HasPrefix(spec, "@") {
		return parseDescriptor(spec, loc)
	}

	// Split on whitespace.
	fields := strings.Fields(spec)

	// Validate number of fields
	if count := len(fields); count != 6 {
		return nil, fmt.Errorf("expected exactly %d fields, found %d: %s", 6, count, fields)
	}

	var err error
	field := func(field string, r bounds) uint64 {
		if err != nil {
			return 0
		}
		var bits uint64
		bits, err = getField(field, r)
		return bits
	}

	var (
		second = field(fields[0], seconds)
		minute = field(fields[1], minutes)
		hour   = field(fields[2], hours)
		dom    = field(fields[3], doms)
		month  = field(fields[4], months)
		dow    = field(fields[5], dows)
	)
	if err != nil {
		return nil, err
	}

	return &specSchedule{
		second:   second,
		minute:   minute,
		hour:     hour,
		dom:      dom,
		month:    month,
		dow:      dow,
		location: loc,
	}, nil
}

// getField returns an Int with the bits set to represent all of the times that
// the field represents or error parsing field value.  A "field" is a comma-separated
// list of "ranges".
func getField(field string, r bounds) (uint64, error) {
	var bits uint64
	ranges := strings.FieldsFunc(field, func(r rune) bool { return r == ',' })
	for _, expr := range ranges {
		bit, err := getRange(expr, r)
		if err != nil {
			return bits, err
		}
		bits |= bit
	}
	return bits, nil
}

// getRange returns the bits indicated by the given expression:
//
//	number | number "-" number [ "/" number ]
//
// or error parsing range.
func getRange(expr string, r bounds) (uint64, error) {
	var (
		start, end, step uint
		rangeAndStep     = strings.Split(expr, "/")
		lowAndHigh       = strings.Split(rangeAndStep[0], "-")
		singleDigit      = len(lowAndHigh) == 1
		err              error
	)

	var extra uint64
	if lowAndHigh[0] == "*" || lowAndHigh[0] == "?" {
		start = r.min
		end = r.max
		extra = starBit
	} else {
		start, err = parseIntOrName(lowAndHigh[0], r.names)
		if err != nil {
			return 0, err
		}
		switch len(lowAndHigh) {
		case 1:
			end = start
		case 2:
			end, err = parseIntOrName(lowAndHigh[1], r.names)
			if err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("too many hyphens: %s", expr)
		}
	}

	switch len(rangeAndStep) {
	case 1:
		step = 1
	case 2:
		step, err = mustParseInt(rangeAndStep[1])
		if err != nil {
			return 0, err
		}

		// Special handling: "N/step" means "N-max/step".
		if singleDigit {
			end = r.max
		}
		if step > 1 {
			extra = 0
		}
	default:
		return 0, fmt.Errorf("too many slashes: %s", expr)
	}

	if start < r.min {
		return 0, fmt.Errorf("beginning of range (%d) below minimum (%d): %s", start, r.min, expr)
	}
	if end > r.max {
		return 0, fmt.Errorf("end of range (%d) above maximum (%d): %s", end, r.max, expr)
	}
	if start > end {
		return 0, fmt.Errorf("beginning of range (%d) beyond end of range (%d): %s", start, end, expr)
	}
	if step == 0 {
		return 0, fmt.Errorf("step of range should be a positive number: %s", expr)
	}

	return getBits(start, end, step) | extra, nil
}

// parseIntOrName returns the (possibly-named) integer contained in expr.
func parseIntOrName(expr string, names map[string]uint) (uint, error) {
	if names != nil {
		if namedInt, ok := names[strings.ToLower(expr)]; ok {
			return namedInt, nil
		}
	}
	return mustParseInt(expr)
}

// mustParseInt parses the given expression as an int or returns an error.
func mustParseInt(expr string) (uint, error) {
	num, err := strconv.Atoi(expr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %s", expr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("negative number (%d) not allowed: %s", num, expr)
	}

	return uint(num), nil
}

// getBits sets all bits in the range [min, max], modulo the given step size.
func getBits(min, max, step uint) uint64 {
	var bits uint64

	// If step is 1, use shifts.
	if step == 1 {
		return ^(math.MaxUint64 << (max + 1)) & (math.MaxUint64 << min)
	}

	// Else, use a simple loop.
	for i := min; i <= max; i += step {
		bits |= 1 << i
	}
	return bits
}

// all returns all bits within the given bounds.  (plus the star bit)
func allBits(r bounds) uint64 {
	return getBits(r.min, r.max, 1) | starBit
}

// parseDescriptor returns a predefined schedule for the expression, or error if none matches.
func parseDescriptor(descriptor string, loc *time.Location) (Schedule, error) {
	switch descriptor {
	case "@yearly", "@annually":
		return &specSchedule{
			second:   1 << seconds.min,
			minute:   1 << minutes.min,
			hour:     1 << hours.min,
			dom:      1 << doms.min,
			month:    1 << months.min,
			dow:      allBits(dows),
			location: loc,
		}, nil

	case "@monthly":
		return &specSchedule{
			second:   1 << seconds.min,
			minute:   1 << minutes.min,
			hour:     1 << hours.min,
			dom:      1 << doms.min,
			month:    allBits(months),
			dow:      allBits(dows),
			location: loc,
		}, nil

	case "@weekly":
		return &specSchedule{
			second:   1 << seconds.min,
			minute:   1 << minutes.min,
			hour:     1 << hours.min,
			dom:      allBits(doms),
			month:    allBits(months),
			dow:      1 << dows.min,
			location: loc,
		}, nil

	case "@daily", "@midnight":
		return &specSchedule{
			second:   1 << seconds.min,
			minute:   1 << minutes.min,
			hour:     1 << hours.min,
			dom:      allBits(doms),
			month:    allBits(months),
			dow:      allBits(dows),
			location: loc,
		}, nil

	case "@hourly":
		return &specSchedule{
			second:   1 << seconds.min,
			minute:   1 << minutes.min,
			hour:     allBits(hours),
			dom:      allBits(doms),
			month:    allBits(months),
			dow:      allBits(dows),
			location: loc,
		}, nil

	}

	const everyDescriptor = "@every "
	if strings.HasPrefix(descriptor, everyDescriptor) {
		duration, err := time.ParseDuration(descriptor[len(everyDescriptor):])
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration %s: %s", descriptor, err)
		}
		return every(duration), nil
	}

	return nil, fmt.Errorf("unrecognized descriptor: %s", descriptor)
}
