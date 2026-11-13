// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
	_ "time/tzdata"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRefused   = "refused"
	StatusCancelled = "cancelled"

	PartFull = "full"
	PartAM   = "am"
	PartPM   = "pm"
)

// Request is one leave request. Stored under "r:<id>".
type Request struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Type      string  `json:"type"`
	Start     string  `json:"start"` // YYYY-MM-DD
	End       string  `json:"end"`   // YYYY-MM-DD, inclusive
	Part      string  `json:"part"`  // full / am / pm (half days only when Start == End)
	Days      float64 `json:"days"`
	Note      string  `json:"note,omitempty"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"created_at"`
	DeciderID string  `json:"decider_id,omitempty"`
	DecidedAt int64   `json:"decided_at,omitempty"`
	Comment   string  `json:"comment,omitempty"`
	// Bot posts that show this request and must be refreshed when it changes:
	// manager user ID -> post ID in the manager's DM with the bot, and the requester's copy.
	ManagerPosts  map[string]string `json:"manager_posts,omitempty"`
	RequesterPost string            `json:"requester_post,omitempty"`
}

// IndexEntry is the compact copy kept in the "idx" list so listings don't load every request.
type IndexEntry struct {
	ID     string  `json:"id"`
	UserID string  `json:"u"`
	Type   string  `json:"t"`
	Start  string  `json:"s"`
	End    string  `json:"e"`
	Days   float64 `json:"d"`
	Status string  `json:"st"`
}

func (r *Request) entry() IndexEntry {
	return IndexEntry{ID: r.ID, UserID: r.UserID, Type: r.Type, Start: r.Start, End: r.End, Days: r.Days, Status: r.Status}
}

func reqKey(id string) string { return "r:" + id }

const indexKey = "idx"

// kvUpdate applies fn atomically to the JSON value stored at key (retrying on concurrent writes).
func (p *Plugin) kvUpdate(key string, fn func(old []byte) ([]byte, error)) error {
	for i := 0; i < 20; i++ {
		old, appErr := p.API.KVGet(key)
		if appErr != nil {
			return appErr
		}
		next, err := fn(old)
		if err != nil {
			return err
		}
		ok, appErr := p.API.KVCompareAndSet(key, old, next)
		if appErr != nil {
			return appErr
		}
		if ok {
			return nil
		}
		time.Sleep(time.Duration(10*(i+1)) * time.Millisecond)
	}
	return userErr{key: "err.busy"}
}

func (p *Plugin) getRequest(id string) (*Request, error) {
	if !model.IsValidId(id) {
		return nil, userErr{key: "err.notFound"}
	}
	b, appErr := p.API.KVGet(reqKey(id))
	if appErr != nil {
		return nil, appErr
	}
	if len(b) == 0 {
		return nil, userErr{key: "err.notFound"}
	}
	var r Request
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// updateRequest changes a stored request atomically and keeps the index in sync.
func (p *Plugin) updateRequest(id string, fn func(r *Request) error) (*Request, error) {
	var out *Request
	err := p.kvUpdate(reqKey(id), func(old []byte) ([]byte, error) {
		if len(old) == 0 {
			return nil, userErr{key: "err.notFound"}
		}
		var r Request
		if err := json.Unmarshal(old, &r); err != nil {
			return nil, err
		}
		if err := fn(&r); err != nil {
			return nil, err
		}
		out = &r
		return json.Marshal(&r)
	})
	if err != nil {
		return nil, err
	}
	return out, p.saveIndexEntry(out.entry())
}

func (p *Plugin) createRequest(r *Request) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	ok, appErr := p.API.KVCompareAndSet(reqKey(r.ID), nil, b)
	if appErr != nil {
		return appErr
	}
	if !ok {
		return fmt.Errorf("id collision")
	}
	return p.saveIndexEntry(r.entry())
}

func (p *Plugin) saveIndexEntry(e IndexEntry) error {
	return p.kvUpdate(indexKey, func(old []byte) ([]byte, error) {
		var list []IndexEntry
		if len(old) > 0 {
			if err := json.Unmarshal(old, &list); err != nil {
				return nil, err
			}
		}
		found := false
		for i := range list {
			if list[i].ID == e.ID {
				list[i] = e
				found = true
				break
			}
		}
		if !found {
			list = append(list, e)
		}
		return json.Marshal(list)
	})
}

func (p *Plugin) loadIndex() []IndexEntry {
	b, appErr := p.API.KVGet(indexKey)
	if appErr != nil || len(b) == 0 {
		return nil
	}
	var list []IndexEntry
	if err := json.Unmarshal(b, &list); err != nil {
		p.API.LogError("conges: index illisible", "err", err.Error())
		return nil
	}
	return list
}

// filterIndex returns the entries matching keep, sorted by start date.
func (p *Plugin) filterIndex(keep func(e IndexEntry) bool) []IndexEntry {
	var out []IndexEntry
	for _, e := range p.loadIndex() {
		if keep(e) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Start != out[j].Start {
			return out[i].Start < out[j].Start
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func active(status string) bool { return status == StatusPending || status == StatusApproved }

// overlapping returns an active request of userID that overlaps [start, end], if any.
func (p *Plugin) overlapping(userID, start, end string) *IndexEntry {
	for _, e := range p.loadIndex() {
		if e.UserID == userID && active(e.Status) && e.Start <= end && e.End >= start {
			e := e
			return &e
		}
	}
	return nil
}
