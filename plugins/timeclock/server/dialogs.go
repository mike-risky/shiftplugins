// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

var hhmmRe = regexp.MustCompile(`^(\d{1,2})\s*[:hH.]\s*(\d{2})?$|^(\d{1,2})\s*[hH]?$`)

// parseClock accepts 8:00, 08:00, 8h00, 8h, 8.30, 17
func parseClock(s string) (int, int, bool) {
	s = strings.TrimSpace(s)
	m := hhmmRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	hs, ms := m[1], m[2]
	if hs == "" {
		hs = m[3]
	}
	h, _ := strconv.Atoi(hs)
	mi := 0
	if ms != "" {
		mi, _ = strconv.Atoi(ms)
	}
	if h > 23 || mi > 59 {
		return 0, 0, false
	}
	return h, mi, true
}

func (p *Plugin) parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"02.01.2006", "2.1.2006", "02.01.06", "2.1.06", "2006-01-02", "02/01/2006", "2/1/2006"} {
		if t, err := time.ParseInLocation(layout, s, p.tz()); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseMinutes accepts 30, 0:30, 1h, 1h15
func parseMinutes(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, true
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < 24*60 {
		return n, true
	}
	if h, m, ok := parseClock(s); ok {
		return h*60 + m, true
	}
	return 0, false
}

func at(day time.Time, h, m int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location())
}

// ---------- "Corriger une journée" ----------

type editState struct {
	Target string `json:"t,omitempty"`
}

func (p *Plugin) openEditDialog(triggerID, editorID, targetID, channelID string) error {
	l := p.langOf(editorID)
	if triggerID == "" {
		return userErr(tr(l, "dlg.notrigger"))
	}
	manager := p.isManager(editorID)
	var els []model.DialogElement
	st := editState{}
	if manager {
		els = append(els, model.DialogElement{
			DisplayName: tr(l, "edit.person"), Name: "user", Type: "select", DataSource: model.PostActionDataSourceUsers, Default: targetID,
		})
	} else {
		st.Target = editorID
	}
	today := p.t(nowMs()).Format("02.01.2006")
	els = append(els,
		model.DialogElement{DisplayName: tr(l, "f.date"), Name: "date", Type: "text", Default: today, Placeholder: tr(l, "f.dateph"), MaxLength: 10},
		model.DialogElement{DisplayName: tr(l, "f.start"), Name: "start", Type: "text", Placeholder: "08:00", Optional: true, MaxLength: 5,
			HelpText: tr(l, "f.starthelp")},
		model.DialogElement{DisplayName: tr(l, "f.end"), Name: "end", Type: "text", Placeholder: "17:00", Optional: true, MaxLength: 5,
			HelpText: tr(l, "f.endhelp")},
		model.DialogElement{DisplayName: tr(l, "f.pause"), Name: "pause", Type: "text", SubType: "number", Placeholder: "30", Optional: true, MaxLength: 4},
		model.DialogElement{DisplayName: tr(l, "f.note"), Name: "note", Type: "text", Optional: true, MaxLength: 100},
	)
	b, _ := json.Marshal(st)
	if err := p.API.OpenInteractiveDialog(model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       dialogURL,
		Dialog: model.Dialog{
			CallbackId:       "edit",
			Title:            tr(l, "edit.title"),
			IntroductionText: tr(l, "edit.intro"),
			SubmitLabel:      tr(l, "f.save"),
			Elements:         els,
			State:            string(b),
		},
	}); err != nil {
		return err
	}
	return nil
}

func (p *Plugin) submitEdit(req *model.SubmitDialogRequest, editorID string) *model.SubmitDialogResponse {
	str := func(k string) string { return subStr(req.Submission, k) }
	errs := map[string]string{}
	l := p.langOf(editorID)
	manager := p.isManager(editorID)
	var st editState
	_ = json.Unmarshal([]byte(req.State), &st)
	target := st.Target
	if manager {
		target = str("user")
	}
	if !manager && (!p.conf().SelfEdit || target != editorID) {
		return &model.SubmitDialogResponse{Error: tr(l, "fix.only")}
	}
	if !model.IsValidId(target) {
		errs["user"] = tr(l, "e.person")
	} else if u, err := p.API.GetUser(target); err != nil || u.IsBot {
		errs["user"] = tr(l, "e.personnf")
	}
	day, okDay := p.parseDate(str("date"))
	if !okDay {
		errs["date"] = tr(l, "e.date")
	}
	startS, endS := str("start"), str("end")
	sh, sm, okS := parseClock(startS)
	eh, em, okE := parseClock(endS)
	clear := startS == "" && endS == ""
	if !clear {
		if !okS {
			errs["start"] = tr(l, "e.time")
		}
		if !okE {
			errs["end"] = tr(l, "e.time")
		}
	}
	pauseMin, okP := parseMinutes(str("pause"))
	if !okP {
		errs["pause"] = tr(l, "e.pause")
	}
	if len(errs) > 0 {
		return &model.SubmitDialogResponse{Errors: errs}
	}

	var ne *Entry
	if !clear {
		start := at(day, sh, sm)
		end := at(day, eh, em)
		if !end.After(start) {
			end = end.AddDate(0, 0, 1)
		}
		if end.UnixMilli() > nowMs()+60_000 {
			return &model.SubmitDialogResponse{Errors: map[string]string{"end": tr(l, "e.future")}}
		}
		dur := end.Sub(start)
		pause := time.Duration(pauseMin) * time.Minute
		if pause >= dur {
			return &model.SubmitDialogResponse{Errors: map[string]string{"pause": tr(l, "e.pauselong")}}
		}
		ne = &Entry{ID: model.NewId(), Start: start.UnixMilli(), End: end.UnixMilli(), EditedBy: editorID, Note: str("note")}
		if pause > 0 {
			ps := start.Add((dur - pause) / 2)
			ne.Pauses = []Pause{{Start: ps.UnixMilli(), End: ps.Add(pause).UnixMilli()}}
		}
	}

	dayKey := day.Format("2006-01-02")
	replacedOpen := ""
	err := p.updateMonth(target, day.Format("2006-01"), func(m *Month) error {
		kept := m.Entries[:0]
		for _, e := range m.Entries {
			if p.dayKey(e.Start) == dayKey {
				if e.End == 0 && !e.Missing {
					if nowMs()-e.Start < p.staleMs() {
						return userErr(tr(l, "e.ondut"))
					}
					replacedOpen = e.ID
				}
				continue
			}
			kept = append(kept, e)
		}
		m.Entries = kept
		if ne != nil {
			m.Entries = append(m.Entries, ne)
		}
		return nil
	})
	if err != nil {
		return &model.SubmitDialogResponse{Error: errText(l, err)}
	}
	if c := p.getCurrent(target); replacedOpen != "" && c != nil && c.EntryID == replacedOpen {
		_ = p.setCurrent(target, nil)
	}

	what := func(l string) string {
		if ne == nil {
			return tr(l, "edit.cleared")
		}
		return tr(l, "edit.what", p.clock(ne.Start), p.clock(ne.End), hm(ne.pauseMs(ne.End)), hm(ne.workMs(ne.End)))
	}
	p.API.SendEphemeralPost(editorID, &model.Post{
		UserId: p.botID, ChannelId: req.ChannelId,
		Message: tr(l, "edit.done", dateLong(l, day), p.userName(l, target), what(l)),
	})
	if target != editorID {
		tl := p.langOf(target)
		p.dm(target, tr(tl, "edit.dm", p.userName(tl, editorID), dateLong(tl, day), what(tl)))
	}
	return &model.SubmitDialogResponse{}
}

// ---------- "J'ai oublié de pointer la fin" ----------

type forgotState struct {
	Month string `json:"m"`
	ID    string `json:"id"`
}

func (p *Plugin) openForgotDialog(triggerID, userID string) error {
	l := p.langOf(userID)
	c, e := p.openEntry(userID)
	if e == nil {
		return userErr(tr(l, "forgot.none"))
	}
	if triggerID == "" {
		return userErr(tr(l, "dlg.notrigger"))
	}
	b, _ := json.Marshal(forgotState{Month: c.Month, ID: c.EntryID})
	startT := p.t(e.Start)
	if err := p.API.OpenInteractiveDialog(model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       dialogURL,
		Dialog: model.Dialog{
			CallbackId:       "forgot",
			Title:            tr(l, "forgot.title"),
			IntroductionText: tr(l, "forgot.intro", dateLong(l, startT), p.clock(e.Start)),
			SubmitLabel:      tr(l, "f.save"),
			State:            string(b),
			Elements: []model.DialogElement{
				{DisplayName: tr(l, "forgot.date"), Name: "date", Type: "text", Default: startT.Format("02.01.2006"), Placeholder: tr(l, "f.dateph"), MaxLength: 10},
				{DisplayName: tr(l, "forgot.time"), Name: "end", Type: "text", Placeholder: "17:00", MaxLength: 5},
			},
		},
	}); err != nil {
		return err
	}
	return nil
}

func (p *Plugin) submitForgot(req *model.SubmitDialogRequest, userID string) *model.SubmitDialogResponse {
	l := p.langOf(userID)
	var st forgotState
	if json.Unmarshal([]byte(req.State), &st) != nil || !monthRe.MatchString(st.Month) {
		return &model.SubmitDialogResponse{Error: tr(l, "e.form")}
	}
	str := func(k string) string { return subStr(req.Submission, k) }
	day, ok := p.parseDate(str("date"))
	if !ok {
		return &model.SubmitDialogResponse{Errors: map[string]string{"date": tr(l, "e.date")}}
	}
	h, m, ok := parseClock(str("end"))
	if !ok {
		return &model.SubmitDialogResponse{Errors: map[string]string{"end": tr(l, "e.time")}}
	}
	end := at(day, h, m).UnixMilli()
	e, err := p.closeAt(userID, st.Month, st.ID, end, userID)
	if err != nil {
		return &model.SubmitDialogResponse{Errors: map[string]string{"end": errText(l, err)}}
	}
	p.API.SendEphemeralPost(userID, p.buildPanel(userID, req.ChannelId,
		tr(l, "forgot.ok", dateLong(l, p.t(end)), p.clock(end), hm(e.workMs(end)))))
	return &model.SubmitDialogResponse{}
}

func (p *Plugin) handleDialog(w http.ResponseWriter, r *http.Request, userID string) {
	var req model.SubmitDialogRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Cancelled {
		writeJSON(w, http.StatusOK, &model.SubmitDialogResponse{})
		return
	}
	var resp *model.SubmitDialogResponse
	switch req.CallbackId {
	case "edit":
		resp = p.submitEdit(&req, userID)
	case "forgot":
		resp = p.submitForgot(&req, userID)
	default:
		resp = &model.SubmitDialogResponse{Error: tr(p.langOf(userID), "dlg.unknown")}
	}
	writeJSON(w, http.StatusOK, resp)
}

// dm sends a direct message from the bot.
func (p *Plugin) dm(userID, msg string) {
	ch, err := p.API.GetDirectChannel(userID, p.botID)
	if err != nil {
		return
	}
	_, _ = p.API.CreatePost(&model.Post{UserId: p.botID, ChannelId: ch.Id, Message: msg})
}

// subStr reads a dialog value as text (number fields may arrive as numbers).
func subStr(sub map[string]any, k string) string {
	switch v := sub[k].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
