// Package schedule provides the scheduled-jobs backend behind the
// internal/application/schedule contract. Business modules only register jobs;
// the concrete scheduling loop (stdlib time + a 5-field cron parser) is hidden
// behind Worker.
package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parse accepts a 5-field cron expression:
//
//	┌────────────── minute (0-59)
//	│ ┌──────────── hour (0-23)
//	│ │ ┌────────── day of month (1-31)
//	│ │ │ ┌──────── month (1-12, or JAN..DEC)
//	│ │ │ │ ┌────── day of week (0-7, or SUN..SAT; 0 and 7 are Sunday)
//	│ │ │ │ │
//	* * * * *
//
// Each field supports `*`, `?` (alias for `*`), `*/n`, `a-b`, `a-b/n`,
// comma-separated lists, and single values. `a` is optional in `a-b`, so
// `a-b/n` covers `a/n` only when `a-b` is a range; plain `*/n` covers "every
// n". Names are case-insensitive and valid inside ranges and steps too.
func Parse(expr string) (*Spec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron %q: expected 5 fields (minute hour day-of-month month day-of-week), got %d", expr, len(fields))
	}

	s := &Spec{
		minute: fieldSpec{kind: fieldMinute, min: 0, max: 59},
		hour:   fieldSpec{kind: fieldHour, min: 0, max: 23},
		dom:    fieldSpec{kind: fieldDayOfMonth, min: 1, max: 31},
		month:  fieldSpec{kind: fieldMonth, min: 1, max: 12},
		dow:    fieldSpec{kind: fieldDayOfWeek, min: 0, max: 7},
	}

	specs := []*fieldSpec{&s.minute, &s.hour, &s.dom, &s.month, &s.dow}
	names := []string{"minute", "hour", "day-of-month", "month", "day-of-week"}
	for i, tok := range fields {
		if err := specs[i].parse(tok); err != nil {
			return nil, fmt.Errorf("cron %q: field %d (%s): %w", expr, i+1, names[i], err)
		}
	}
	return s, nil
}

// Spec is a compiled 5-field cron expression. Match reports whether t (already
// in the scheduler's location) falls on an allowed minute.
type Spec struct {
	minute fieldSpec
	hour   fieldSpec
	dom    fieldSpec
	month  fieldSpec
	dow    fieldSpec
}

// Match reports whether t satisfies the cron expression.
func (s *Spec) Match(t time.Time) bool {
	dow := int(t.Weekday())
	if dow == 0 && s.dow.has(7) {
		dow = 7
	}
	return s.minute.has(t.Minute()) &&
		s.hour.has(t.Hour()) &&
		s.dom.has(t.Day()) &&
		s.month.has(int(t.Month())) &&
		s.dow.has(dow)
}

type fieldKind int

const (
	fieldMinute fieldKind = iota
	fieldHour
	fieldDayOfMonth
	fieldMonth
	fieldDayOfWeek
)

func (k fieldKind) String() string {
	return [...]string{"minute", "hour", "day-of-month", "month", "day-of-week"}[k]
}

// fieldSpec is a single cron field compiled to a bitmask over its value range.
// Bit v is set when value v is allowed.
type fieldSpec struct {
	kind fieldKind
	min  int
	max  int
	mask uint64
}

func (f *fieldSpec) has(v int) bool {
	return f.mask&(uint64(1)<<v) != 0
}

func (f *fieldSpec) set(v int) {
	f.mask |= uint64(1) << v
}

func (f *fieldSpec) setAll() {
	for v := f.min; v <= f.max; v++ {
		f.set(v)
	}
}

// parse compiles one comma-separated token into the bitmask.
func (f *fieldSpec) parse(token string) error {
	if token == "?" {
		token = "*"
	}
	for _, part := range strings.Split(token, ",") {
		if err := f.parsePart(part); err != nil {
			return err
		}
	}
	return nil
}

// parsePart compiles one list item, optionally with a `/step` suffix.
func (f *fieldSpec) parsePart(part string) error {
	step := 1
	if i := strings.IndexByte(part, '/'); i >= 0 {
		s, err := strconv.Atoi(part[i+1:])
		if err != nil || s <= 0 {
			return fmt.Errorf("invalid step %q", part[i+1:])
		}
		step = s
		part = part[:i]
	}

	var lo, hi int
	switch {
	case part == "*":
		lo, hi = f.min, f.max
	case strings.Contains(part, "-"):
		parts := strings.SplitN(part, "-", 2)
		var err error
		if lo, err = f.resolve(parts[0]); err != nil {
			return err
		}
		if hi, err = f.resolve(parts[1]); err != nil {
			return err
		}
	default:
		v, err := f.resolve(part)
		if err != nil {
			return err
		}
		lo, hi = v, v
	}

	if lo < f.min || hi > f.max || lo > hi {
		return fmt.Errorf("value %d-%d out of range [%d-%d]", lo, hi, f.min, f.max)
	}
	for v := lo; v <= hi; v += step {
		f.set(v)
	}
	return nil
}

// resolve maps a single value to a number, accepting numeric values and the
// month/day-of-week name aliases.
func (f *fieldSpec) resolve(token string) (int, error) {
	if n, err := strconv.Atoi(token); err == nil {
		return n, nil
	}

	name := strings.ToUpper(token)
	switch f.kind {
	case fieldMonth:
		if n, ok := monthNames[name]; ok {
			return n, nil
		}
	case fieldDayOfWeek:
		if n, ok := dowNames[name]; ok {
			return n, nil
		}
	}
	return 0, fmt.Errorf("invalid value %q for %s", token, f.kind)
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var dowNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}
