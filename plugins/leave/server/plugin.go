// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	pluginID     = "com.shiftplugins.leave"
	interactPath = "/plugins/" + pluginID + "/interactive"
	dialogPath   = "/plugins/" + pluginID + "/dialog"
	maxNoteLen   = 500
	maxRangeDays = 366

	colorPending   = "#F5A623"
	colorApproved  = "#74FAC1"
	colorRefused   = "#D24B4E"
	colorCancelled = "#9B9B9B"
)

const botUsername = "leave"

type configuration struct {
	Managers      string
	LeaveTypes    string
	CountWeekends bool
	AnnualDays    int
	TimeZone      string
}

// types returns the configured leave types, or the defaults in the server language.
func (c *configuration) types(serverLang string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range strings.Split(c.LeaveTypes, ",") {
		t = strings.TrimSpace(t)
		if t != "" && len([]rune(t)) <= 40 && !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return defaultTypesByLang[serverLang]
	}
	return out
}

func (c *configuration) managerNames() []string {
	var out []string
	for _, n := range strings.FieldsFunc(c.Managers, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' }) {
		n = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(n), "@"))
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

type Plugin struct {
	plugin.MattermostPlugin
	botID   string
	cfgLock sync.RWMutex
	cfg     *configuration
}

// userErr is an error shown to the user: a catalog key, optionally with a status
// (translated too) as argument.
type userErr struct{ key, status string }

func (e userErr) Error() string { return e.key }

func errText(lang string, err error) string {
	if ue, ok := err.(userErr); ok {
		if ue.status != "" {
			return tr(lang, ue.key, tr(lang, "w."+ue.status))
		}
		return tr(lang, ue.key)
	}
	return tr(lang, "err.generic")
}

// ---------- lifecycle ----------

func (p *Plugin) OnActivate() error {
	sl := p.serverLang()
	botID, err := p.API.EnsureBotUser(&model.Bot{
		Username:    botUsername,
		DisplayName: tr(sl, "bot.display"),
		Description: tr(sl, "cmd.desc"),
	})
	if err != nil {
		return fmt.Errorf("ensure bot: %w", err)
	}
	p.botID = botID
	// Keep the bot's display name in the server language (also after a language change).
	dn, desc := tr(sl, "bot.display"), tr(sl, "cmd.desc")
	if _, err := p.API.PatchBot(botID, &model.BotPatch{DisplayName: &dn, Description: &desc}); err != nil {
		p.API.LogWarn("leave: patch bot", "err", err.Error())
	}
	// The command name follows the server language (/leave, /conges, /ferien, /ferie);
	// its subcommands are understood in every language.
	return p.API.RegisterCommand(&model.Command{
		Trigger:          p.trigger(),
		AutoComplete:     true,
		AutoCompleteHint: tr(sl, "cmd.hint"),
		AutoCompleteDesc: tr(sl, "cmd.desc"),
		DisplayName:      tr(sl, "bot.display"),
	})
}

func (p *Plugin) OnConfigurationChange() error {
	c := &configuration{}
	if err := p.API.LoadPluginConfiguration(c); err != nil {
		return err
	}
	loc := defaultLoc
	if c.TimeZone != "" {
		if l, err := time.LoadLocation(c.TimeZone); err == nil {
			loc = l
		} else {
			p.API.LogWarn("leave: unknown time zone, using Europe/Zurich", "tz", c.TimeZone)
		}
	}
	p.cfgLock.Lock()
	p.cfg = c
	tz.Store(loc)
	p.cfgLock.Unlock()
	return nil
}

func (p *Plugin) config() *configuration {
	p.cfgLock.RLock()
	defer p.cfgLock.RUnlock()
	if p.cfg == nil {
		return &configuration{}
	}
	return p.cfg
}

func (p *Plugin) trigger() string { return triggerByLang[p.serverLang()] }

func (p *Plugin) types() []string { return p.config().types(p.serverLang()) }

// ---------- people ----------

func displayName(u *model.User) string {
	if n := strings.TrimSpace(u.FirstName + " " + u.LastName); n != "" {
		return n
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

func (p *Plugin) name(userID string) string {
	if u, err := p.API.GetUser(userID); err == nil {
		return displayName(u)
	}
	return "?"
}

// managers returns the IDs of the users listed in the settings (active accounts only).
func (p *Plugin) managers() []string {
	var out []string
	for _, n := range p.config().managerNames() {
		if u, err := p.API.GetUserByUsername(n); err == nil && u.DeleteAt == 0 && !u.IsBot {
			out = append(out, u.Id)
		}
	}
	return out
}

func (p *Plugin) sysadmins() []string {
	var out []string
	for page := 0; page < 10; page++ {
		users, err := p.API.GetUsers(&model.UserGetOptions{Role: model.SystemAdminRoleId, Page: page, PerPage: 100, Active: true})
		if err != nil || len(users) == 0 {
			break
		}
		for _, u := range users {
			if !u.IsBot {
				out = append(out, u.Id)
			}
		}
		if len(users) < 100 {
			break
		}
	}
	return out
}

func without(list []string, s string) []string {
	out := make([]string, 0, len(list))
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (p *Plugin) isSysadmin(userID string) bool {
	u, err := p.API.GetUser(userID)
	return err == nil && u.IsSystemAdmin()
}

// ---------- slash command ----------

func ephemeral(t string) (*model.CommandResponse, *model.AppError) {
	return &model.CommandResponse{ResponseType: model.CommandResponseTypeEphemeral, Text: t}, nil
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	lang := p.lang(args.UserId)
	fields := strings.Fields(args.Command)
	sub := ""
	if len(fields) > 1 {
		sub = subCommand(fields[1])
	}
	switch sub {
	case "new":
		p.openRequestDialog(args.UserId, args.TriggerId)
		return &model.CommandResponse{}, nil
	case "mine":
		p.sendView(args.UserId, args.ChannelId, "mine")
	case "queue":
		if !p.loadDeciders().isManager(args.UserId) {
			return ephemeral(tr(lang, "cmd.onlyMgr"))
		}
		p.sendView(args.UserId, args.ChannelId, "queue")
	case "team":
		p.sendView(args.UserId, args.ChannelId, "team")
	case "button":
		ok := p.isSysadmin(args.UserId) || p.API.HasPermissionToChannel(args.UserId, args.ChannelId, model.PermissionManagePublicChannelProperties) && p.API.HasPermissionToChannel(args.UserId, args.ChannelId, model.PermissionManagePrivateChannelProperties)
		if !ok {
			return ephemeral(tr(lang, "cmd.onlyAdmin"))
		}
		if err := p.postButton(args.ChannelId, lang); err != nil {
			return ephemeral(tr(lang, "cmd.pinFail", err.Error()))
		}
		return ephemeral(tr(lang, "cmd.pinned"))
	case "help":
		return ephemeral(tr(lang, "help", strings.TrimPrefix(fields[0], "/")))
	default:
		p.sendView(args.UserId, args.ChannelId, "menu")
	}
	return &model.CommandResponse{}, nil
}

// postButton posts and pins a bot message with the main buttons (in the language of the
// admin who posts it), so the feature is one tap away on mobile (channel info → pinned).
func (p *Plugin) postButton(channelID, lang string) error {
	n := 0
	post := &model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		Message:   tr(lang, "pin.text"),
		IsPinned:  true,
	}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{{
		Actions: []*model.PostAction{
			button(&n, tr(lang, "pin.new"), "primary", map[string]any{"action": "new"}),
			button(&n, tr(lang, "nav.mine"), "default", map[string]any{"action": "view", "view": "mine"}),
			button(&n, tr(lang, "nav.team"), "default", map[string]any{"action": "view", "view": "team"}),
		},
	}})
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return appErr
	}
	return nil
}

// ---------- HTTP ----------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
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

func main() {
	plugin.ClientMain(&Plugin{})
}
