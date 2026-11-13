// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

// Translations. English is the default and the fallback.
// Messages shown to one person follow their Mattermost language;
// bot messages posted in a channel follow the server's default language.

import (
	"fmt"
	"strings"
	"time"
)

var supportedLangs = map[string]bool{"en": true, "fr": true, "de": true, "it": true}

var catalog = map[string]map[string]string{
	// ----- slash command -----
	"cmd_desc":            {"en": "Open this channel's shift planning", "fr": "Ouvrir le planning des shifts de ce canal", "de": "Schichtplan dieses Kanals öffnen", "it": "Apri il planning dei turni di questo canale"},
	"cmd_admin_only":      {"en": "Only a channel admin can do that.", "fr": "Seul un admin du canal peut faire ça.", "de": "Nur ein Kanal-Admin kann das tun.", "it": "Solo un admin del canale può farlo."},
	"cmd_enable_first":    {"en": "Planning is not enabled in this channel. Type `/planning enable` first.", "fr": "Le planning n'est pas activé sur ce canal. Tapez d'abord `/planning activer`.", "de": "Der Schichtplan ist in diesem Kanal nicht aktiviert. Tippen Sie zuerst `/planning aktivieren`.", "it": "Il planning non è attivo in questo canale. Digita prima `/planning attiva`."},
	"cmd_button_reposted": {"en": "The “Open planning” message has been posted and pinned again.", "fr": "Le message « Ouvrir le planning » a été reposté et épinglé.", "de": "Die Nachricht „Schichtplan öffnen“ wurde erneut gepostet und angeheftet.", "it": "Il messaggio «Apri il planning» è stato ripubblicato e fissato."},
	"cmd_toggle_admin":    {"en": "Only a channel admin can enable or disable the planning.", "fr": "Seul un admin du canal peut activer ou désactiver le planning.", "de": "Nur ein Kanal-Admin kann den Schichtplan aktivieren oder deaktivieren.", "it": "Solo un admin del canale può attivare o disattivare il planning."},
	"cmd_error":           {"en": "Error: %s", "fr": "Erreur : %s", "de": "Fehler: %s", "it": "Errore: %s"},
	"cmd_enabled":         {"en": "📅 Planning enabled in this channel.", "fr": "📅 Planning activé sur ce canal.", "de": "📅 Schichtplan in diesem Kanal aktiviert.", "it": "📅 Planning attivato in questo canale."},
	"cmd_disabled":        {"en": "Planning disabled in this channel (shifts are kept).", "fr": "Planning désactivé sur ce canal (les shifts sont conservés).", "de": "Schichtplan in diesem Kanal deaktiviert (die Schichten bleiben erhalten).", "it": "Planning disattivato in questo canale (i turni vengono conservati)."},
	"cmd_not_enabled":     {"en": "Planning is not enabled in this channel. An admin can type `/planning enable`.", "fr": "Le planning n'est pas activé sur ce canal. Un admin peut taper `/planning activer`.", "de": "Der Schichtplan ist in diesem Kanal nicht aktiviert. Ein Admin kann `/planning aktivieren` tippen.", "it": "Il planning non è attivo in questo canale. Un admin può digitare `/planning attiva`."},

	// ----- channel posts -----
	"post_disabled":   {"en": "The planning has been disabled in this channel.", "fr": "Le planning a été désactivé sur ce canal.", "de": "Der Schichtplan wurde in diesem Kanal deaktiviert.", "it": "Il planning è stato disattivato in questo canale."},
	"post_pinned":     {"en": "📅 **Team planning**\nTap the button to see the shifts and take one. On a computer you can also use the calendar icon at the top right.", "fr": "📅 **Planning de l'équipe**\nTouchez le bouton pour voir les shifts et en prendre un. Sur ordinateur, vous pouvez aussi utiliser l'icône calendrier en haut à droite.", "de": "📅 **Schichtplan des Teams**\nTippen Sie auf die Schaltfläche, um die Schichten zu sehen und eine zu übernehmen. Am Computer können Sie auch das Kalendersymbol oben rechts verwenden.", "it": "📅 **Planning del team**\nTocca il pulsante per vedere i turni e prenderne uno. Dal computer puoi usare anche l'icona del calendario in alto a destra."},
	"btn_open":        {"en": "📅 Open planning", "fr": "📅 Ouvrir le planning", "de": "📅 Schichtplan öffnen", "it": "📅 Apri il planning"},
	"what":            {"en": "**%s** on %s", "fr": "**%s** du %s", "de": "**%s** am %s", "it": "**%s** di %s"},
	"n_take":          {"en": "✅ {user} took the shift %s.", "fr": "✅ {user} a pris le shift %s.", "de": "✅ {user} hat die Schicht %s übernommen.", "it": "✅ {user} ha preso il turno %s."},
	"n_release":       {"en": "↩️ {user} released the shift %s.", "fr": "↩️ {user} a libéré le shift %s.", "de": "↩️ {user} hat die Schicht %s freigegeben.", "it": "↩️ {user} ha liberato il turno %s."},
	"n_lock_month":    {"en": "🔒 The shifts of **%s** are locked.", "fr": "🔒 Les shifts de **%s** sont verrouillés.", "de": "🔒 Die Schichten im **%s** sind gesperrt.", "it": "🔒 I turni di **%s** sono bloccati."},
	"n_unlock_month":  {"en": "🔓 The shifts of **%s** are open again.", "fr": "🔓 Les shifts de **%s** sont rouverts.", "de": "🔓 Die Schichten im **%s** sind wieder offen.", "it": "🔓 I turni di **%s** sono di nuovo aperti."},
	"n_offer":         {"en": "🔄 {user} is offering their shift %s. Can someone take it?", "fr": "🔄 {user} propose son shift %s. Quelqu'un peut le reprendre ?", "de": "🔄 {user} bietet die Schicht %s an. Kann jemand übernehmen?", "it": "🔄 {user} offre il suo turno %s. Qualcuno può prenderlo?"},
	"n_cancel_offer":  {"en": "{user} is keeping their shift %s after all.", "fr": "{user} garde finalement son shift %s.", "de": "{user} behält die Schicht %s nun doch.", "it": "{user} alla fine tiene il suo turno %s."},
	"n_claim_pending": {"en": "⏳ {user} wants to take over the shift %s from **%s**. A channel admin must approve.", "fr": "⏳ {user} veut reprendre le shift %s de **%s**. Un admin du canal doit valider.", "de": "⏳ {user} möchte die Schicht %s von **%s** übernehmen. Ein Kanal-Admin muss zustimmen.", "it": "⏳ {user} vuole prendere il turno %s di **%s**. Un admin del canale deve approvare."},
	"n_claim":         {"en": "🔁 {user} takes over the shift %s from **%s**.", "fr": "🔁 {user} reprend le shift %s de **%s**.", "de": "🔁 {user} übernimmt die Schicht %s von **%s**.", "it": "🔁 {user} prende il turno %s di **%s**."},
	"n_refused":       {"en": "❌ Swap refused: **%s** stays on the shift %s. The shift is still on offer.", "fr": "❌ Échange refusé : **%s** reste sur le shift %s. Le shift est toujours proposé.", "de": "❌ Tausch abgelehnt: **%s** bleibt auf der Schicht %s. Die Schicht wird weiterhin angeboten.", "it": "❌ Scambio rifiutato: **%s** resta sul turno %s. Il turno è ancora disponibile."},
	"n_approved":      {"en": "✅ Swap approved: **%s** replaces **%s** on the shift %s.", "fr": "✅ Échange validé : **%s** remplace **%s** sur le shift %s.", "de": "✅ Tausch genehmigt: **%s** ersetzt **%s** auf der Schicht %s.", "it": "✅ Scambio approvato: **%s** sostituisce **%s** sul turno %s."},
	"btn_claim_post":  {"en": "🔄 Take this shift", "fr": "🔄 Reprendre ce shift", "de": "🔄 Schicht übernehmen", "it": "🔄 Prendi questo turno"},
	"btn_approve":     {"en": "✅ Approve", "fr": "✅ Approuver", "de": "✅ Genehmigen", "it": "✅ Approva"},
	"btn_refuse":      {"en": "❌ Refuse", "fr": "❌ Refuser", "de": "❌ Ablehnen", "it": "❌ Rifiuta"},
	"handled":         {"en": "_✔️ Done_", "fr": "_✔️ Traité_", "de": "_✔️ Erledigt_", "it": "_✔️ Fatto_"},

	// ----- mobile list (per user) -----
	"list_title":       {"en": "📅 **Planning** from %s to %s", "fr": "📅 **Planning** du %s au %s", "de": "📅 **Schichtplan** vom %s bis %s", "it": "📅 **Planning** dal %s al %s"},
	"list_empty":       {"en": "_No shifts planned in these 7 days._", "fr": "_Aucun shift prévu sur ces 7 jours._", "de": "_Keine Schichten in diesen 7 Tagen._", "it": "_Nessun turno previsto in questi 7 giorni._"},
	"btn_prev":         {"en": "‹ Previous week", "fr": "‹ Semaine précédente", "de": "‹ Vorherige Woche", "it": "‹ Settimana precedente"},
	"btn_next":         {"en": "Next week ›", "fr": "Semaine suivante ›", "de": "Nächste Woche ›", "it": "Settimana successiva ›"},
	"btn_release":      {"en": "Release %s", "fr": "Libérer %s", "de": "%s freigeben", "it": "Libera %s"},
	"btn_take":         {"en": "Take %s", "fr": "Prendre %s", "de": "%s übernehmen", "it": "Prendi %s"},
	"btn_offer":        {"en": "🔄 Offer a swap", "fr": "🔄 Proposer un échange", "de": "🔄 Tausch anbieten", "it": "🔄 Proponi uno scambio"},
	"btn_cancel_offer": {"en": "Cancel my offer", "fr": "Annuler mon offre", "de": "Mein Angebot zurückziehen", "it": "Annulla la mia offerta"},
	"btn_claim_from":   {"en": "🔄 Take %s from %s", "fr": "🔄 Reprendre %s de %s", "de": "🔄 %s von %s übernehmen", "it": "🔄 Prendi %s di %s"},
	"tag_pending":      {"en": " (⏳ → %s)", "fr": " (⏳ → %s)", "de": " (⏳ → %s)", "it": " (⏳ → %s)"},
	"tag_offered":      {"en": " (🔄 up for grabs)", "fr": " (🔄 à reprendre)", "de": " (🔄 zu übernehmen)", "it": " (🔄 da prendere)"},
	"deleted_user":     {"en": "(deleted)", "fr": "(supprimé)", "de": "(gelöscht)", "it": "(eliminato)"},

	// ----- errors (per user) -----
	"e_not_logged_in":     {"en": "not logged in", "fr": "non connecté", "de": "nicht angemeldet", "it": "non connesso"},
	"e_bad_params":        {"en": "invalid parameters", "fr": "paramètres invalides", "de": "ungültige Parameter", "it": "parametri non validi"},
	"e_not_member":        {"en": "you are not a member of this channel", "fr": "vous n'êtes pas membre de ce canal", "de": "Sie sind kein Mitglied dieses Kanals", "it": "non sei membro di questo canale"},
	"e_bad_request":       {"en": "invalid request", "fr": "requête invalide", "de": "ungültige Anfrage", "it": "richiesta non valida"},
	"e_admin_only":        {"en": "reserved for channel admins", "fr": "réservé aux admins du canal", "de": "nur für Kanal-Admins", "it": "riservato agli admin del canale"},
	"e_not_enabled":       {"en": "the planning is not enabled in this channel", "fr": "le planning n'est pas activé sur ce canal", "de": "der Schichtplan ist in diesem Kanal nicht aktiviert", "it": "il planning non è attivo in questo canale"},
	"e_bad_start":         {"en": "invalid start time (HH:MM)", "fr": "heure de début invalide (HH:MM)", "de": "ungültige Startzeit (HH:MM)", "it": "ora di inizio non valida (HH:MM)"},
	"e_bad_end":           {"en": "invalid end time (HH:MM)", "fr": "heure de fin invalide (HH:MM)", "de": "ungültige Endzeit (HH:MM)", "it": "ora di fine non valida (HH:MM)"},
	"e_places_range":      {"en": "number of people must be between 1 and 10", "fr": "nombre de places entre 1 et 10", "de": "Anzahl Personen zwischen 1 und 10", "it": "numero di persone tra 1 e 10"},
	"e_dates_range":       {"en": "between 1 and 120 dates", "fr": "entre 1 et 120 dates", "de": "zwischen 1 und 120 Daten", "it": "tra 1 e 120 date"},
	"e_bad_date":          {"en": "invalid date", "fr": "date invalide", "de": "ungültiges Datum", "it": "data non valida"},
	"e_not_in_channel":    {"en": "this person is not in the channel", "fr": "cette personne n'est pas dans le canal", "de": "diese Person ist nicht im Kanal", "it": "questa persona non è nel canale"},
	"e_shift_gone":        {"en": "this shift no longer exists", "fr": "ce shift n'existe plus", "de": "diese Schicht existiert nicht mehr", "it": "questo turno non esiste più"},
	"e_locked":            {"en": "this shift is locked", "fr": "ce shift est verrouillé", "de": "diese Schicht ist gesperrt", "it": "questo turno è bloccato"},
	"e_locked_ask_admin":  {"en": "this shift is locked, ask an admin (or offer a swap)", "fr": "ce shift est verrouillé, demandez à un admin (ou proposez un échange)", "de": "diese Schicht ist gesperrt, fragen Sie einen Admin (oder bieten Sie einen Tausch an)", "it": "questo turno è bloccato, chiedi a un admin (o proponi uno scambio)"},
	"e_already_have":      {"en": "you already have this shift", "fr": "vous avez déjà ce shift", "de": "Sie haben diese Schicht bereits", "it": "hai già questo turno"},
	"e_full":              {"en": "this shift is full", "fr": "ce shift est complet", "de": "diese Schicht ist voll", "it": "questo turno è al completo"},
	"e_not_yours":         {"en": "you don't have this shift", "fr": "vous n'avez pas ce shift", "de": "Sie haben diese Schicht nicht", "it": "non hai questo turno"},
	"e_already_on":        {"en": "already on this shift", "fr": "déjà inscrit", "de": "bereits eingetragen", "it": "già iscritto"},
	"e_too_many":          {"en": "more people signed up than places", "fr": "plus de personnes inscrites que de places", "de": "mehr eingetragene Personen als Plätze", "it": "più persone iscritte che posti"},
	"e_unknown_action":    {"en": "unknown action", "fr": "action inconnue", "de": "unbekannte Aktion", "it": "azione sconosciuta"},
	"e_swaps_off":         {"en": "shift swaps are turned off", "fr": "les échanges de shifts sont désactivés", "de": "Schichttausch ist deaktiviert", "it": "gli scambi di turni sono disattivati"},
	"e_past":              {"en": "this shift is already over", "fr": "ce shift est déjà passé", "de": "diese Schicht ist bereits vorbei", "it": "questo turno è già passato"},
	"e_already_offered":   {"en": "you are already offering this shift", "fr": "vous proposez déjà ce shift", "de": "Sie bieten diese Schicht bereits an", "it": "stai già offrendo questo turno"},
	"e_no_offer":          {"en": "no offer in progress", "fr": "aucune offre en cours", "de": "kein laufendes Angebot", "it": "nessuna offerta in corso"},
	"e_not_offered":       {"en": "this shift is no longer on offer", "fr": "ce shift n'est plus proposé", "de": "diese Schicht wird nicht mehr angeboten", "it": "questo turno non è più disponibile"},
	"e_own_shift":         {"en": "this is your own shift", "fr": "c'est votre propre shift", "de": "das ist Ihre eigene Schicht", "it": "è il tuo turno"},
	"e_already_on_shift":  {"en": "you are already on this shift", "fr": "vous êtes déjà sur ce shift", "de": "Sie sind bereits auf dieser Schicht", "it": "sei già su questo turno"},
	"e_pending_exists":    {"en": "someone is already waiting for approval on this shift", "fr": "quelqu'un attend déjà la validation pour ce shift", "de": "jemand wartet bereits auf die Genehmigung für diese Schicht", "it": "qualcuno attende già l'approvazione per questo turno"},
	"e_approve_admin":     {"en": "only a channel admin can approve a swap", "fr": "seul un admin du canal peut valider un échange", "de": "nur ein Kanal-Admin kann einen Tausch genehmigen", "it": "solo un admin del canale può approvare uno scambio"},
	"e_no_pending":        {"en": "no pending request for this shift", "fr": "aucune demande en attente pour ce shift", "de": "keine offene Anfrage für diese Schicht", "it": "nessuna richiesta in sospeso per questo turno"},
	"e_person_already_on": {"en": "this person is already on this shift", "fr": "cette personne est déjà sur ce shift", "de": "diese Person ist bereits auf dieser Schicht", "it": "questa persona è già su questo turno"},
	"e_conflict":          {"en": "write conflict, please try again", "fr": "conflit d'écriture, réessayez", "de": "Schreibkonflikt, bitte erneut versuchen", "it": "conflitto di scrittura, riprova"},
}

var weekdays = map[string][]string{
	"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
	"fr": {"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
	"de": {"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
	"it": {"domenica", "lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato"},
}

var months = map[string][]string{
	"en": {"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	"fr": {"", "janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
	"de": {"", "Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
	"it": {"", "gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre"},
}

func normLang(l string) string {
	l = strings.ToLower(l)
	if len(l) > 2 {
		l = l[:2]
	}
	if supportedLangs[l] {
		return l
	}
	return "en"
}

func tr(lang, key string, args ...any) string {
	m, ok := catalog[key]
	if !ok {
		return key
	}
	s, ok := m[normLang(lang)]
	if !ok {
		s = m["en"]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// serverLang is the language of bot messages posted in channels.
func (p *Plugin) serverLang() string {
	if c := p.API.GetConfig(); c != nil && c.LocalizationSettings.DefaultServerLocale != nil {
		return normLang(*c.LocalizationSettings.DefaultServerLocale)
	}
	return "en"
}

// userLang is the language of one person (their Mattermost setting).
func (p *Plugin) userLang(userID string) string {
	if u, err := p.API.GetUser(userID); err == nil && u.Locale != "" {
		return normLang(u.Locale)
	}
	return p.serverLang()
}

// fmtDate: "Thursday, December 10" / "jeudi 10 décembre" / "Donnerstag, 10. Dezember" / "giovedì 10 dicembre".
func fmtDate(lang, date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	lang = normLang(lang)
	wd, mo := weekdays[lang][t.Weekday()], months[lang][t.Month()]
	switch lang {
	case "en":
		return fmt.Sprintf("%s, %s %d", wd, mo, t.Day())
	case "de":
		return fmt.Sprintf("%s, %d. %s", wd, t.Day(), mo)
	default:
		return fmt.Sprintf("%s %d %s", wd, t.Day(), mo)
	}
}

// shortDate: "December 10" / "10 décembre" / "10. Dezember" / "10 dicembre".
func shortDate(lang, date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	lang = normLang(lang)
	mo := months[lang][t.Month()]
	switch lang {
	case "en":
		return fmt.Sprintf("%s %d", mo, t.Day())
	case "de":
		return fmt.Sprintf("%d. %s", t.Day(), mo)
	default:
		return fmt.Sprintf("%d %s", t.Day(), mo)
	}
}

func monthLabel(lang, month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return fmt.Sprintf("%s %d", months[normLang(lang)][t.Month()], t.Year())
}

// errText translates an error for one person.
func (p *Plugin) errText(userID string, err error) string {
	if ue, ok := err.(userErr); ok {
		return tr(p.userLang(userID), string(ue))
	}
	return err.Error()
}
