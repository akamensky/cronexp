[![GoDoc](http://godoc.org/github.com/akamensky/cronexp?status.png)](http://godoc.org/github.com/akamensky/cronexp)

# cronexp

`cronexp` is a modified version of original [cron library](https://github.com/robfig/cron) from commit 
[bc59245](https://github.com/robfig/cron/commit/bc59245fe10efaed9d51b56900192527ed733435). See [LICENSES](LICENSES) for
the original license notice.

The original code has been modified with following:
- No job scheduling - this is a parser only with ability to get next execution time
- Seconds and DOW are mandatory fields - thus expanding scheduling granularity to second by default
- No separate parser type - it is replaced by `Parse` and `ParseWithLocation` exported methods

See `godoc` for more details on usage.

### Cron spec format

There are two cron spec formats in common usage:

- The "standard" cron format, described on [the Cron wikipedia page] and used by
  the cron Linux system utility.

- The cron format used by [the Quartz Scheduler], commonly used for scheduled
  jobs in Java software

[the Cron wikipedia page]: https://en.wikipedia.org/wiki/Cron
[the Quartz Scheduler]: http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/tutorial-lesson-06.html

The original library provides seconds and day-of-week as optional fields. This implementation is intended to be strict,
by making seconds and day-of-week required.
