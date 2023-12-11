package gocron

import "time"

// everySchedule represents a simple recurring duty cycle, e.g. "Every 5 minutes".
// It does not support jobs more frequent than once a second.
type everySchedule struct {
	delay time.Duration
}

// every returns a crontab Schedule that activates once every duration.
// Delays of less than a second are not supported (will round up to 1 second).
// Any fields less than a second are truncated.
func every(duration time.Duration) everySchedule {
	if duration < time.Second {
		duration = time.Second
	}
	return everySchedule{
		delay: duration - time.Duration(duration.Nanoseconds())%time.Second,
	}
}

// Next returns the next time this should be run.
// This rounds so that the next activation time will be on the second.
func (s everySchedule) Next(t time.Time) time.Time {
	return t.Add(s.delay - time.Duration(t.Nanosecond())*time.Nanosecond)
}
