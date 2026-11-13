// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

// Everything users see is a bot post with buttons (works in the mobile app):
//   - the pinned post in an activated channel (one tap to clock in/out),
//   - an ephemeral "panel" (status + buttons) opened by /pointage,
//   - ephemeral monthly views ("Mes heures", "Équipe" for managers).

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type btnSeq struct{ n int }

func (b *btnSeq) button(name, style string, ctx map[string]any) *model.PostAction {
	b.n++
	return &model.PostAction{
		Id:          fmt.Sprintf("pt%d", b.n),
		Type:        model.PostActionTypeButton,
		Name:        name,
		Style:       style,
		Integration: &model.PostActionIntegration{URL: actionURL, Context: ctx},
	}
}

func ctx(action string, kv ...string) map[string]any {
	m := map[string]any{"action": action}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// ---------- pinned post ----------

func pinnedPost(l, channelID, botID string) *model.Post {
	post := &model.Post{
		UserId:    botID,
		ChannelId: channelID,
		IsPinned:  true,
		Message:   tr(l, "title") + "\n" + tr(l, "pin.text"),
	}
	b := &btnSeq{}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{{
		Actions: []*model.PostAction{
			b.button(tr(l, "btn.in"), "primary", ctx("in", "src", "pin")),
			b.button(tr(l, "btn.toggle"), "default", ctx("pause", "src", "pin")),
			b.button(tr(l, "btn.out"), "default", ctx("out", "src", "pin")),
			b.button(tr(l, "btn.mine"), "default", ctx("mine", "src", "pin")),
		},
	}})
	return post
}

// ---------- panel ----------

func (p *Plugin) statusLine(l, userID string) (string, *Entry) {
	_, e := p.openEntry(userID)
	now := nowMs()
	if e == nil {
		return tr(l, "status.off"), nil
	}
	if e.paused() {
		return tr(l, "status.paused", p.clock(e.Pauses[len(e.Pauses)-1].Start), p.clock(e.Start)), e
	}
	if now-e.Start >= p.staleMs() {
		return tr(l, "status.stale", dateLong(l, p.t(e.Start)), p.clock(e.Start)), e
	}
	return tr(l, "status.on", p.clock(e.Start)), e
}

func (p *Plugin) buildPanel(userID, channelID, notice string) *model.Post {
	l := p.langOf(userID)
	status, e := p.statusLine(l, userID)
	now := nowMs()
	today := p.dayKey(now)
	text := status + tr(l, "panel.totals", hm(p.dayTotal(userID, today)), hm(p.monthTotal(userID, p.monthOf(now))))
	b := &btnSeq{}
	var acts []*model.PostAction
	switch {
	case e == nil:
		acts = append(acts, b.button(tr(l, "btn.in"), "primary", ctx("in")))
	case now-e.Start >= p.staleMs():
		acts = append(acts, b.button(tr(l, "btn.forgot"), "primary", ctx("forgot")))
	case e.paused():
		acts = append(acts, b.button(tr(l, "btn.resume"), "primary", ctx("resume")), b.button(tr(l, "btn.out"), "default", ctx("out")))
	default:
		acts = append(acts, b.button(tr(l, "btn.pause"), "default", ctx("pause")), b.button(tr(l, "btn.out"), "primary", ctx("out")))
		if now-e.Start >= int64(p.conf().ReminderHours)*3600_000 && p.conf().ReminderHours > 0 {
			acts = append(acts, b.button(tr(l, "btn.forgot"), "default", ctx("forgot")))
		}
	}
	acts = append(acts, b.button(tr(l, "btn.mine"), "default", ctx("mine")))
	if p.isManager(userID) {
		acts = append(acts, b.button(tr(l, "btn.team"), "default", ctx("team")))
	}
	msg := tr(l, "title")
	if notice != "" {
		msg = notice
	}
	post := &model.Post{UserId: p.botID, ChannelId: channelID, CreateAt: model.GetMillis(), Message: msg}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{{Text: text, Actions: acts}})
	return post
}

// ---------- monthly data ----------

type dayRow struct {
	Day     string // YYYY-MM-DD
	Entries []*Entry
	Work    int64
	Pause   int64
	Missing int
	Open    bool
	Edited  bool
}

func (p *Plugin) monthRows(userID, month string) ([]*dayRow, int64, int) {
	m, _, err := p.loadMonth(userID, month)
	if err != nil {
		return nil, 0, 0
	}
	now := nowMs()
	byDay := map[string]*dayRow{}
	var rows []*dayRow
	var total int64
	missing := 0
	for _, e := range m.Entries {
		d := p.dayKey(e.Start)
		r := byDay[d]
		if r == nil {
			r = &dayRow{Day: d}
			byDay[d] = r
			rows = append(rows, r)
		}
		r.Entries = append(r.Entries, e)
		w := p.work(e, now)
		r.Work += w
		total += w
		end := e.End
		if end == 0 {
			end = now
		}
		if !e.Missing && !p.stale(e, now) {
			r.Pause += e.pauseMs(end)
		}
		if e.Missing || p.stale(e, now) {
			r.Missing++
			missing++
		} else if e.End == 0 {
			r.Open = true
		}
		if e.EditedBy != "" {
			r.Edited = true
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Day < rows[j].Day })
	return rows, total, missing
}

func (p *Plugin) monthTotal(userID, month string) int64 {
	_, t, _ := p.monthRows(userID, month)
	return t
}

func (p *Plugin) periods(r *dayRow) string {
	var parts []string
	for _, e := range r.Entries {
		switch {
		case e.Missing || p.stale(e, nowMs()):
			parts = append(parts, p.clock(e.Start)+"–⚠️")
		case e.End == 0:
			parts = append(parts, p.clock(e.Start)+"–…")
		default:
			parts = append(parts, p.clock(e.Start)+"–"+p.clock(e.End))
		}
	}
	return strings.Join(parts, ", ")
}

// ---------- "Mes heures" (or one person's month, for managers) ----------

func (p *Plugin) buildMonth(viewerID, targetID, channelID, month string) *model.Post {
	l := p.langOf(viewerID)
	rows, total, missing := p.monthRows(targetID, month)
	var lines []string
	for _, r := range rows {
		t := p.t(r.Entries[0].Start)
		line := fmt.Sprintf("%s · %s · **%s**", dayShort(l, t), p.periods(r), hm(r.Work))
		if r.Pause > 0 {
			line += tr(l, "out.pause", hm(r.Pause))
		}
		if r.Edited {
			line += " ✏️"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, tr(l, "month.none"))
	}
	self := viewerID == targetID
	who := tr(l, "month.self")
	if !self {
		who = p.userName(l, targetID)
	}
	head := tr(l, "month.head", who, monthLabel(l, month), hm(total), len(rows))
	if missing > 0 {
		head += tr(l, "month.missing", missing)
	}

	b := &btnSeq{}
	nav := []*model.PostAction{
		b.button("‹ "+monthLabel(l, shiftMonth(month, -1)), "default", ctx("month", "month", shiftMonth(month, -1), "user", targetID)),
		b.button(monthLabel(l, shiftMonth(month, 1))+" ›", "default", ctx("month", "month", shiftMonth(month, 1), "user", targetID)),
	}
	tools := []*model.PostAction{b.button(tr(l, "btn.csv"), "default", ctx("csv", "month", month, "user", targetID))}
	manager := p.isManager(viewerID)
	if manager || p.conf().SelfEdit {
		tools = append(tools, b.button(tr(l, "btn.edit"), "default", ctx("edit", "user", targetID)))
	}
	if manager && !self {
		tools = append(tools, b.button(tr(l, "btn.back"), "default", ctx("team", "month", month)))
	} else {
		tools = append(tools, b.button(tr(l, "btn.clock"), "default", ctx("panel")))
	}
	post := &model.Post{UserId: p.botID, ChannelId: channelID, CreateAt: model.GetMillis(), Message: head}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{
		{Text: strings.Join(lines, "\n")},
		{Actions: nav},
		{Actions: tools},
	})
	return post
}

// ---------- "Équipe" (managers) ----------

type teamRow struct {
	ID, Name string
	Days     int
	Total    int64
	Missing  int
}

func (p *Plugin) teamRows(l, month string) []*teamRow {
	var out []*teamRow
	for _, id := range p.usersWithMonth(month) {
		rows, total, missing := p.monthRows(id, month)
		if len(rows) == 0 {
			continue
		}
		out = append(out, &teamRow{ID: id, Name: p.userName(l, id), Days: len(rows), Total: total, Missing: missing})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func (p *Plugin) onDuty(l string) []string {
	now := nowMs()
	var out []string
	for _, k := range p.allKeys("cur:") {
		id := strings.TrimPrefix(k, "cur:")
		_, e := p.openEntry(id)
		if e == nil || now-e.Start >= p.staleMs() {
			continue
		}
		s := tr(l, "duty.since", p.userName(l, id), p.clock(e.Start))
		if e.paused() {
			s += tr(l, "duty.paused")
		}
		out = append(out, s+")")
	}
	sort.Strings(out)
	return out
}

func (p *Plugin) buildTeam(viewerID, channelID, month string) *model.Post {
	l := p.langOf(viewerID)
	rows := p.teamRows(l, month)
	var lines []string
	var total int64
	for _, r := range rows {
		line := tr(l, "team.line", r.Name, hm(r.Total), r.Days)
		if r.Missing > 0 {
			line += tr(l, "month.missing", r.Missing)
		}
		lines = append(lines, line)
		total += r.Total
	}
	if len(lines) == 0 {
		lines = append(lines, tr(l, "month.none"))
	}
	head := tr(l, "team.head", monthLabel(l, month), hm(total))
	if duty := p.onDuty(l); len(duty) > 0 {
		head += tr(l, "team.on") + strings.Join(duty, ", ")
	} else {
		head += tr(l, "team.nobody")
	}
	b := &btnSeq{}
	pick := &model.PostAction{
		Id:          "ptpick",
		Type:        model.PostActionTypeSelect,
		Name:        tr(l, "btn.detail"),
		DataSource:  model.PostActionDataSourceUsers,
		Integration: &model.PostActionIntegration{URL: actionURL, Context: ctx("pick", "month", month)},
	}
	post := &model.Post{UserId: p.botID, ChannelId: channelID, CreateAt: model.GetMillis(), Message: head}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{
		{Text: strings.Join(lines, "\n")},
		{Actions: []*model.PostAction{pick}},
		{Actions: []*model.PostAction{
			b.button("‹ "+monthLabel(l, shiftMonth(month, -1)), "default", ctx("team", "month", shiftMonth(month, -1))),
			b.button(monthLabel(l, shiftMonth(month, 1))+" ›", "default", ctx("team", "month", shiftMonth(month, 1))),
		}},
		{Actions: []*model.PostAction{
			b.button(tr(l, "btn.csvall"), "primary", ctx("csv_team", "month", month)),
			b.button(tr(l, "btn.edit"), "default", ctx("edit")),
			b.button(tr(l, "btn.clock"), "default", ctx("panel")),
		}},
	})
	return post
}
