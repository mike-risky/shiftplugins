// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	pluginID   = "com.shiftplugins.timeclock"
	actionURL  = "/plugins/" + pluginID + "/interactive"
	dialogURL  = "/plugins/" + pluginID + "/dialog"
	botName    = "pointage"
	defaultTZ  = "Europe/Zurich"
	tickPeriod = 10 * time.Minute
)

type configuration struct {
	Managers      string // comma/space separated usernames
	ReminderHours int
	MaxHours      int
	SelfEdit      bool
	Timezone      string
}

type Plugin struct {
	plugin.MattermostPlugin
	botID   string
	cfgLock sync.RWMutex
	cfg     *configuration
	loc     *time.Location
	stop    chan struct{}
}

func (p *Plugin) OnActivate() error {
	botID, err := p.API.EnsureBotUser(&model.Bot{
		Username:    botName,
		DisplayName: "Pointage",
		Description: "Time clock by ShiftPlugins (shiftplugins.com)",
	})
	if err != nil {
		return fmt.Errorf("ensure bot: %w", err)
	}
	p.botID = botID
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          "pointage",
		AutoComplete:     true,
		AutoCompleteDesc: tr(p.serverLang(), "cmd.desc"),
		AutoCompleteHint: tr(p.serverLang(), "cmd.hint"),
		DisplayName:      "Pointage",
	}); err != nil {
		return err
	}
	p.stop = make(chan struct{})
	go p.ensureButtons()
	go p.reminderLoop(p.stop)
	return nil
}

func (p *Plugin) OnDeactivate() error {
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
	return nil
}

func (p *Plugin) OnConfigurationChange() error {
	c := &configuration{ReminderHours: 12, MaxHours: 16, Timezone: defaultTZ}
	if err := p.API.LoadPluginConfiguration(c); err != nil {
		return err
	}
	if c.ReminderHours < 0 {
		c.ReminderHours = 0
	}
	if c.MaxHours < 1 {
		c.MaxHours = 16
	}
	loc, err := time.LoadLocation(strings.TrimSpace(c.Timezone))
	if err != nil || strings.TrimSpace(c.Timezone) == "" {
		loc, _ = time.LoadLocation(defaultTZ)
	}
	p.cfgLock.Lock()
	p.cfg = c
	p.loc = loc
	p.cfgLock.Unlock()
	return nil
}

func (p *Plugin) conf() *configuration {
	p.cfgLock.RLock()
	defer p.cfgLock.RUnlock()
	if p.cfg == nil {
		return &configuration{ReminderHours: 12, MaxHours: 16, Timezone: defaultTZ}
	}
	return p.cfg
}

func (p *Plugin) tz() *time.Location {
	p.cfgLock.RLock()
	defer p.cfgLock.RUnlock()
	if p.loc == nil {
		l, _ := time.LoadLocation(defaultTZ)
		return l
	}
	return p.loc
}

// isManager: system admins and the usernames listed in the "Responsables" setting.
func (p *Plugin) isManager(userID string) bool {
	if p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		return true
	}
	list := strings.FieldsFunc(strings.ToLower(p.conf().Managers), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n'
	})
	if len(list) == 0 {
		return false
	}
	u, err := p.API.GetUser(userID)
	if err != nil {
		return false
	}
	for _, n := range list {
		if strings.TrimPrefix(n, "@") == strings.ToLower(u.Username) {
			return true
		}
	}
	return false
}

// canManageChannel: who may activate the pinned button in a channel.
func (p *Plugin) canManageChannel(userID, channelID string) bool {
	if p.isManager(userID) {
		return true
	}
	if cm, err := p.API.GetChannelMember(channelID, userID); err == nil && cm.SchemeAdmin {
		return true
	}
	if ch, err := p.API.GetChannel(channelID); err == nil && ch.TeamId != "" {
		if tm, err := p.API.GetTeamMember(ch.TeamId, userID); err == nil && tm.SchemeAdmin {
			return true
		}
	}
	return false
}

func displayName(u *model.User) string {
	if n := strings.TrimSpace(u.FirstName + " " + u.LastName); n != "" {
		return n
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

func (p *Plugin) userName(l, id string) string {
	if u, err := p.API.GetUser(id); err == nil {
		return displayName(u)
	}
	return tr(l, "deleted")
}

func main() {
	plugin.ClientMain(&Plugin{})
}
