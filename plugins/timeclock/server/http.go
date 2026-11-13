// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	switch r.URL.Path {
	case "/interactive":
		p.handleInteractive(w, r, userID)
	case "/dialog":
		p.handleDialog(w, r, userID)
	default:
		http.NotFound(w, r)
	}
}

func errText(l string, err error) string {
	if ue, ok := err.(userErr); ok {
		return string(ue)
	}
	return tr(l, "error", err.Error())
}

// ---------- slash command ----------

// Sub-commands are accepted in all four languages.
var cmdAlias = map[string]string{}

func init() {
	for k, words := range map[string][]string{
		"in":      {"debut", "début", "commencer", "start", "in", "beginn", "einstempeln", "inizio", "entrata"},
		"pause":   {"pause", "break", "pausa"},
		"resume":  {"reprise", "reprendre", "resume", "weiter", "riprendi", "ripresa"},
		"out":     {"fin", "terminer", "stop", "out", "end", "ende", "ausstempeln", "fine", "uscita"},
		"hours":   {"heures", "mois", "mes", "hours", "month", "stunden", "monat", "ore", "mese"},
		"team":    {"equipe", "équipe", "admin", "team", "squadra"},
		"export":  {"export", "csv", "exportieren", "esporta"},
		"fix":     {"corriger", "correction", "fix", "edit", "korrigieren", "correggi"},
		"enable":  {"activer", "enable", "aktivieren", "attiva", "attivare"},
		"disable": {"desactiver", "désactiver", "disable", "deaktivieren", "disattiva", "disattivare"},
		"button":  {"bouton", "button", "knopf", "pulsante"},
		"help":    {"aide", "help", "hilfe", "aiuto"},
	} {
		for _, w := range words {
			cmdAlias[w] = k
		}
	}
}

func parseMonth(s, def string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, true
	}
	if monthRe.MatchString(s) {
		return s, true
	}
	for _, layout := range []string{"1.2006", "01.2006", "1/2006", "01/2006", "1-2006", "01-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01"), true
		}
	}
	return "", false
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	uid, ch := args.UserId, args.ChannelId
	l := p.langOf(uid)
	reply := func(t string) (*model.CommandResponse, *model.AppError) {
		return &model.CommandResponse{ResponseType: model.CommandResponseTypeEphemeral, Text: t}, nil
	}
	fields := strings.Fields(args.Command)
	word, arg := "", ""
	if len(fields) > 1 {
		word = strings.ToLower(fields[1])
	}
	if len(fields) > 2 {
		arg = fields[2]
	}
	sub := cmdAlias[word]
	if word != "" && sub == "" {
		return reply(tr(l, "unknown.cmd") + "\n" + tr(l, "help"))
	}
	thisMonth := p.monthOf(nowMs())
	panel := func(notice string) (*model.CommandResponse, *model.AppError) {
		p.API.SendEphemeralPost(uid, p.buildPanel(uid, ch, notice))
		return &model.CommandResponse{}, nil
	}
	run := func(f func(string) (string, error)) (*model.CommandResponse, *model.AppError) {
		msg, err := f(uid)
		if err != nil {
			return panel("⚠️ " + errText(l, err))
		}
		return panel(msg)
	}

	switch sub {
	case "":
		return panel("")
	case "help":
		return reply(tr(l, "help"))
	case "in":
		return run(p.clockIn)
	case "pause":
		return run(func(u string) (string, error) { return p.clockPause(u, "pause") })
	case "resume":
		return run(func(u string) (string, error) { return p.clockPause(u, "resume") })
	case "out":
		return run(p.clockOut)
	case "hours":
		month, ok := parseMonth(arg, thisMonth)
		if !ok {
			return reply(tr(l, "badmonth", word))
		}
		p.API.SendEphemeralPost(uid, p.buildMonth(uid, uid, ch, month))
		return &model.CommandResponse{}, nil
	case "team":
		if !p.isManager(uid) {
			return reply(tr(l, "managers.only"))
		}
		month, ok := parseMonth(arg, thisMonth)
		if !ok {
			return reply(tr(l, "badmonth", word))
		}
		p.API.SendEphemeralPost(uid, p.buildTeam(uid, ch, month))
		return &model.CommandResponse{}, nil
	case "export":
		month, ok := parseMonth(arg, thisMonth)
		if !ok {
			return reply(tr(l, "badmonth", word))
		}
		if p.isManager(uid) {
			return reply(p.sendCSV(uid, nil, month))
		}
		return reply(p.sendCSV(uid, []string{uid}, month))
	case "fix":
		if !p.isManager(uid) && !p.conf().SelfEdit {
			return reply(tr(l, "fix.only"))
		}
		if err := p.openEditDialog(args.TriggerId, uid, uid, ch); err != nil {
			return reply("⚠️ " + errText(l, err))
		}
		return &model.CommandResponse{}, nil
	case "enable", "disable", "button":
		if !p.canManageChannel(uid, ch) {
			return reply(tr(l, "chadmin.only"))
		}
		if sub == "disable" {
			p.setEnabled(ch, false)
			p.removeButtonPost(ch)
			return reply(tr(l, "disabled"))
		}
		p.removeButtonPost(ch)
		p.setEnabled(ch, true)
		p.postButton(ch)
		return reply(tr(l, "enabled"))
	}
	return reply(tr(l, "unknown.cmd"))
}

// ---------- pinned post per channel ----------

func enabledKey(ch string) string { return "on:" + ch }
func pinKey(ch string) string     { return "pin:" + ch }

func (p *Plugin) setEnabled(ch string, on bool) {
	if on {
		_ = p.API.KVSet(enabledKey(ch), []byte("1"))
	} else {
		_ = p.API.KVDelete(enabledKey(ch))
	}
}

func (p *Plugin) postButton(ch string) {
	if created, err := p.API.CreatePost(pinnedPost(p.serverLang(), ch, p.botID)); err == nil {
		_ = p.API.KVSet(pinKey(ch), []byte(created.Id))
	} else {
		p.API.LogError("pointage: cannot post pinned message", "channel", ch, "err", err.Error())
	}
}

func (p *Plugin) removeButtonPost(ch string) {
	if id, err := p.API.KVGet(pinKey(ch)); err == nil && len(id) > 0 {
		_ = p.API.DeletePost(string(id))
		_ = p.API.KVDelete(pinKey(ch))
	}
}

// ensureButtons re-posts the pinned message in active channels where it was deleted.
func (p *Plugin) ensureButtons() {
	for _, k := range p.allKeys("on:") {
		ch := strings.TrimPrefix(k, "on:")
		if id, err := p.API.KVGet(pinKey(ch)); err == nil && len(id) > 0 {
			if post, err := p.API.GetPost(string(id)); err == nil && post.DeleteAt == 0 {
				continue
			}
		}
		p.postButton(ch)
	}
}

// ---------- buttons ----------

func (p *Plugin) handleInteractive(w http.ResponseWriter, r *http.Request, userID string) {
	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	str := func(k string) string { v, _ := req.Context[k].(string); return v }
	action, src, ch := str("action"), str("src"), req.ChannelId
	fromPin := src == "pin"
	ok := func() { writeJSON(w, http.StatusOK, &model.PostActionIntegrationResponse{}) }
	// replies as a bot ephemeral post (more reliable than ephemeral_text across clients)
	text := func(t string) {
		p.API.SendEphemeralPost(userID, &model.Post{UserId: p.botID, ChannelId: ch, Message: t})
		ok()
	}
	// show replaces the ephemeral message the button was on (or posts a new one from the pinned post)
	show := func(post *model.Post) {
		if fromPin || req.PostId == "" {
			p.API.SendEphemeralPost(userID, post)
		} else {
			post.Id = req.PostId
			p.API.UpdateEphemeralPost(userID, post)
		}
		ok()
	}
	month := str("month")
	if !monthRe.MatchString(month) {
		month = p.monthOf(nowMs())
	}
	target := str("user")
	manager := p.isManager(userID)
	l := p.langOf(userID)
	if target == "" || (target != userID && !manager) {
		target = userID
	}

	clockRes := func(msg string, err error) {
		if err != nil {
			msg = "⚠️ " + errText(l, err)
		}
		if fromPin {
			text(msg)
			return
		}
		show(p.buildPanel(userID, ch, msg))
	}

	switch action {
	case "in":
		clockRes(p.clockIn(userID))
	case "pause":
		clockRes(p.clockPause(userID, "toggle"))
	case "resume":
		clockRes(p.clockPause(userID, "resume"))
	case "out":
		clockRes(p.clockOut(userID))
	case "panel":
		show(p.buildPanel(userID, ch, ""))
	case "mine", "month":
		show(p.buildMonth(userID, target, ch, month))
	case "team", "pick":
		if !manager {
			text(tr(l, "managers.only"))
			return
		}
		if action == "pick" {
			if sel := str("selected_option"); model.IsValidId(sel) {
				show(p.buildMonth(userID, sel, ch, month))
				return
			}
		}
		show(p.buildTeam(userID, ch, month))
	case "csv":
		text(p.sendCSV(userID, []string{target}, month))
	case "csv_team":
		if !manager {
			text(tr(l, "managers.only"))
			return
		}
		text(p.sendCSV(userID, nil, month))
	case "edit":
		if !manager && !p.conf().SelfEdit {
			text(tr(l, "fix.only"))
			return
		}
		if err := p.openEditDialog(req.TriggerId, userID, target, ch); err != nil {
			text("⚠️ " + errText(l, err))
			return
		}
		ok()
	case "forgot":
		if err := p.openForgotDialog(req.TriggerId, userID); err != nil {
			text("⚠️ " + errText(l, err))
			return
		}
		ok()
	default:
		text(tr(l, "unknown.act"))
	}
}
