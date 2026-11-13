// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

// Shift swaps: a member offers their place, a colleague takes it over,
// optionally after an admin approves (plugin setting SwapApproval).

import (
	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) nameOf(id string) string {
	if u, err := p.API.GetUser(id); err == nil {
		return displayName(u)
	}
	return "?"
}

func swapButton(name, style, action string, s *Shift, from string) *model.PostAction {
	return &model.PostAction{
		Id:    "pl" + action,
		Type:  model.PostActionTypeButton,
		Name:  name,
		Style: style,
		Integration: &model.PostActionIntegration{
			URL: "/plugins/com.shiftplugins.planning/interactive",
			Context: map[string]any{
				"action": action, "shift_id": s.ID, "month": s.Date[:7], "from_user": from, "src": "post",
			},
		},
	}
}

// swapAction runs inside updateMonth; it mutates s and returns the channel notice.
func (p *Plugin) swapAction(req *actionReq, s *Shift, userID string, admin bool) (string, error) {
	cfg := p.config()
	if !cfg.AllowSwaps {
		return "", userErr("e_swaps_off")
	}
	L := p.serverLang()
	what := tr(L, "what", shiftLabel(s), fmtDate(L, s.Date))
	switch req.Action {
	case "offer":
		if s.Date < today() {
			return "", userErr("e_past")
		}
		if !contains(s.Users, userID) {
			return "", userErr("e_not_yours")
		}
		if s.offerOf(userID) != nil {
			return "", userErr("e_already_offered")
		}
		s.Offers = append(s.Offers, &Offer{From: userID})
		req.buttons = []*model.PostAction{swapButton(tr(L, "btn_claim_post"), "primary", "claim", s, userID)}
		return tr(L, "n_offer", what), nil

	case "cancel_offer":
		o := s.offerOf(userID)
		if o == nil {
			return "", userErr("e_no_offer")
		}
		s.dropOffer(userID)
		return tr(L, "n_cancel_offer", what), nil

	case "claim":
		o := s.offerOf(req.FromUser)
		if o == nil || !contains(s.Users, req.FromUser) {
			return "", userErr("e_not_offered")
		}
		if req.FromUser == userID {
			return "", userErr("e_own_shift")
		}
		if contains(s.Users, userID) {
			return "", userErr("e_already_on_shift")
		}
		if o.To != "" {
			return "", userErr("e_pending_exists")
		}
		if cfg.SwapApproval {
			o.To = userID
			req.buttons = []*model.PostAction{
				swapButton(tr(L, "btn_approve"), "good", "approve", s, req.FromUser),
				swapButton(tr(L, "btn_refuse"), "danger", "refuse", s, req.FromUser),
			}
			return tr(L, "n_claim_pending", what, p.nameOf(req.FromUser)), nil
		}
		s.Users = append(without(s.Users, req.FromUser), userID)
		s.dropOffer(req.FromUser)
		return tr(L, "n_claim", what, p.nameOf(req.FromUser)), nil

	case "approve", "refuse":
		if !admin {
			return "", userErr("e_approve_admin")
		}
		o := s.offerOf(req.FromUser)
		if o == nil || o.To == "" {
			return "", userErr("e_no_pending")
		}
		to := o.To
		if req.Action == "refuse" {
			o.To = ""
			req.buttons = []*model.PostAction{swapButton(tr(L, "btn_claim_post"), "primary", "claim", s, req.FromUser)}
			return tr(L, "n_refused", p.nameOf(req.FromUser), what), nil
		}
		if contains(s.Users, to) {
			s.dropOffer(req.FromUser)
			return "", userErr("e_person_already_on")
		}
		s.Users = append(without(s.Users, req.FromUser), to)
		s.dropOffer(req.FromUser)
		return tr(L, "n_approved", p.nameOf(to), p.nameOf(req.FromUser), what), nil
	}
	return "", userErr("e_unknown_action")
}
