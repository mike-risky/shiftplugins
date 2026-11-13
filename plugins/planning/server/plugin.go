// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	maxPlaces   = 10
	maxRepeat   = 120
	maxLabelLen = 40
)

var (
	monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
	timeRe  = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

type configuration struct {
	Notify       bool
	AllowSwaps   bool
	SwapApproval bool
}

type Plugin struct {
	plugin.MattermostPlugin
	botID   string
	cfgLock sync.RWMutex
	cfg     *configuration
}

type Shift struct {
	ID     string   `json:"id"`
	Date   string   `json:"date"`
	Label  string   `json:"label"`
	Start  string   `json:"start,omitempty"`
	End    string   `json:"end,omitempty"`
	Places int      `json:"places"`
	Users  []string `json:"users"`
	Locked bool     `json:"locked,omitempty"`
	Offers []*Offer `json:"offers,omitempty"`
}

// Offer: a member offers their place on a shift; To is set when a colleague
// asked to take it and an admin still has to approve.
type Offer struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
}

func (s *Shift) offerOf(userID string) *Offer {
	for _, o := range s.Offers {
		if o.From == userID {
			return o
		}
	}
	return nil
}

func (s *Shift) dropOffer(userID string) {
	out := []*Offer{}
	for _, o := range s.Offers {
		if o.From != userID {
			out = append(out, o)
		}
	}
	s.Offers = out
}

// Month is only a storage bucket (one KV entry per channel and month);
// the calendar shown to users is continuous.
type Month struct {
	Shifts []*Shift `json:"shifts"`
}

func (m *Month) find(id string) *Shift {
	for _, s := range m.Shifts {
		if s.ID == id {
			return s
		}
	}
	return nil
}

type userInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type userErr string

func (e userErr) Error() string { return string(e) }

// subAlias maps the /planning sub-commands in every language to one action.
var subAlias = map[string]string{
	"enable": "enable", "on": "enable", "activer": "enable", "aktivieren": "enable", "attiva": "enable", "attivare": "enable",
	"disable": "disable", "off": "disable", "desactiver": "disable", "désactiver": "disable", "deaktivieren": "disable", "disattiva": "disable", "disattivare": "disable",
	"button": "button", "bouton": "button", "knopf": "button", "pulsante": "button",
}

// ---------- lifecycle ----------

func (p *Plugin) OnActivate() error {
	botID, err := p.API.EnsureBotUser(&model.Bot{
		Username:    "planning",
		DisplayName: "Planning",
		Description: "Shift planning by ShiftPlugins (shiftplugins.com)",
	})
	if err != nil {
		return fmt.Errorf("ensure bot: %w", err)
	}
	p.botID = botID
	go p.ensureButtons()
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "planning",
		AutoComplete:     true,
		AutoCompleteDesc: tr(p.serverLang(), "cmd_desc"),
		DisplayName:      "Planning",
	})
}

func (p *Plugin) OnConfigurationChange() error {
	c := &configuration{Notify: true, AllowSwaps: true}
	if err := p.API.LoadPluginConfiguration(c); err != nil {
		return err
	}
	p.cfgLock.Lock()
	p.cfg = c
	p.cfgLock.Unlock()
	return nil
}

func (p *Plugin) config() configuration {
	p.cfgLock.RLock()
	defer p.cfgLock.RUnlock()
	if p.cfg == nil {
		return configuration{Notify: true, AllowSwaps: true}
	}
	return *p.cfg
}

func (p *Plugin) notifyOn() bool {
	p.cfgLock.RLock()
	defer p.cfgLock.RUnlock()
	return p.cfg == nil || p.cfg.Notify
}

// On desktop/web the webapp intercepts /planning and opens the calendar.
// This only runs where the plugin UI isn't available (mobile app).
func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	ul := p.userLang(args.UserId)
	reply := func(key string, a ...any) (*model.CommandResponse, *model.AppError) {
		return &model.CommandResponse{ResponseType: model.CommandResponseTypeEphemeral, Text: tr(ul, key, a...)}, nil
	}
	fields := strings.Fields(args.Command)
	sub := ""
	if len(fields) > 1 {
		sub = strings.ToLower(fields[1])
	}
	switch subAlias[sub] {
	case "button":
		if !p.isAdmin(args.UserId, args.ChannelId) {
			return reply("cmd_admin_only")
		}
		if !p.isEnabled(args.ChannelId) {
			return reply("cmd_enable_first")
		}
		p.removeButtonPost(args.ChannelId)
		p.postButton(args.ChannelId)
		return reply("cmd_button_reposted")
	case "enable", "disable":
		if !p.isAdmin(args.UserId, args.ChannelId) {
			return reply("cmd_toggle_admin")
		}
		on := subAlias[sub] == "enable"
		if err := p.setEnabled(args.ChannelId, on); err != nil {
			return reply("cmd_error", err.Error())
		}
		p.announceEnabled(args.ChannelId, on)
		if on {
			return reply("cmd_enabled")
		}
		return reply("cmd_disabled")
	}
	if !p.isEnabled(args.ChannelId) {
		return reply("cmd_not_enabled")
	}
	p.API.SendEphemeralPost(args.UserId, p.buildList(args.UserId, args.ChannelId, today()))
	return &model.CommandResponse{}, nil
}

func (p *Plugin) announceEnabled(channelID string, on bool) {
	if p.botID == "" {
		return
	}
	p.removeButtonPost(channelID)
	if on {
		p.postButton(channelID)
		return
	}
	_, _ = p.API.CreatePost(&model.Post{UserId: p.botID, ChannelId: channelID, Message: tr(p.serverLang(), "post_disabled")})
}

// ensureButtons posts the pinned button in every activated channel that doesn't have one yet
// (channels activated before v1.2.0).
func (p *Plugin) ensureButtons() {
	for page := 0; ; page++ {
		keys, err := p.API.KVList(page, 200)
		if err != nil || len(keys) == 0 {
			return
		}
		for _, k := range keys {
			if !strings.HasPrefix(k, "on:") {
				continue
			}
			ch := strings.TrimPrefix(k, "on:")
			if id, err := p.API.KVGet(pinKey(ch)); err == nil && len(id) > 0 {
				if post, err := p.API.GetPost(string(id)); err == nil && post.DeleteAt == 0 {
					continue
				}
			}
			p.postButton(ch)
		}
		if len(keys) < 200 {
			return
		}
	}
}

func pinKey(channelID string) string { return "pin:" + channelID }

// postButton posts and pins the "Ouvrir le planning" message, so it is one tap away
// on mobile (channel info → Messages épinglés) and on desktop.
func (p *Plugin) postButton(channelID string) {
	post := &model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		IsPinned:  true,
		Message:   tr(p.serverLang(), "post_pinned"),
	}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{{Actions: []*model.PostAction{p.openButton("primary")}}})
	if created, err := p.API.CreatePost(post); err == nil {
		_ = p.API.KVSet(pinKey(channelID), []byte(created.Id))
	}
}

func (p *Plugin) removeButtonPost(channelID string) {
	if id, err := p.API.KVGet(pinKey(channelID)); err == nil && len(id) > 0 {
		_ = p.API.DeletePost(string(id))
		_ = p.API.KVDelete(pinKey(channelID))
	}
}

func (p *Plugin) openButton(style string) *model.PostAction {
	return &model.PostAction{
		Id:    "plopen",
		Type:  model.PostActionTypeButton,
		Name:  tr(p.serverLang(), "btn_open"),
		Style: style,
		Integration: &model.PostActionIntegration{
			URL:     "/plugins/com.shiftplugins.planning/interactive",
			Context: map[string]any{"action": "open"},
		},
	}
}

// ---------- storage ----------

func enabledKey(channelID string) string { return "on:" + channelID }

func (p *Plugin) isEnabled(channelID string) bool {
	v, err := p.API.KVGet(enabledKey(channelID))
	return err == nil && string(v) == "1"
}

func (p *Plugin) setEnabled(channelID string, on bool) error {
	if on {
		if err := p.API.KVSet(enabledKey(channelID), []byte("1")); err != nil {
			return err
		}
		return nil
	}
	if err := p.API.KVDelete(enabledKey(channelID)); err != nil {
		return err
	}
	return nil
}

func monthKey(channelID, month string) string { return "m:" + channelID + ":" + month }

func (p *Plugin) loadMonth(channelID, month string) (*Month, []byte, error) {
	raw, appErr := p.API.KVGet(monthKey(channelID, month))
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
func (p *Plugin) updateMonth(channelID, month string, fn func(m *Month) error) (*Month, error) {
	for i := 0; i < 8; i++ {
		m, old, err := p.loadMonth(channelID, month)
		if err != nil {
			return nil, err
		}
		if err := fn(m); err != nil {
			return nil, err
		}
		sort.SliceStable(m.Shifts, func(a, b int) bool {
			x, y := m.Shifts[a], m.Shifts[b]
			if x.Date != y.Date {
				return x.Date < y.Date
			}
			return x.Start < y.Start
		})
		nv, _ := json.Marshal(m)
		ok, appErr := p.API.KVCompareAndSet(monthKey(channelID, month), old, nv)
		if appErr != nil {
			return nil, appErr
		}
		if ok {
			return m, nil
		}
		time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
	}
	return nil, userErr("e_conflict")
}

// ---------- helpers ----------

func (p *Plugin) isAdmin(userID, channelID string) bool {
	if p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
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
	if u.Nickname != "" {
		return u.Nickname
	}
	if n := strings.TrimSpace(u.FirstName + " " + u.LastName); n != "" {
		return n
	}
	return u.Username
}

func shiftLabel(s *Shift) string {
	l := s.Label
	if s.Start != "" {
		l += " " + s.Start
		if s.End != "" {
			l += "–" + s.End
		}
	}
	return strings.TrimSpace(l)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func without(list []string, s string) []string {
	out := []string{}
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

func validDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// failT sends a translated error (key from the catalog) in the user's language.
func (p *Plugin) failT(w http.ResponseWriter, code int, userID, key string) {
	fail(w, code, tr(p.userLang(userID), key))
}

// ---------- HTTP ----------

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		fail(w, http.StatusUnauthorized, tr(p.serverLang(), "e_not_logged_in"))
		return
	}
	switch {
	case r.URL.Path == "/api/v1/month" && r.Method == http.MethodGet:
		p.handleGet(w, r, userID)
	case r.URL.Path == "/api/v1/action" && r.Method == http.MethodPost:
		p.handleAction(w, r, userID)
	case r.URL.Path == "/interactive" && r.Method == http.MethodPost:
		p.handleInteractive(w, r, userID)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) checkAccess(w http.ResponseWriter, userID, channelID, month string) bool {
	if !monthRe.MatchString(month) || !model.IsValidId(channelID) {
		p.failT(w, http.StatusBadRequest, userID, "e_bad_params")
		return false
	}
	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionReadChannel) {
		p.failT(w, http.StatusForbidden, userID, "e_not_member")
		return false
	}
	return true
}

func (p *Plugin) respond(w http.ResponseWriter, userID, channelID, month string, m *Month) {
	admin := p.isAdmin(userID, channelID)
	users := map[string]userInfo{}
	var members []userInfo
	if admin {
		for page := 0; ; page++ {
			list, err := p.API.GetUsersInChannel(channelID, "username", page, 200)
			if err != nil || len(list) == 0 {
				break
			}
			for _, u := range list {
				if u.IsBot || u.DeleteAt != 0 {
					continue
				}
				ui := userInfo{u.Id, u.Username, displayName(u)}
				members = append(members, ui)
				users[u.Id] = ui
			}
			if len(list) < 200 {
				break
			}
		}
		sort.Slice(members, func(i, j int) bool { return strings.ToLower(members[i].Name) < strings.ToLower(members[j].Name) })
	}
	ids := []string{userID}
	for _, s := range m.Shifts {
		ids = append(ids, s.Users...)
	}
	for _, id := range ids {
		if _, ok := users[id]; ok {
			continue
		}
		if u, err := p.API.GetUser(id); err == nil {
			users[id] = userInfo{u.Id, u.Username, displayName(u)}
		} else {
			users[id] = userInfo{id, "?", tr(p.userLang(userID), "deleted_user")}
		}
	}
	if m.Shifts == nil {
		m.Shifts = []*Shift{}
	}
	if members == nil {
		members = []userInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"month":    month,
		"shifts":   m.Shifts,
		"users":    users,
		"members":  members,
		"me":       userID,
		"admin":    admin,
		"enabled":  p.isEnabled(channelID),
		"swaps":    p.config().AllowSwaps,
		"approval": p.config().SwapApproval,
		"today":    today(),
	})
}

func (p *Plugin) handleGet(w http.ResponseWriter, r *http.Request, userID string) {
	channelID := r.URL.Query().Get("channel_id")
	month := r.URL.Query().Get("month")
	if !p.checkAccess(w, userID, channelID, month) {
		return
	}
	if !p.isEnabled(channelID) {
		p.respond(w, userID, channelID, month, &Month{})
		return
	}
	m, _, err := p.loadMonth(channelID, month)
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.respond(w, userID, channelID, month, m)
}

type actionReq struct {
	ChannelID string   `json:"channel_id"`
	Month     string   `json:"month"` // month currently displayed (bucket of shift_id)
	Action    string   `json:"action"`
	ShiftID   string   `json:"shift_id"`
	UserID    string   `json:"user_id"`
	Dates     []string `json:"dates"`
	Label     string   `json:"label"`
	Start     string   `json:"start"`
	End       string   `json:"end"`
	Places    int      `json:"places"`
	FromUser  string   `json:"from_user"`

	buttons []*model.PostAction
}

var adminActions = map[string]bool{
	"create": true, "update": true, "delete": true, "lock": true, "unlock": true,
	"lock_month": true, "unlock_month": true, "assign": true, "remove": true,
	"enable": true, "disable": true, "approve": true, "refuse": true,
}

func (p *Plugin) handleAction(w http.ResponseWriter, r *http.Request, userID string) {
	var req actionReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		p.failT(w, http.StatusBadRequest, userID, "e_bad_request")
		return
	}
	if !p.checkAccess(w, userID, req.ChannelID, req.Month) {
		return
	}
	admin := p.isAdmin(userID, req.ChannelID)
	if adminActions[req.Action] && !admin {
		p.failT(w, http.StatusForbidden, userID, "e_admin_only")
		return
	}

	if req.Action != "enable" && !p.isEnabled(req.ChannelID) {
		p.failT(w, http.StatusConflict, userID, "e_not_enabled")
		return
	}

	var err error
	var notice string
	switch req.Action {
	case "enable", "disable":
		on := req.Action == "enable"
		if err = p.setEnabled(req.ChannelID, on); err == nil {
			p.announceEnabled(req.ChannelID, on)
		}
	case "create":
		err = p.createShifts(&req)
	default:
		notice, err = p.shiftAction(&req, userID, admin)
	}
	if err != nil {
		if _, ok := err.(userErr); ok {
			fail(w, http.StatusConflict, p.errText(userID, err))
		} else {
			fail(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	p.postNotice(req.ChannelID, userID, notice, req.buttons...)

	m, _, lerr := p.loadMonth(req.ChannelID, req.Month)
	if lerr != nil {
		fail(w, http.StatusInternalServerError, lerr.Error())
		return
	}
	p.respond(w, userID, req.ChannelID, req.Month, m)
}

func (p *Plugin) postNotice(channelID, userID, notice string, extra ...*model.PostAction) {
	if notice == "" || !p.notifyOn() || p.botID == "" {
		return
	}
	if strings.Contains(notice, "{user}") {
		if u, err := p.API.GetUser(userID); err == nil {
			notice = strings.ReplaceAll(notice, "{user}", "**"+displayName(u)+"**")
		}
	}
	post := &model.Post{UserId: p.botID, ChannelId: channelID, Message: notice}
	model.ParseMessageAttachment(post, []*model.MessageAttachment{{Actions: append(extra, p.openButton("default"))}})
	_, _ = p.API.CreatePost(post)
}

func cleanShiftFields(req *actionReq) error {
	req.Label = strings.TrimSpace(req.Label)
	if r := []rune(req.Label); len(r) > maxLabelLen {
		req.Label = string(r[:maxLabelLen])
	}
	if req.Label == "" {
		req.Label = "Shift"
	}
	if req.Start != "" && !timeRe.MatchString(req.Start) {
		return userErr("e_bad_start")
	}
	if req.End != "" && !timeRe.MatchString(req.End) {
		return userErr("e_bad_end")
	}
	if req.Start == "" {
		req.End = ""
	}
	if req.Places < 1 || req.Places > maxPlaces {
		return userErr("e_places_range")
	}
	return nil
}

func (p *Plugin) createShifts(req *actionReq) error {
	if err := cleanShiftFields(req); err != nil {
		return err
	}
	if len(req.Dates) == 0 || len(req.Dates) > maxRepeat {
		return userErr("e_dates_range")
	}
	byMonth := map[string][]string{}
	for _, d := range req.Dates {
		if !validDate(d) {
			return userErr("e_bad_date")
		}
		byMonth[d[:7]] = append(byMonth[d[:7]], d)
	}
	for month, dates := range byMonth {
		_, err := p.updateMonth(req.ChannelID, month, func(m *Month) error {
			for _, d := range dates {
				m.Shifts = append(m.Shifts, &Shift{
					ID: model.NewId(), Date: d, Label: req.Label, Start: req.Start, End: req.End,
					Places: req.Places, Users: []string{},
				})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) shiftAction(req *actionReq, userID string, admin bool) (string, error) {
	if req.Action == "update" {
		if err := cleanShiftFields(req); err != nil {
			return "", err
		}
	}
	if req.Action == "assign" {
		if !model.IsValidId(req.UserID) || !p.API.HasPermissionToChannel(req.UserID, req.ChannelID, model.PermissionReadChannel) {
			return "", userErr("e_not_in_channel")
		}
	}
	L := p.serverLang()
	var notice string
	_, err := p.updateMonth(req.ChannelID, req.Month, func(m *Month) error {
		notice = ""
		req.buttons = nil
		switch req.Action {
		case "lock_month", "unlock_month":
			lock := req.Action == "lock_month"
			for _, s := range m.Shifts {
				s.Locked = lock
			}
			if lock {
				notice = tr(L, "n_lock_month", monthLabel(L, req.Month))
			} else {
				notice = tr(L, "n_unlock_month", monthLabel(L, req.Month))
			}
			return nil
		}

		s := m.find(req.ShiftID)
		if s == nil {
			return userErr("e_shift_gone")
		}
		switch req.Action {
		case "take":
			if s.Locked {
				return userErr("e_locked")
			}
			if contains(s.Users, userID) {
				return userErr("e_already_have")
			}
			if len(s.Users) >= s.Places {
				return userErr("e_full")
			}
			s.Users = append(s.Users, userID)
			notice = tr(L, "n_take", tr(L, "what", shiftLabel(s), fmtDate(L, s.Date)))
		case "release":
			if s.Locked {
				return userErr("e_locked_ask_admin")
			}
			if !contains(s.Users, userID) {
				return userErr("e_not_yours")
			}
			s.Users = without(s.Users, userID)
			s.dropOffer(userID)
			notice = tr(L, "n_release", tr(L, "what", shiftLabel(s), fmtDate(L, s.Date)))
		case "assign":
			if contains(s.Users, req.UserID) {
				return userErr("e_already_on")
			}
			if len(s.Users) >= s.Places {
				return userErr("e_full")
			}
			s.Users = append(s.Users, req.UserID)
		case "remove":
			s.Users = without(s.Users, req.UserID)
			s.dropOffer(req.UserID)
		case "offer", "cancel_offer", "claim", "approve", "refuse":
			n, err := p.swapAction(req, s, userID, admin)
			if err != nil {
				return err
			}
			notice = n
		case "lock":
			s.Locked = true
		case "unlock":
			s.Locked = false
		case "update":
			if req.Places < len(s.Users) {
				return userErr("e_too_many")
			}
			s.Label, s.Start, s.End, s.Places = req.Label, req.Start, req.End, req.Places
		case "delete":
			out := m.Shifts[:0]
			for _, x := range m.Shifts {
				if x.ID != s.ID {
					out = append(out, x)
				}
			}
			m.Shifts = out
		default:
			return userErr("e_unknown_action")
		}
		return nil
	})
	return notice, err
}

func main() {
	plugin.ClientMain(&Plugin{})
}
