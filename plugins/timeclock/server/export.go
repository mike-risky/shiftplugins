// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// CSV for Excel (Swiss/French locale): ";" separator, UTF-8 with BOM, CRLF.
func csvCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		s = "'" + s
	}
	if strings.ContainsAny(s, ";\"\n\r") {
		s = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func csvLine(b *bytes.Buffer, cells ...string) {
	for i, c := range cells {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(csvCell(c))
	}
	b.WriteString("\r\n")
}

func (p *Plugin) buildCSV(l string, userIDs []string, month string) ([]byte, int) {
	type person struct {
		id, name, username string
	}
	var people []person
	for _, id := range userIDs {
		pr := person{id: id, name: tr(l, "deleted"), username: ""}
		if u, err := p.API.GetUser(id); err == nil {
			pr.name, pr.username = displayName(u), u.Username
		}
		people = append(people, pr)
	}
	// sort by name
	for i := 1; i < len(people); i++ {
		for j := i; j > 0 && strings.ToLower(people[j].name) < strings.ToLower(people[j-1].name); j-- {
			people[j], people[j-1] = people[j-1], people[j]
		}
	}

	b := &bytes.Buffer{}
	b.WriteString("\xEF\xBB\xBF")
	csvLine(b, strings.Split(tr(l, "csv.cols"), ";")...)
	now := nowMs()
	type tot struct {
		days    int
		ms      int64
		missing int
	}
	totals := map[string]*tot{}
	count := 0
	for _, pr := range people {
		rows, total, missing := p.monthRows(pr.id, month)
		if len(rows) == 0 {
			continue
		}
		count++
		totals[pr.id] = &tot{days: len(rows), ms: total, missing: missing}
		for _, r := range rows {
			for _, e := range r.Entries {
				st := p.t(e.Start)
				end, remark := "", []string{}
				var pause int64
				switch {
				case e.Missing || p.stale(e, now):
					remark = append(remark, tr(l, "r.missing"))
				case e.End == 0:
					remark = append(remark, tr(l, "r.open"))
					pause = e.pauseMs(now)
				default:
					end = p.clock(e.End)
					if p.dayKey(e.End) != p.dayKey(e.Start) {
						end += " (+1)"
					}
					pause = e.pauseMs(e.End)
				}
				if e.EditedBy != "" {
					if e.EditedBy == pr.id {
						remark = append(remark, tr(l, "r.self"))
					} else {
						remark = append(remark, tr(l, "r.by", p.userName(l, e.EditedBy)))
					}
				}
				if e.Note != "" {
					remark = append(remark, e.Note)
				}
				w := p.work(e, now)
				csvLine(b, pr.name, pr.username, st.Format("02.01.2006"), dayAbbr[l][st.Weekday()], p.clock(e.Start), end,
					hm(pause), hm(w), fmt.Sprint(minutes(w)), strings.Join(remark, ", "))
			}
		}
	}
	b.WriteString("\r\n")
	csvLine(b, tr(l, "csv.total", strings.ToUpper(monthLabel(l, month))))
	csvLine(b, strings.Split(tr(l, "csv.totcols"), ";")...)
	for _, pr := range people {
		t := totals[pr.id]
		if t == nil {
			continue
		}
		csvLine(b, pr.name, pr.username, fmt.Sprint(t.days), hm(t.ms), fmt.Sprint(minutes(t.ms)), fmt.Sprint(t.missing))
	}
	return b.Bytes(), count
}

// sendCSV builds the file and sends it to the requester in a private message from the bot
// (works on mobile too). userIDs nil = everybody with hours that month.
func (p *Plugin) sendCSV(requesterID string, userIDs []string, month string) string {
	l := p.langOf(requesterID)
	team := userIDs == nil
	if team {
		userIDs = p.usersWithMonth(month)
	}
	data, n := p.buildCSV(l, userIDs, month)
	if n == 0 {
		return tr(l, "csv.none", monthLabel(l, month))
	}
	ch, err := p.API.GetDirectChannel(requesterID, p.botID)
	if err != nil {
		return tr(l, "csv.dmerr", err.Error())
	}
	name := tr(l, "csv.file") + "-" + month
	label := tr(l, "csv.team")
	if !team && len(userIDs) == 1 {
		if u, err := p.API.GetUser(userIDs[0]); err == nil {
			name += "-" + u.Username
			n := displayName(u)
			if strings.ContainsRune("AEIOUYHaeiouyhÉéÈè", []rune(n)[0]) {
				label = tr(l, "csv.ofv", n)
			} else {
				label = tr(l, "csv.of", n)
			}
		}
	}
	fi, err := p.API.UploadFile(data, ch.Id, name+".csv")
	if err != nil {
		return tr(l, "csv.upload", err.Error())
	}
	_, err = p.API.CreatePost(&model.Post{
		UserId: p.botID, ChannelId: ch.Id, FileIds: []string{fi.Id},
		Message: tr(l, "csv.post", label, monthLabel(l, month), p.t(nowMs()).Format("02.01.2006 15:04")),
	})
	if err != nil {
		return tr(l, "csv.upload", err.Error())
	}
	return tr(l, "csv.sent", botName)
}

// ---------- reminder ----------

func (p *Plugin) reminderLoop(stop chan struct{}) {
	t := time.NewTicker(tickPeriod)
	defer t.Stop()
	time.Sleep(10 * time.Second)
	p.remind()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			p.remind()
		}
	}
}

// remind sends a private message to people still clocked in after ReminderHours.
func (p *Plugin) remind() {
	hours := p.conf().ReminderHours
	if hours <= 0 {
		return
	}
	now := nowMs()
	for _, k := range p.allKeys("cur:") {
		uid := strings.TrimPrefix(k, "cur:")
		c, e := p.openEntry(uid)
		if e == nil || c.Reminded || now-e.Start < int64(hours)*3600_000 {
			continue
		}
		c.Reminded = true
		if p.setCurrent(uid, c) != nil {
			continue
		}
		ch, err := p.API.GetDirectChannel(uid, p.botID)
		if err != nil {
			continue
		}
		l := p.langOf(uid)
		post := &model.Post{
			UserId: p.botID, ChannelId: ch.Id,
			Message: tr(l, "remind", dateLong(l, p.t(e.Start)), p.clock(e.Start), (now-e.Start)/3600_000),
		}
		b := &btnSeq{}
		model.ParseMessageAttachment(post, []*model.MessageAttachment{{Actions: []*model.PostAction{
			b.button(tr(l, "btn.endtime"), "primary", ctx("forgot", "src", "pin")),
			b.button(tr(l, "btn.outnow"), "default", ctx("out", "src", "pin")),
			b.button(tr(l, "btn.still"), "default", ctx("panel", "src", "pin")),
		}}})
		_, _ = p.API.CreatePost(post)
	}
}
