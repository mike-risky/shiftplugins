// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// ---------- deciders (cached per call) ----------

type deciders struct {
	mgrs, admins []string
}

func (p *Plugin) loadDeciders() *deciders {
	return &deciders{mgrs: p.managers(), admins: p.sysadmins()}
}

func (d *deciders) approversFor(requesterID string) []string {
	if m := without(d.mgrs, requesterID); len(m) > 0 {
		return m
	}
	if a := without(d.admins, requesterID); len(a) > 0 {
		return a
	}
	return []string{requesterID}
}

func (d *deciders) canDecide(userID, requesterID string) bool {
	return contains(d.approversFor(requesterID), userID) || (userID != requesterID && contains(d.admins, userID))
}

func (d *deciders) isManager(userID string) bool {
	return contains(d.mgrs, userID) || contains(d.admins, userID)
}

// ---------- buttons & rendering ----------

func button(n *int, name, style string, ctx map[string]any) *model.PostAction {
	*n++
	return &model.PostAction{
		Id:    fmt.Sprintf("cg%d", *n),
		Type:  model.PostActionTypeButton,
		Name:  name,
		Style: style,
		Integration: &model.PostActionIntegration{
			URL:     interactPath,
			Context: ctx,
		},
	}
}

func cancellable(status, end string) bool {
	return status == StatusPending || (status == StatusApproved && end >= today())
}

func statusColor(status string) string {
	switch status {
	case StatusApproved:
		return colorApproved
	case StatusRefused:
		return colorRefused
	case StatusCancelled:
		return colorCancelled
	}
	return colorPending
}

func statusShort(lang, status string) string { return tr(lang, "st."+status) }

func (p *Plugin) statusLine(lang string, r *Request) string {
	switch r.Status {
	case StatusApproved, StatusRefused:
		s := tr(lang, "st.decided", statusShort(lang, r.Status), p.name(r.DeciderID), millisDate(lang, r.DecidedAt))
		if r.Comment != "" {
			s += "\n> " + strings.ReplaceAll(r.Comment, "\n", "\n> ")
		}
		return s
	case StatusCancelled:
		return tr(lang, "st.cancelledBy", p.name(r.UserID), millisDate(lang, r.DecidedAt))
	}
	return tr(lang, "st.waiting")
}

// attachment renders a request. forManager: titled with the person's name and,
// while pending, Approve / Decline buttons. Otherwise: the requester's view with Cancel.
func (p *Plugin) attachment(lang string, r *Request, forManager bool, n *int, extra map[string]any) *model.MessageAttachment {
	title := "🌴 " + r.Type
	if forManager {
		title = "🌴 " + p.name(r.UserID) + " · " + r.Type
	}
	text := fmt.Sprintf("**%s** · %s", capitalize(rangeLabel(lang, r.Start, r.End, r.Part)), daysLabel(lang, r.Days))
	if r.Note != "" {
		text += "\n" + tr(lang, "note") + strings.ReplaceAll(r.Note, "\n", " ")
	}
	text += "\n" + p.statusLine(lang, r)
	ctx := func(action string) map[string]any {
		c := map[string]any{"action": action, "id": r.ID}
		for k, v := range extra {
			c[k] = v
		}
		return c
	}
	var actions []*model.PostAction
	if forManager && r.Status == StatusPending {
		actions = append(actions,
			button(n, tr(lang, "btn.approve"), "success", ctx("approve")),
			button(n, tr(lang, "btn.refuse"), "danger", ctx("refuse")))
	}
	if !forManager && cancellable(r.Status, r.End) {
		actions = append(actions, button(n, tr(lang, "btn.cancel"), "default", ctx("cancel")))
	}
	return &model.MessageAttachment{
		Color:    statusColor(r.Status),
		Title:    title,
		Text:     text,
		Actions:  actions,
		Fallback: title,
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// ---------- dialogs ----------

func (p *Plugin) openRequestDialog(userID, triggerID string) {
	lang := p.lang(userID)
	types := p.types()
	opts := make([]*model.PostActionOptions, 0, len(types))
	for _, t := range types {
		opts = append(opts, &model.PostActionOptions{Text: t, Value: t})
	}
	weekendNote := tr(lang, "dlg.noWeekends")
	if p.config().CountWeekends {
		weekendNote = tr(lang, "dlg.weekends")
	}
	err := p.API.OpenInteractiveDialog(model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       dialogPath,
		Dialog: model.Dialog{
			CallbackId:       "new",
			Title:            tr(lang, "dlg.title"),
			IntroductionText: tr(lang, "dlg.intro") + " " + weekendNote,
			SubmitLabel:      tr(lang, "dlg.submit"),
			Elements: []model.DialogElement{
				{DisplayName: tr(lang, "dlg.type"), Name: "type", Type: "select", Options: opts, Default: types[0]},
				{DisplayName: tr(lang, "dlg.start"), Name: "start", Type: "text", Placeholder: tr(lang, "dlg.datePH"), HelpText: tr(lang, "dlg.startHelp"), MaxLength: 10},
				{DisplayName: tr(lang, "dlg.end"), Name: "end", Type: "text", Placeholder: tr(lang, "dlg.datePH"), HelpText: tr(lang, "dlg.endHelp"), Optional: true, MaxLength: 10},
				{DisplayName: tr(lang, "dlg.part"), Name: "part", Type: "select", Default: PartFull, Options: []*model.PostActionOptions{
					{Text: tr(lang, "part.full"), Value: PartFull},
					{Text: tr(lang, "part.am"), Value: PartAM},
					{Text: tr(lang, "part.pm"), Value: PartPM},
				}},
				{DisplayName: tr(lang, "dlg.note"), Name: "note", Type: "textarea", Optional: true, MaxLength: maxNoteLen, Placeholder: tr(lang, "dlg.optional")},
			},
		},
	})
	if err != nil {
		p.API.LogWarn("leave: open dialog", "err", err.Error())
	}
}

func (p *Plugin) openRefuseDialog(userID, triggerID, state string) {
	lang := p.lang(userID)
	err := p.API.OpenInteractiveDialog(model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       dialogPath,
		Dialog: model.Dialog{
			CallbackId:  "refuse",
			Title:       tr(lang, "ref.title"),
			SubmitLabel: tr(lang, "ref.submit"),
			State:       state,
			Elements: []model.DialogElement{
				{DisplayName: tr(lang, "ref.reason"), Name: "comment", Type: "textarea", Optional: true, MaxLength: maxNoteLen, Placeholder: tr(lang, "ref.reasonPH")},
			},
		},
	})
	if err != nil {
		p.API.LogWarn("leave: open decline dialog", "err", err.Error())
	}
}

func (p *Plugin) handleDialog(w http.ResponseWriter, r *http.Request, userID string) {
	var req model.SubmitDialogRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Cancelled {
		writeJSON(w, &model.SubmitDialogResponse{})
		return
	}
	lang := p.lang(userID)
	str := func(k string) string { v, _ := req.Submission[k].(string); return strings.TrimSpace(v) }
	switch req.CallbackId {
	case "new":
		errs, err := p.submitRequest(userID, req.ChannelId, str("type"), str("start"), str("end"), str("part"), str("note"))
		if len(errs) > 0 {
			writeJSON(w, &model.SubmitDialogResponse{Errors: errs})
			return
		}
		if err != nil {
			writeJSON(w, &model.SubmitDialogResponse{Error: errText(lang, err)})
			return
		}
	case "refuse":
		parts := strings.Split(req.State, "|")
		for len(parts) < 4 {
			parts = append(parts, "")
		}
		id, postID, channelID, view := parts[0], parts[1], parts[2], parts[3]
		comment := str("comment")
		if len([]rune(comment)) > maxNoteLen {
			comment = string([]rune(comment)[:maxNoteLen])
		}
		if _, err := p.decide(id, userID, false, comment); err != nil {
			writeJSON(w, &model.SubmitDialogResponse{Error: errText(lang, err)})
			return
		}
		if postID != "" && view != "" {
			p.updateView(userID, channelID, postID, view)
		}
	default:
		writeJSON(w, &model.SubmitDialogResponse{Error: tr(lang, "err.unknown")})
		return
	}
	writeJSON(w, &model.SubmitDialogResponse{})
}

// submitRequest validates and stores a new request, then notifies the approvers.
// It returns per-field errors for the dialog.
func (p *Plugin) submitRequest(userID, channelID, typ, startS, endS, part, note string) (map[string]string, error) {
	cfg := p.config()
	lang := p.lang(userID)
	errs := map[string]string{}
	if !contains(p.types(), typ) {
		errs["type"] = tr(lang, "err.type")
	}
	start, ok := parseDate(startS)
	if !ok {
		errs["start"] = tr(lang, "err.date", "15.12.2026")
	}
	end := start
	if endS != "" {
		if end, ok = parseDate(endS); !ok {
			errs["end"] = tr(lang, "err.date", "19.12.2026")
		}
	}
	if len(errs) > 0 {
		return errs, nil
	}
	if end < start {
		errs["end"] = tr(lang, "err.endBefore")
		return errs, nil
	}
	if s, _ := parseISO(start); s.AddDate(0, 0, maxRangeDays).Format(isoFmt) <= end {
		errs["end"] = tr(lang, "err.tooLong")
		return errs, nil
	}
	if start != end || (part != PartAM && part != PartPM) {
		part = PartFull
	}
	days := countDays(start, end, part, cfg.CountWeekends)
	if days == 0 {
		errs["start"] = tr(lang, "err.weekendOnly")
		return errs, nil
	}
	if o := p.overlapping(userID, start, end); o != nil {
		errs["start"] = tr(lang, "err.overlap", tr(lang, "w."+o.Status), rangeLabel(lang, o.Start, o.End, PartFull))
		return errs, nil
	}
	if len([]rune(note)) > maxNoteLen {
		note = string([]rune(note)[:maxNoteLen])
	}

	r := &Request{
		ID: model.NewId(), UserID: userID, Type: typ, Start: start, End: end, Part: part,
		Days: days, Note: note, Status: StatusPending, CreatedAt: model.GetMillis(),
	}
	if err := p.createRequest(r); err != nil {
		p.API.LogError("leave: create", "err", err.Error())
		return nil, err
	}

	d := p.loadDeciders()
	approvers := d.approversFor(userID)
	managerPosts := map[string]string{}
	requesterName := p.name(userID)
	var names []string
	for _, a := range approvers {
		names = append(names, p.name(a))
		al := p.lang(a)
		msg := tr(al, "msg.newReq", requesterName)
		if a == userID {
			msg = tr(al, "msg.selfReq")
		}
		if id := p.dmPost(a, msg, p.attachment(al, r, true, new(int), nil)); id != "" {
			managerPosts[a] = id
		}
	}
	reqPost := p.dmPost(userID, tr(lang, "msg.sentTo", strings.Join(names, ", ")), p.attachment(lang, r, false, new(int), nil))
	if _, err := p.updateRequest(r.ID, func(x *Request) error {
		x.ManagerPosts = managerPosts
		x.RequesterPost = reqPost
		return nil
	}); err != nil {
		p.API.LogError("leave: save posts", "err", err.Error())
	}
	if channelID != "" {
		dm, _ := p.API.GetDirectChannel(userID, p.botID)
		if dm == nil || dm.Id != channelID {
			p.API.SendEphemeralPost(userID, &model.Post{
				UserId: p.botID, ChannelId: channelID,
				Message: tr(lang, "msg.sentEph", r.Type, rangeLabel(lang, start, end, part), botUsername),
			})
		}
	}
	return nil, nil
}

// dmPost posts a bot message in the DM with userID and returns its ID.
func (p *Plugin) dmPost(userID, msg string, att *model.MessageAttachment) string {
	ch, appErr := p.API.GetDirectChannel(userID, p.botID)
	if appErr != nil {
		p.API.LogError("leave: dm channel", "err", appErr.Error())
		return ""
	}
	post := &model.Post{UserId: p.botID, ChannelId: ch.Id, Message: msg}
	if att != nil {
		model.ParseMessageAttachment(post, []*model.MessageAttachment{att})
	}
	created, appErr := p.API.CreatePost(post)
	if appErr != nil {
		p.API.LogError("leave: dm post", "err", appErr.Error())
		return ""
	}
	return created.Id
}

// refreshPosts re-renders every bot post that shows this request.
func (p *Plugin) refreshPosts(r *Request) {
	update := func(postID, viewerID string, forManager bool) {
		if postID == "" {
			return
		}
		post, err := p.API.GetPost(postID)
		if err != nil || post.DeleteAt != 0 {
			return
		}
		model.ParseMessageAttachment(post, []*model.MessageAttachment{p.attachment(p.lang(viewerID), r, forManager, new(int), nil)})
		if _, err := p.API.UpdatePost(post); err != nil {
			p.API.LogWarn("leave: update post", "err", err.Error())
		}
	}
	for m, id := range r.ManagerPosts {
		update(id, m, true)
	}
	update(r.RequesterPost, r.UserID, false)
}

func (p *Plugin) decide(id, deciderID string, approve bool, comment string) (*Request, error) {
	cur, err := p.getRequest(id)
	if err != nil {
		return nil, err
	}
	if !p.loadDeciders().canDecide(deciderID, cur.UserID) {
		return nil, userErr{key: "err.cannotDecide"}
	}
	r, err := p.updateRequest(id, func(x *Request) error {
		if x.Status != StatusPending {
			return userErr{key: "err.alreadyDone", status: x.Status}
		}
		x.Status = StatusRefused
		if approve {
			x.Status = StatusApproved
		}
		x.DeciderID = deciderID
		x.DecidedAt = model.GetMillis()
		x.Comment = comment
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.refreshPosts(r)
	if deciderID != r.UserID {
		rl := p.lang(r.UserID)
		key := "msg.refused"
		if approve {
			key = "msg.approved"
		}
		msg := tr(rl, key, p.name(deciderID), r.Type, rangeLabel(rl, r.Start, r.End, r.Part))
		if comment != "" {
			msg += "\n> " + strings.ReplaceAll(comment, "\n", "\n> ")
		}
		p.dmPost(r.UserID, msg, nil)
	}
	return r, nil
}

func (p *Plugin) cancel(id, userID string) (*Request, error) {
	var wasApproved bool
	r, err := p.updateRequest(id, func(x *Request) error {
		if x.UserID != userID {
			return userErr{key: "err.onlyOwner"}
		}
		if !cancellable(x.Status, x.End) {
			return userErr{key: "err.notCancellable"}
		}
		wasApproved = x.Status == StatusApproved
		x.Status = StatusCancelled
		x.DecidedAt = model.GetMillis()
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.refreshPosts(r)
	if wasApproved {
		for m := range r.ManagerPosts {
			if m != userID {
				ml := p.lang(m)
				p.dmPost(m, tr(ml, "msg.cancelledApproved", p.name(userID), r.Type, rangeLabel(ml, r.Start, r.End, r.Part)), nil)
			}
		}
	}
	return r, nil
}

// ---------- interactive buttons ----------

func (p *Plugin) handleInteractive(w http.ResponseWriter, r *http.Request, userID string) {
	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	lang := p.lang(userID)
	reply := func(text string) {
		writeJSON(w, &model.PostActionIntegrationResponse{EphemeralText: text})
	}
	str := func(k string) string { v, _ := req.Context[k].(string); return v }
	eph := str("eph") // view name when the button sits in an ephemeral view
	refresh := func() {
		if eph != "" {
			p.updateView(userID, req.ChannelId, req.PostId, eph)
		}
	}

	switch str("action") {
	case "new":
		p.openRequestDialog(userID, req.TriggerId)
	case "view":
		v := str("view")
		if eph != "" {
			p.updateView(userID, req.ChannelId, req.PostId, v)
		} else {
			p.sendView(userID, req.ChannelId, v)
		}
	case "approve":
		if _, err := p.decide(str("id"), userID, true, ""); err != nil {
			reply("⚠️ " + errText(lang, err))
			refresh()
			return
		}
		refresh()
	case "refuse":
		cur, err := p.getRequest(str("id"))
		if err == nil && cur.Status != StatusPending {
			err = userErr{key: "err.alreadyDone", status: cur.Status}
		}
		if err == nil && !p.loadDeciders().canDecide(userID, cur.UserID) {
			err = userErr{key: "err.cannotDecide"}
		}
		if err != nil {
			reply("⚠️ " + errText(lang, err))
			refresh()
			return
		}
		state := cur.ID
		if eph != "" {
			state += "|" + req.PostId + "|" + req.ChannelId + "|" + eph
		}
		p.openRefuseDialog(userID, req.TriggerId, state)
	case "cancel":
		if _, err := p.cancel(str("id"), userID); err != nil {
			reply("⚠️ " + errText(lang, err))
			refresh()
			return
		}
		refresh()
	default:
		reply(tr(lang, "err.unknown"))
		return
	}
	writeJSON(w, &model.PostActionIntegrationResponse{})
}

// ---------- ephemeral views (menu, mine, queue, team) ----------

func (p *Plugin) sendView(userID, channelID, view string) {
	p.API.SendEphemeralPost(userID, p.buildView(userID, channelID, view))
}

func (p *Plugin) updateView(userID, channelID, postID, view string) {
	post := p.buildView(userID, channelID, view)
	post.Id = postID
	p.API.UpdateEphemeralPost(userID, post)
}

func (p *Plugin) buildView(userID, channelID, view string) *model.Post {
	d := p.loadDeciders()
	manager := d.isManager(userID)
	lang := p.lang(userID)
	n := 0
	var msg string
	var atts []*model.MessageAttachment
	nav := func(name, v, style string) *model.PostAction {
		return button(&n, name, style, map[string]any{"action": "view", "view": v, "eph": v})
	}
	pendingForMe := func() []IndexEntry {
		return p.filterIndex(func(e IndexEntry) bool {
			return e.Status == StatusPending && d.canDecide(userID, e.UserID)
		})
	}

	switch view {
	case "mine":
		msg = tr(lang, "v.mine")
		if b := p.balanceLine(lang, userID); b != "" {
			msg += "\n" + b
		}
		yearStart := today()[:4] + "-01-01"
		list := p.filterIndex(func(e IndexEntry) bool { return e.UserID == userID && e.End >= yearStart })
		// most recent first, at most 15
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
		if len(list) > 15 {
			list = list[:15]
		}
		for _, e := range list {
			if r, err := p.getRequest(e.ID); err == nil {
				atts = append(atts, p.attachment(lang, r, false, &n, map[string]any{"eph": "mine"}))
			}
		}
		if len(list) == 0 {
			atts = append(atts, &model.MessageAttachment{Text: tr(lang, "v.mineEmpty")})
		}
	case "queue":
		msg = tr(lang, "v.queue")
		list := pendingForMe()
		for _, e := range list {
			if r, err := p.getRequest(e.ID); err == nil {
				atts = append(atts, p.attachment(lang, r, true, &n, map[string]any{"eph": "queue"}))
			}
		}
		if len(list) == 0 {
			atts = append(atts, &model.MessageAttachment{Text: tr(lang, "v.queueEmpty")})
		}
	case "team":
		from, to := today(), addDays(today(), 42)
		msg = tr(lang, "v.team", fmtDate(lang, from), fmtDate(lang, to))
		list := p.filterIndex(func(e IndexEntry) bool {
			if e.End < from || e.Start > to {
				return false
			}
			return e.Status == StatusApproved || (manager && e.Status == StatusPending)
		})
		var lines []string
		for _, e := range list {
			line := fmt.Sprintf("- **%s** · %s · %s", p.name(e.UserID), rangeLabel(lang, e.Start, e.End, partOf(p, e)), daysLabel(lang, e.Days))
			if manager {
				line += " · " + e.Type
				if e.Status == StatusPending {
					line += " · " + tr(lang, "v.teamPend")
				}
			}
			lines = append(lines, line)
		}
		text := strings.Join(lines, "\n")
		if len(lines) == 0 {
			text = tr(lang, "v.teamEmpty")
		}
		atts = append(atts, &model.MessageAttachment{Text: text})
	default:
		view = "menu"
		msg = tr(lang, "v.menu")
	}

	row := []*model.PostAction{}
	if view != "menu" {
		row = append(row, nav(tr(lang, "nav.menu"), "menu", "default"))
	}
	row = append(row, button(&n, tr(lang, "nav.new"), "primary", map[string]any{"action": "new"}))
	if view != "mine" {
		row = append(row, nav(tr(lang, "nav.mine"), "mine", "default"))
	}
	if view != "team" {
		row = append(row, nav(tr(lang, "nav.team"), "team", "default"))
	}
	if manager && view != "queue" {
		label := tr(lang, "nav.queue")
		if c := len(pendingForMe()); c > 0 {
			label += fmt.Sprintf(" (%d)", c)
		}
		row = append(row, nav(label, "queue", "default"))
	}
	atts = append(atts, &model.MessageAttachment{Actions: row})

	post := &model.Post{UserId: p.botID, ChannelId: channelID, CreateAt: model.GetMillis(), Message: msg}
	model.ParseMessageAttachment(post, atts)
	return post
}

// partOf loads the half-day info only for single-day entries (the index doesn't keep it).
func partOf(p *Plugin, e IndexEntry) string {
	if e.Start != e.End {
		return PartFull
	}
	if r, err := p.getRequest(e.ID); err == nil {
		return r.Part
	}
	return PartFull
}

// balanceLine: "Vacation 2026: 12 days approved, 3 pending · 10 left of 25".
// The allowance applies to the first leave type in the settings.
func (p *Plugin) balanceLine(lang, userID string) string {
	cfg := p.config()
	if cfg.AnnualDays <= 0 {
		return ""
	}
	typ := p.types()[0]
	year := time.Now().In(loc()).Format("2006")
	var approved, pending float64
	for _, e := range p.loadIndex() {
		if e.UserID != userID || e.Type != typ || e.Start[:4] != year {
			continue
		}
		switch e.Status {
		case StatusApproved:
			approved += e.Days
		case StatusPending:
			pending += e.Days
		}
	}
	s := tr(lang, "bal.approved", typ, year, fmtNum(lang, approved))
	if pending > 0 {
		s += tr(lang, "bal.pending", fmtNum(lang, pending))
	}
	return s + tr(lang, "bal.left", fmtNum(lang, float64(cfg.AnnualDays)-approved), cfg.AnnualDays)
}
