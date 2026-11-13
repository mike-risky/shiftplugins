// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Storage (plugin KV store):
//   t:<userID>:<YYYY-MM>  -> Month (entries whose start falls in that month, local time)
//   cur:<userID>          -> Current (the open entry, if the person is clocked in)
//   on:<channelID>        -> "1" if the pinned button is active in that channel
//   pin:<channelID>       -> id of the pinned post

type Pause struct {
	Start int64 `json:"s"`
	End   int64 `json:"e,omitempty"`
}

type Entry struct {
	ID       string  `json:"id"`
	Start    int64   `json:"s"`           // unix ms
	End      int64   `json:"e,omitempty"` // 0 while open
	Pauses   []Pause `json:"p,omitempty"`
	Missing  bool    `json:"m,omitempty"` // end was never recorded (forgotten clock-out)
	EditedBy string  `json:"ed,omitempty"`
	Note     string  `json:"n,omitempty"`
}

func (e *Entry) paused() bool {
	return len(e.Pauses) > 0 && e.Pauses[len(e.Pauses)-1].End == 0
}

// pauseMs: total pause time, open pause counted until `until`.
func (e *Entry) pauseMs(until int64) int64 {
	var t int64
	for _, pa := range e.Pauses {
		end := pa.End
		if end == 0 {
			end = until
		}
		if end > pa.Start {
			t += end - pa.Start
		}
	}
	return t
}

// workMs: worked time; an open entry counts until `now`.
func (e *Entry) workMs(now int64) int64 {
	end := e.End
	if end == 0 {
		if e.Missing {
			return 0
		}
		end = now
	}
	w := end - e.Start - e.pauseMs(end)
	if w < 0 {
		return 0
	}
	return w
}

type Month struct {
	Entries []*Entry `json:"entries"`
}

func (m *Month) find(id string) *Entry {
	for _, e := range m.Entries {
		if e.ID == id {
			return e
		}
	}
	return nil
}

type Current struct {
	Month    string `json:"month"`
	EntryID  string `json:"id"`
	Reminded bool   `json:"r,omitempty"`
}

type userErr string

func (e userErr) Error() string { return string(e) }

func monthKey(userID, month string) string { return "t:" + userID + ":" + month }
func curKey(userID string) string          { return "cur:" + userID }

func (p *Plugin) loadMonth(userID, month string) (*Month, []byte, error) {
	raw, appErr := p.API.KVGet(monthKey(userID, month))
	if appErr != nil {
		return nil, nil, appErr
	}
	m := &Month{}
	if raw != nil {
		if err := json.Unmarshal(raw, m); err != nil {
			return nil, nil, err
		}
	}
	return m, raw, nil
}

// updateMonth applies fn atomically (compare-and-set with retries).
func (p *Plugin) updateMonth(userID, month string, fn func(m *Month) error) error {
	for i := 0; i < 8; i++ {
		m, old, err := p.loadMonth(userID, month)
		if err != nil {
			return err
		}
		if err := fn(m); err != nil {
			return err
		}
		sort.SliceStable(m.Entries, func(a, b int) bool { return m.Entries[a].Start < m.Entries[b].Start })
		var nv []byte
		if len(m.Entries) > 0 {
			nv, _ = json.Marshal(m)
		}
		var ok bool
		var appErr error
		if nv == nil {
			if old == nil {
				return nil
			}
			ok, appErr = p.kvCASDelete(monthKey(userID, month), old)
		} else {
			ok, appErr = p.kvCAS(monthKey(userID, month), old, nv)
		}
		if appErr != nil {
			return appErr
		}
		if ok {
			return nil
		}
		time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
	}
	return fmt.Errorf("conflit d'écriture, réessayez")
}

func (p *Plugin) kvCAS(key string, old, nv []byte) (bool, error) {
	ok, err := p.API.KVCompareAndSet(key, old, nv)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (p *Plugin) kvCASDelete(key string, old []byte) (bool, error) {
	ok, err := p.API.KVCompareAndDelete(key, old)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (p *Plugin) getCurrent(userID string) *Current {
	raw, err := p.API.KVGet(curKey(userID))
	if err != nil || raw == nil {
		return nil
	}
	c := &Current{}
	if json.Unmarshal(raw, c) != nil || c.EntryID == "" {
		return nil
	}
	return c
}

func (p *Plugin) setCurrent(userID string, c *Current) error {
	if c == nil {
		if err := p.API.KVDelete(curKey(userID)); err != nil {
			return err
		}
		return nil
	}
	b, _ := json.Marshal(c)
	if err := p.API.KVSet(curKey(userID), b); err != nil {
		return err
	}
	return nil
}

// openEntry returns the person's open entry (nil if clocked out).
func (p *Plugin) openEntry(userID string) (*Current, *Entry) {
	c := p.getCurrent(userID)
	if c == nil {
		return nil, nil
	}
	m, _, err := p.loadMonth(userID, c.Month)
	if err != nil {
		return nil, nil
	}
	e := m.find(c.EntryID)
	if e == nil || e.End != 0 || e.Missing {
		_ = p.setCurrent(userID, nil)
		return nil, nil
	}
	return c, e
}

// allKeys lists every KV key of the plugin with the given prefix.
func (p *Plugin) allKeys(prefix string) []string {
	var out []string
	for page := 0; ; page++ {
		keys, err := p.API.KVList(page, 500)
		if err != nil || len(keys) == 0 {
			return out
		}
		for _, k := range keys {
			if strings.HasPrefix(k, prefix) {
				out = append(out, k)
			}
		}
		if len(keys) < 500 {
			return out
		}
	}
}

// usersWithMonth returns the ids of everybody who has entries in that month.
func (p *Plugin) usersWithMonth(month string) []string {
	var ids []string
	for _, k := range p.allKeys("t:") {
		parts := strings.Split(k, ":")
		if len(parts) == 3 && parts[2] == month {
			ids = append(ids, parts[1])
		}
	}
	return ids
}
