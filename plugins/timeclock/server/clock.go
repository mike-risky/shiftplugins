// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func nowMs() int64 { return model.GetMillis() }

func (p *Plugin) t(ms int64) time.Time { return time.UnixMilli(ms).In(p.tz()) }

func (p *Plugin) clock(ms int64) string  { return p.t(ms).Format("15:04") }
func (p *Plugin) dayKey(ms int64) string { return p.t(ms).Format("2006-01-02") }
func (p *Plugin) monthOf(ms int64) string {
	return p.t(ms).Format("2006-01")
}

// hm formats a duration in ms as h:mm.
func hm(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	m := (ms + 30000) / 60000
	return fmt.Sprintf("%d:%02d", m/60, m%60)
}

func minutes(ms int64) int64 {
	if ms < 0 {
		return 0
	}
	return (ms + 30000) / 60000
}

func shiftMonth(month string, n int) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.AddDate(0, n, 0).Format("2006-01")
}

// staleMs: an open entry older than this is considered a forgotten clock-out.
func (p *Plugin) staleMs() int64 { return int64(p.conf().MaxHours) * 3600_000 }

// markMissing closes a forgotten entry without an end time (counts 0 h until corrected).
func (p *Plugin) markMissing(userID string, c *Current) error {
	err := p.updateMonth(userID, c.Month, func(m *Month) error {
		if e := m.find(c.EntryID); e != nil && e.End == 0 {
			e.Missing = true
			if e.paused() {
				e.Pauses[len(e.Pauses)-1].End = e.Pauses[len(e.Pauses)-1].Start
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return p.setCurrent(userID, nil)
}

func (p *Plugin) missingText(l string, e *Entry) string {
	return tr(l, "missing", dateLong(l, p.t(e.Start)), p.clock(e.Start))
}

func (p *Plugin) clockIn(userID string) (string, error) {
	l := p.langOf(userID)
	now := nowMs()
	prefix := ""
	if c, e := p.openEntry(userID); e != nil {
		if now-e.Start < p.staleMs() {
			if e.paused() {
				return "", userErr(tr(l, "in.alreadyp", p.clock(e.Start)))
			}
			return "", userErr(tr(l, "in.already", p.clock(e.Start)))
		}
		if err := p.markMissing(userID, c); err != nil {
			return "", err
		}
		prefix = p.missingText(l, e) + "\n"
	}
	month := p.monthOf(now)
	e := &Entry{ID: model.NewId(), Start: now}
	if err := p.updateMonth(userID, month, func(m *Month) error {
		m.Entries = append(m.Entries, e)
		return nil
	}); err != nil {
		return "", err
	}
	if err := p.setCurrent(userID, &Current{Month: month, EntryID: e.ID}); err != nil {
		return "", err
	}
	return prefix + tr(l, "in.ok", p.clock(now)), nil
}

// clockPause starts a pause, or ends it if one is running (resume=true forces "end").
func (p *Plugin) clockPause(userID string, mode string) (string, error) {
	l := p.langOf(userID)
	now := nowMs()
	c, e := p.openEntry(userID)
	if e == nil {
		return "", userErr(tr(l, "notin.start"))
	}
	if now-e.Start >= p.staleMs() {
		return "", userErr(tr(l, "stale.use", dateLong(l, p.t(e.Start)), p.clock(e.Start)))
	}
	paused := e.paused()
	if mode == "pause" && paused {
		return "", userErr(tr(l, "pause.already", p.clock(e.Pauses[len(e.Pauses)-1].Start)))
	}
	if mode == "resume" && !paused {
		return "", userErr(tr(l, "pause.not"))
	}
	var msg string
	err := p.updateMonth(userID, c.Month, func(m *Month) error {
		x := m.find(c.EntryID)
		if x == nil || x.End != 0 {
			return userErr(tr(l, "notfound"))
		}
		if x.paused() {
			last := &x.Pauses[len(x.Pauses)-1]
			last.End = now
			msg = tr(l, "resume.ok", p.clock(now), hm(last.End-last.Start))
		} else {
			x.Pauses = append(x.Pauses, Pause{Start: now})
			msg = tr(l, "pause.ok", p.clock(now))
		}
		return nil
	})
	return msg, err
}

func (p *Plugin) clockOut(userID string) (string, error) {
	l := p.langOf(userID)
	now := nowMs()
	c, e := p.openEntry(userID)
	if e == nil {
		return "", userErr(tr(l, "notin"))
	}
	if now-e.Start >= p.staleMs() {
		return "", userErr(tr(l, "out.stale", dateLong(l, p.t(e.Start)), p.clock(e.Start), p.conf().MaxHours))
	}
	var done *Entry
	err := p.updateMonth(userID, c.Month, func(m *Month) error {
		x := m.find(c.EntryID)
		if x == nil || x.End != 0 {
			return userErr(tr(l, "notfound"))
		}
		if x.paused() {
			x.Pauses[len(x.Pauses)-1].End = now
		}
		x.End = now
		done = x
		return nil
	})
	if err != nil {
		return "", err
	}
	_ = p.setCurrent(userID, nil)
	day := p.dayTotal(userID, p.dayKey(done.Start))
	msg := tr(l, "out.ok", p.clock(now), hm(done.workMs(now)))
	if pm := done.pauseMs(now); pm > 0 {
		msg += tr(l, "out.pause", hm(pm))
	}
	msg += tr(l, "out.day", hm(day))
	return msg, nil
}

// closeAt sets the end of the person's open (or forgotten) entry to a given time.
func (p *Plugin) closeAt(userID, entryMonth, entryID string, endMs int64, editor string) (*Entry, error) {
	l := p.langOf(editor)
	var done *Entry
	err := p.updateMonth(userID, entryMonth, func(m *Month) error {
		x := m.find(entryID)
		if x == nil {
			return userErr(tr(l, "close.notfound"))
		}
		if x.End != 0 && !x.Missing {
			return userErr(tr(l, "close.done"))
		}
		if endMs <= x.Start {
			return userErr(tr(l, "close.before", p.clock(x.Start)))
		}
		if endMs > nowMs()+60_000 {
			return userErr(tr(l, "close.future"))
		}
		for i := range x.Pauses {
			if x.Pauses[i].End == 0 || x.Pauses[i].End > endMs {
				x.Pauses[i].End = endMs
			}
		}
		x.End = endMs
		x.Missing = false
		x.EditedBy = editor
		done = x
		return nil
	})
	if err != nil {
		return nil, err
	}
	if c := p.getCurrent(userID); c != nil && c.EntryID == entryID {
		_ = p.setCurrent(userID, nil)
	}
	return done, nil
}

// dayTotal: worked ms on a local date (YYYY-MM-DD), open entry counted until now.
func (p *Plugin) dayTotal(userID, day string) int64 {
	m, _, err := p.loadMonth(userID, day[:7])
	if err != nil {
		return 0
	}
	now := nowMs()
	var t int64
	for _, e := range m.Entries {
		if p.dayKey(e.Start) == day {
			t += p.work(e, now)
		}
	}
	return t
}

// work: worked time of an entry; a forgotten open entry (older than MaxHours) counts 0 until fixed.
func (p *Plugin) work(e *Entry, now int64) int64 {
	if p.stale(e, now) {
		return 0
	}
	return e.workMs(now)
}

func (p *Plugin) stale(e *Entry, now int64) bool {
	return e.End == 0 && !e.Missing && now-e.Start >= p.staleMs()
}
