// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	defaultLoc = func() *time.Location {
		if l, err := time.LoadLocation("Europe/Zurich"); err == nil {
			return l
		}
		return time.UTC
	}()
	isoRe = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	chRe  = regexp.MustCompile(`^(\d{1,2})[./\-](\d{1,2})(?:[./\-](\d{2}|\d{4}))?\.?$`)
)

const isoFmt = "2006-01-02"

// tz is the time zone used for "today" and the times shown (setting TimeZone).
var tz atomic.Pointer[time.Location]

func loc() *time.Location {
	if l := tz.Load(); l != nil {
		return l
	}
	return defaultLoc
}

func today() string { return time.Now().In(loc()).Format(isoFmt) }

func parseISO(d string) (time.Time, bool) {
	t, err := time.Parse(isoFmt, d)
	return t, err == nil
}

func addDays(date string, n int) string {
	t, ok := parseISO(date)
	if !ok {
		return date
	}
	return t.AddDate(0, 0, n).Format(isoFmt)
}

// parseDate accepts 15.12.2026, 15.12.26, 15.12, 15/12/2026 and 2026-12-15.
// Without a year: the current year, or next year if that date is more than 2 months ago.
func parseDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	var y, m, d int
	if g := isoRe.FindStringSubmatch(s); g != nil {
		y, _ = strconv.Atoi(g[1])
		m, _ = strconv.Atoi(g[2])
		d, _ = strconv.Atoi(g[3])
	} else if g := chRe.FindStringSubmatch(s); g != nil {
		d, _ = strconv.Atoi(g[1])
		m, _ = strconv.Atoi(g[2])
		now := time.Now().In(loc())
		switch len(g[3]) {
		case 0:
			y = now.Year()
			cand := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
			if cand.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)) {
				y++
			}
		case 2:
			y, _ = strconv.Atoi(g[3])
			y += 2000
		default:
			y, _ = strconv.Atoi(g[3])
		}
	} else {
		return "", false
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != y || int(t.Month()) != m || t.Day() != d || y < 2000 || y > 2100 {
		return "", false
	}
	return t.Format(isoFmt), true
}

// countDays counts the leave days in [start, end]; a half day counts 0.5.
func countDays(start, end, part string, weekends bool) float64 {
	s, ok1 := parseISO(start)
	e, ok2 := parseISO(end)
	if !ok1 || !ok2 || e.Before(s) {
		return 0
	}
	n := 0.0
	for t := s; !t.After(e); t = t.AddDate(0, 0, 1) {
		if weekends || (t.Weekday() != time.Saturday && t.Weekday() != time.Sunday) {
			n++
		}
	}
	if start == end && part != PartFull && n > 0 {
		return 0.5
	}
	return n
}

// fmtDate formats 2026-12-15 in the user's language ("15.12.2026", "Dec 15, 2026"...).
func fmtDate(lang, d string) string {
	t, ok := parseISO(d)
	if !ok {
		return d
	}
	return t.Format(dateLayout[lang])
}

func shortDay(lang, d string) string {
	t, ok := parseISO(d)
	if !ok {
		return d
	}
	return daysShort[lang][t.Weekday()] + " " + t.Format(dateLayout[lang])
}

func partLabel(lang, part string) string {
	switch part {
	case PartAM:
		return tr(lang, "lc.am")
	case PartPM:
		return tr(lang, "lc.pm")
	}
	return ""
}

// rangeLabel: "Monday Dec 15, 2026 (morning)" or "from Mon Dec 14, 2026 to Fri Dec 18, 2026".
func rangeLabel(lang, start, end, part string) string {
	if start == end {
		t, ok := parseISO(start)
		label := start
		if ok {
			label = daysLong[lang][t.Weekday()] + " " + t.Format(dateLayout[lang])
		}
		if pl := partLabel(lang, part); pl != "" {
			label += " (" + pl + ")"
		}
		return label
	}
	return tr(lang, "range", shortDay(lang, start), shortDay(lang, end))
}

func daysLabel(lang string, n float64) string {
	if n <= 1 {
		return tr(lang, "day1", fmtNum(lang, n))
	}
	return tr(lang, "dayN", fmtNum(lang, n))
}

func fmtNum(lang string, n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if lang != "en" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

func millisDate(lang string, ms int64) string {
	t := time.UnixMilli(ms).In(loc())
	return tr(lang, "at", t.Format(dateLayout[lang]), t.Format("15:04"))
}
