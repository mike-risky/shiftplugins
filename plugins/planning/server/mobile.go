// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

// Mobile-friendly view: an ephemeral message listing one week of shifts with
// Prendre / Libérer buttons. Interactive buttons work in the mobile app.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/mattermost/mattermost/server/public/model"
)

const listDays = 7

var zurich = func() *time.Location {
	if l, err := time.LoadLocation("Europe/Zurich"); err == nil {
		return l
	}
	return time.UTC
}()

func today() string { return time.Now().In(zurich).Format("2006-01-02") }

func addDays(date string, n int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// shiftsBetween returns the shifts with from <= date < to (at most two month buckets).
func (p *Plugin) shiftsBetween(channelID, from, to string) []*Shift {
	var out []*Shift
	seen := map[string]bool{}
	for _, d := range []string{from, addDays(to, -1)} {
		month := d[:7]
		if seen[month] {
			continue
		}
		seen[month] = true
		m, _, err := p.loadMonth(channelID, month)
		if err != nil {
			continue
		}
		for _, s := range m.Shifts {
			if s.Date >= from && s.Date < to {
				out = append(out, s)
			}
		}
	}
	return out
}

func button(n *int, name, style string, ctx map[string]any) *model.PostAction {
	*n++
	return &model.PostAction{
		Id:    fmt.Sprintf("pl%d", *n),
		Type:  model.PostActionTypeButton,
		Name:  name,
		Style: style,
		Integration: &model.PostActionIntegration{
			URL:     "/plugins/com.shiftplugins.planning/interactive",
			Context: ctx,
		},
	}
}

func (p *Plugin) buildList(userID, channelID, from string) *model.Post {
	ul := p.userLang(userID)
	to := addDays(from, listDays)
	shifts := p.shiftsBetween(channelID, from, to)
	names := map[string]string{}
	name := func(id string) string {
		if n, ok := names[id]; ok {
			return n
		}
		n := "?"
		if u, err := p.API.GetUser(id); err == nil {
			n = displayName(u)
		}
		names[id] = n
		return n
	}

	n := 0
	var atts []*model.MessageAttachment
	byDate := map[string][]*Shift{}
	var dates []string
	for _, s := range shifts {
		if _, ok := byDate[s.Date]; !ok {
			dates = append(dates, s.Date)
		}
		byDate[s.Date] = append(byDate[s.Date], s)
	}
	for _, d := range dates {
		var lines []string
		var actions []*model.PostAction
		for _, s := range byDate[d] {
			mine := contains(s.Users, userID)
			full := len(s.Users) >= s.Places
			icon := "⚪"
			switch {
			case mine:
				icon = "✅"
			case full:
				icon = "⛔"
			}
			if s.Locked {
				icon += "🔒"
			}
			who := make([]string, 0, len(s.Users))
			for _, u := range s.Users {
				label := name(u)
				if o := s.offerOf(u); o != nil {
					if o.To != "" {
						label += tr(ul, "tag_pending", name(o.To))
					} else {
						label += tr(ul, "tag_offered")
					}
				}
				who = append(who, label)
			}
			line := fmt.Sprintf("%s **%s** · %d/%d", icon, shiftLabel(s), len(s.Users), s.Places)
			if len(who) > 0 {
				line += " · " + strings.Join(who, ", ")
			}
			lines = append(lines, line)
			mk := func(action string, extra map[string]any) map[string]any {
				c := map[string]any{"shift_id": s.ID, "month": s.Date[:7], "from": from, "action": action}
				for k, v := range extra {
					c[k] = v
				}
				return c
			}
			swaps := p.config().AllowSwaps && s.Date >= today()
			switch {
			case mine:
				if !s.Locked {
					actions = append(actions, button(&n, tr(ul, "btn_release", shiftLabel(s)), "default", mk("release", nil)))
				}
				if swaps {
					if s.offerOf(userID) == nil {
						actions = append(actions, button(&n, tr(ul, "btn_offer"), "default", mk("offer", nil)))
					} else if s.offerOf(userID).To == "" {
						actions = append(actions, button(&n, tr(ul, "btn_cancel_offer"), "default", mk("cancel_offer", nil)))
					}
				}
			default:
				if !s.Locked && !full {
					actions = append(actions, button(&n, tr(ul, "btn_take", shiftLabel(s)), "primary", mk("take", nil)))
				}
				if swaps {
					for _, o := range s.Offers {
						if o.To == "" {
							actions = append(actions, button(&n, tr(ul, "btn_claim_from", shiftLabel(s), name(o.From)), "primary", mk("claim", map[string]any{"from_user": o.From})))
						}
					}
				}
			}
		}
		atts = append(atts, &model.MessageAttachment{
			Title:   capFirst(fmtDate(ul, d)),
			Text:    strings.Join(lines, "\n"),
			Actions: actions,
		})
	}
	if len(dates) == 0 {
		atts = append(atts, &model.MessageAttachment{Text: tr(ul, "list_empty")})
	}
	atts = append(atts, &model.MessageAttachment{
		Actions: []*model.PostAction{
			button(&n, tr(ul, "btn_prev"), "default", map[string]any{"action": "nav", "from": addDays(from, -listDays)}),
			button(&n, tr(ul, "btn_next"), "default", map[string]any{"action": "nav", "from": to}),
		},
	})

	post := &model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		CreateAt:  model.GetMillis(),
		Message:   tr(ul, "list_title", shortDate(ul, from), shortDate(ul, addDays(to, -1))),
	}
	model.ParseMessageAttachment(post, atts)
	return post
}

func (p *Plugin) handleInteractive(w http.ResponseWriter, r *http.Request, userID string) {
	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		p.failT(w, http.StatusBadRequest, userID, "e_bad_request")
		return
	}
	ul := p.userLang(userID)
	reply := func(text string) {
		writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{EphemeralText: text})
	}
	replyKey := func(key string) { reply(capFirst(tr(ul, key)) + ".") }
	channelID := req.ChannelId
	str := func(k string) string { v, _ := req.Context[k].(string); return v }
	action, from := str("action"), str("from")
	if !validDate(from) {
		from = today()
	}
	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionReadChannel) {
		replyKey("e_not_member")
		return
	}
	if !p.isEnabled(channelID) {
		replyKey("e_not_enabled")
		return
	}

	switch action {
	case "open":
		p.API.SendEphemeralPost(userID, p.buildList(userID, channelID, today()))
		writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{})
		return
	case "nav":
	case "take", "release", "offer", "cancel_offer", "claim", "approve", "refuse":
		month := str("month")
		if !monthRe.MatchString(month) {
			replyKey("e_bad_request")
			return
		}
		onPost := str("src") == "post"
		areq := &actionReq{ChannelID: channelID, Month: month, Action: action, ShiftID: str("shift_id"), FromUser: str("from_user")}
		notice, err := p.shiftAction(areq, userID, p.isAdmin(userID, channelID))
		if err != nil {
			reply("⚠️ " + capFirst(p.errText(userID, err)) + ".")
			if !onPost {
				p.refreshList(userID, channelID, from, req.PostId)
			}
			return
		}
		p.postNotice(channelID, userID, notice, areq.buttons...)
		if onPost {
			// The request on this channel post is handled: remove its buttons.
			if post, perr := p.API.GetPost(req.PostId); perr == nil {
				post.DelProp(model.PostPropsAttachments)
				post.Message += "\n" + tr(p.serverLang(), "handled")
				writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{Update: post})
				return
			}
			writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{})
			return
		}
	default:
		replyKey("e_unknown_action")
		return
	}
	p.refreshList(userID, channelID, from, req.PostId)
	writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{})
}

func capFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToUpper(string(r[:1])) + string(r[1:])
}

func (p *Plugin) refreshList(userID, channelID, from, postID string) {
	post := p.buildList(userID, channelID, from)
	post.Id = postID
	p.API.UpdateEphemeralPost(userID, post)
}
