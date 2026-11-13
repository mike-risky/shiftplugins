// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

import (
	"fmt"
	"strings"
)

// Supported languages. Each user sees the plugin in their own Mattermost language
// (Settings → Display → Language); anything else falls back to English.
var langs = []string{"en", "fr", "de", "it"}

func normLang(l string) string {
	l = strings.ToLower(l)
	if len(l) > 2 {
		l = l[:2]
	}
	for _, x := range langs {
		if x == l {
			return l
		}
	}
	return "en"
}

// serverLang is the server's default client language (used for things shared by
// everybody: slash command name, bot display name, default leave types).
func (p *Plugin) serverLang() string {
	if c := p.API.GetConfig(); c != nil && c.LocalizationSettings.DefaultClientLocale != nil {
		return normLang(*c.LocalizationSettings.DefaultClientLocale)
	}
	return "en"
}

func (p *Plugin) lang(userID string) string {
	if u, err := p.API.GetUser(userID); err == nil && u.Locale != "" {
		return normLang(u.Locale)
	}
	return p.serverLang()
}

// tr translates key into lang (fmt-style args), falling back to English.
func tr(lang, key string, args ...any) string {
	m, ok := catalog[key]
	if !ok {
		return key
	}
	s, ok := m[lang]
	if !ok || s == "" {
		s = m["en"]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

type tx map[string]string

var catalog = map[string]tx{
	// --- statuses ---
	"st.pending":     {"en": "⏳ Pending", "fr": "⏳ En attente", "de": "⏳ Ausstehend", "it": "⏳ In attesa"},
	"st.approved":    {"en": "✅ Approved", "fr": "✅ Approuvée", "de": "✅ Genehmigt", "it": "✅ Approvata"},
	"st.refused":     {"en": "❌ Declined", "fr": "❌ Refusée", "de": "❌ Abgelehnt", "it": "❌ Rifiutata"},
	"st.cancelled":   {"en": "🚫 Cancelled", "fr": "🚫 Annulée", "de": "🚫 Storniert", "it": "🚫 Annullata"},
	"w.pending":      {"en": "pending", "fr": "en attente", "de": "ausstehend", "it": "in attesa"},
	"w.approved":     {"en": "approved", "fr": "approuvée", "de": "genehmigt", "it": "approvata"},
	"w.refused":      {"en": "declined", "fr": "refusée", "de": "abgelehnt", "it": "rifiutata"},
	"w.cancelled":    {"en": "cancelled", "fr": "annulée", "de": "storniert", "it": "annullata"},
	"st.decided":     {"en": "%s by %s on %s", "fr": "%s par %s le %s", "de": "%s von %s am %s", "it": "%s da %s il %s"},
	"st.cancelledBy": {"en": "🚫 Cancelled by %s on %s", "fr": "🚫 Annulée par %s le %s", "de": "🚫 Storniert von %s am %s", "it": "🚫 Annullata da %s il %s"},
	"st.waiting":     {"en": "⏳ **Awaiting approval**", "fr": "⏳ **En attente de validation**", "de": "⏳ **Wartet auf Genehmigung**", "it": "⏳ **In attesa di approvazione**"},
	"note":           {"en": "_Note:_ ", "fr": "_Note :_ ", "de": "_Notiz:_ ", "it": "_Nota:_ "},
	"btn.approve":    {"en": "✅ Approve", "fr": "✅ Approuver", "de": "✅ Genehmigen", "it": "✅ Approva"},
	"btn.refuse":     {"en": "❌ Decline", "fr": "❌ Refuser", "de": "❌ Ablehnen", "it": "❌ Rifiuta"},
	"btn.cancel":     {"en": "Cancel request", "fr": "Annuler la demande", "de": "Antrag stornieren", "it": "Annulla richiesta"},

	// --- request dialog ---
	"dlg.title":      {"en": "Leave request", "fr": "Demande de congé", "de": "Abwesenheitsantrag", "it": "Richiesta di assenza"},
	"dlg.intro":      {"en": "Your manager receives the request and you get their answer by direct message.", "fr": "Votre responsable reçoit la demande et vous recevez sa réponse en message privé.", "de": "Ihre vorgesetzte Person erhält den Antrag, und Sie bekommen die Antwort per Direktnachricht.", "it": "Il tuo responsabile riceve la richiesta e tu ricevi la risposta in un messaggio diretto."},
	"dlg.noWeekends": {"en": "Saturdays and Sundays are not counted.", "fr": "Les samedis et dimanches ne sont pas comptés.", "de": "Samstage und Sonntage werden nicht gezählt.", "it": "Sabati e domeniche non vengono conteggiati."},
	"dlg.weekends":   {"en": "Every day is counted, weekends included.", "fr": "Tous les jours sont comptés, week-ends compris.", "de": "Alle Tage werden gezählt, auch Wochenenden.", "it": "Vengono conteggiati tutti i giorni, weekend compresi."},
	"dlg.submit":     {"en": "Send", "fr": "Envoyer", "de": "Senden", "it": "Invia"},
	"dlg.type":       {"en": "Type", "fr": "Type", "de": "Art", "it": "Tipo"},
	"dlg.start":      {"en": "First day", "fr": "Premier jour", "de": "Erster Tag", "it": "Primo giorno"},
	"dlg.datePH":     {"en": "DD.MM.YYYY", "fr": "JJ.MM.AAAA", "de": "TT.MM.JJJJ", "it": "GG.MM.AAAA"},
	"dlg.startHelp":  {"en": "E.g. 15.12.2026, 15/12/2026 or 15.12", "fr": "Par ex. 15.12.2026 ou 15.12", "de": "Z. B. 15.12.2026 oder 15.12", "it": "Es. 15.12.2026, 15/12/2026 o 15.12"},
	"dlg.end":        {"en": "Last day", "fr": "Dernier jour", "de": "Letzter Tag", "it": "Ultimo giorno"},
	"dlg.endHelp":    {"en": "Leave empty for a single day.", "fr": "Laissez vide pour un seul jour.", "de": "Für einen einzelnen Tag leer lassen.", "it": "Lascia vuoto per un solo giorno."},
	"dlg.part":       {"en": "Duration (single day)", "fr": "Durée (pour un seul jour)", "de": "Dauer (einzelner Tag)", "it": "Durata (un solo giorno)"},
	"part.full":      {"en": "Full day", "fr": "Journée complète", "de": "Ganzer Tag", "it": "Giornata intera"},
	"part.am":        {"en": "Morning", "fr": "Matin", "de": "Vormittag", "it": "Mattina"},
	"part.pm":        {"en": "Afternoon", "fr": "Après-midi", "de": "Nachmittag", "it": "Pomeriggio"},
	"lc.am":          {"en": "morning", "fr": "matin", "de": "Vormittag", "it": "mattina"},
	"lc.pm":          {"en": "afternoon", "fr": "après-midi", "de": "Nachmittag", "it": "pomeriggio"},
	"dlg.note":       {"en": "Note", "fr": "Note", "de": "Notiz", "it": "Nota"},
	"dlg.optional":   {"en": "Optional", "fr": "Facultatif", "de": "Optional", "it": "Facoltativo"},

	// --- decline dialog ---
	"ref.title":    {"en": "Decline request", "fr": "Refuser la demande", "de": "Antrag ablehnen", "it": "Rifiuta richiesta"},
	"ref.submit":   {"en": "Decline", "fr": "Refuser", "de": "Ablehnen", "it": "Rifiuta"},
	"ref.reason":   {"en": "Reason", "fr": "Motif", "de": "Begründung", "it": "Motivo"},
	"ref.reasonPH": {"en": "Optional, sent to the person", "fr": "Facultatif, envoyé à la personne", "de": "Optional, wird der Person gesendet", "it": "Facoltativo, inviato alla persona"},

	// --- errors ---
	"err.type":           {"en": "Choose a type.", "fr": "Choisissez un type.", "de": "Wählen Sie eine Art.", "it": "Scegli un tipo."},
	"err.date":           {"en": "Invalid date. Example: %s", "fr": "Date invalide. Exemple : %s", "de": "Ungültiges Datum. Beispiel: %s", "it": "Data non valida. Esempio: %s"},
	"err.endBefore":      {"en": "The last day is before the first day.", "fr": "Le dernier jour est avant le premier jour.", "de": "Der letzte Tag liegt vor dem ersten Tag.", "it": "L'ultimo giorno è prima del primo giorno."},
	"err.tooLong":        {"en": "A request cannot exceed one year.", "fr": "Une demande ne peut pas dépasser un an.", "de": "Ein Antrag darf ein Jahr nicht überschreiten.", "it": "Una richiesta non può superare un anno."},
	"err.weekendOnly":    {"en": "This period only contains weekends.", "fr": "Cette période ne contient que des week-ends.", "de": "Dieser Zeitraum enthält nur Wochenenden.", "it": "Questo periodo contiene solo weekend."},
	"err.overlap":        {"en": "You already have a request (%s) %s.", "fr": "Vous avez déjà une demande (%s) %s.", "de": "Sie haben bereits einen Antrag (%s) %s.", "it": "Hai già una richiesta (%s) %s."},
	"err.notFound":       {"en": "Request not found.", "fr": "Demande introuvable.", "de": "Antrag nicht gefunden.", "it": "Richiesta non trovata."},
	"err.cannotDecide":   {"en": "You cannot approve this request.", "fr": "Vous ne pouvez pas valider cette demande.", "de": "Sie können diesen Antrag nicht genehmigen.", "it": "Non puoi approvare questa richiesta."},
	"err.alreadyDone":    {"en": "This request has already been handled (%s).", "fr": "Cette demande a déjà été traitée (%s).", "de": "Dieser Antrag wurde bereits bearbeitet (%s).", "it": "Questa richiesta è già stata gestita (%s)."},
	"err.onlyOwner":      {"en": "Only the requester can cancel this request.", "fr": "Seule la personne concernée peut annuler sa demande.", "de": "Nur die antragstellende Person kann den Antrag stornieren.", "it": "Solo chi ha fatto la richiesta può annullarla."},
	"err.notCancellable": {"en": "This request can no longer be cancelled.", "fr": "Cette demande ne peut plus être annulée.", "de": "Dieser Antrag kann nicht mehr storniert werden.", "it": "Questa richiesta non può più essere annullata."},
	"err.busy":           {"en": "The server is busy, try again.", "fr": "Le serveur est occupé, réessayez.", "de": "Der Server ist ausgelastet, bitte erneut versuchen.", "it": "Il server è occupato, riprova."},
	"err.generic":        {"en": "Something went wrong, try again.", "fr": "Une erreur est survenue, réessayez.", "de": "Ein Fehler ist aufgetreten, bitte erneut versuchen.", "it": "Si è verificato un errore, riprova."},
	"err.unknown":        {"en": "Unknown action.", "fr": "Action inconnue.", "de": "Unbekannte Aktion.", "it": "Azione sconosciuta."},

	// --- messages ---
	"msg.newReq":            {"en": "📩 New leave request from **%s**", "fr": "📩 Nouvelle demande de congé de **%s**", "de": "📩 Neuer Abwesenheitsantrag von **%s**", "it": "📩 Nuova richiesta di assenza da **%s**"},
	"msg.selfReq":           {"en": "📩 Your request (no other manager is set up, so you can approve it yourself)", "fr": "📩 Votre demande (aucun autre responsable n'est configuré, vous pouvez la valider vous-même)", "de": "📩 Ihr Antrag (keine andere vorgesetzte Person ist eingerichtet, Sie können ihn selbst genehmigen)", "it": "📩 La tua richiesta (nessun altro responsabile è configurato, puoi approvarla tu)"},
	"msg.sentTo":            {"en": "📨 Your request was sent to %s.", "fr": "📨 Votre demande a été envoyée à %s.", "de": "📨 Ihr Antrag wurde an %s gesendet.", "it": "📨 La tua richiesta è stata inviata a %s."},
	"msg.sentEph":           {"en": "✅ Request sent (%s, %s). You'll get the answer by direct message from @%s.", "fr": "✅ Demande envoyée (%s, %s). Vous recevrez la réponse en message privé de @%s.", "de": "✅ Antrag gesendet (%s, %s). Sie erhalten die Antwort per Direktnachricht von @%s.", "it": "✅ Richiesta inviata (%s, %s). Riceverai la risposta in un messaggio diretto da @%s."},
	"msg.approved":          {"en": "✅ Your leave request was **approved** by %s: %s, %s.", "fr": "✅ Votre demande de congé a été **approuvée** par %s : %s, %s.", "de": "✅ Ihr Abwesenheitsantrag wurde von %s **genehmigt**: %s, %s.", "it": "✅ La tua richiesta di assenza è stata **approvata** da %s: %s, %s."},
	"msg.refused":           {"en": "❌ Your leave request was **declined** by %s: %s, %s.", "fr": "❌ Votre demande de congé a été **refusée** par %s : %s, %s.", "de": "❌ Ihr Abwesenheitsantrag wurde von %s **abgelehnt**: %s, %s.", "it": "❌ La tua richiesta di assenza è stata **rifiutata** da %s: %s, %s."},
	"msg.cancelledApproved": {"en": "🚫 **%s** cancelled an approved leave: %s, %s.", "fr": "🚫 **%s** a annulé son congé approuvé : %s, %s.", "de": "🚫 **%s** hat eine genehmigte Abwesenheit storniert: %s, %s.", "it": "🚫 **%s** ha annullato un'assenza approvata: %s, %s."},

	// --- views ---
	"v.menu":       {"en": "🌴 **Leave & absences**", "fr": "🌴 **Congés et absences**", "de": "🌴 **Ferien und Abwesenheiten**", "it": "🌴 **Ferie e assenze**"},
	"v.mine":       {"en": "📋 **My leave requests**", "fr": "📋 **Mes demandes de congé**", "de": "📋 **Meine Abwesenheitsanträge**", "it": "📋 **Le mie richieste di assenza**"},
	"v.mineEmpty":  {"en": "_No requests this year._", "fr": "_Aucune demande cette année._", "de": "_Keine Anträge in diesem Jahr._", "it": "_Nessuna richiesta quest'anno._"},
	"v.queue":      {"en": "✅ **Requests to approve**", "fr": "✅ **Demandes à valider**", "de": "✅ **Zu genehmigende Anträge**", "it": "✅ **Richieste da approvare**"},
	"v.queueEmpty": {"en": "_No pending requests._ 🎉", "fr": "_Aucune demande en attente._ 🎉", "de": "_Keine offenen Anträge._ 🎉", "it": "_Nessuna richiesta in attesa._ 🎉"},
	"v.team":       {"en": "👥 **Absences** from %s to %s", "fr": "👥 **Absences** du %s au %s", "de": "👥 **Abwesenheiten** vom %s bis %s", "it": "👥 **Assenze** dal %s al %s"},
	"v.teamPend":   {"en": "⏳ _pending_", "fr": "⏳ _en attente_", "de": "⏳ _ausstehend_", "it": "⏳ _in attesa_"},
	"v.teamEmpty":  {"en": "_Nobody is away during this period._", "fr": "_Personne n'est absent sur cette période._", "de": "_In diesem Zeitraum ist niemand abwesend._", "it": "_Nessuno è assente in questo periodo._"},
	"nav.menu":     {"en": "‹ Menu", "fr": "‹ Menu", "de": "‹ Menü", "it": "‹ Menu"},
	"nav.new":      {"en": "➕ New request", "fr": "➕ Nouvelle demande", "de": "➕ Neuer Antrag", "it": "➕ Nuova richiesta"},
	"nav.mine":     {"en": "📋 My requests", "fr": "📋 Mes demandes", "de": "📋 Meine Anträge", "it": "📋 Le mie richieste"},
	"nav.team":     {"en": "👥 Absences", "fr": "👥 Absences", "de": "👥 Abwesenheiten", "it": "👥 Assenze"},
	"nav.queue":    {"en": "✅ To approve", "fr": "✅ À valider", "de": "✅ Zu genehmigen", "it": "✅ Da approvare"},
	"bal.approved": {"en": "%s %s: %s days approved", "fr": "%s %s : %s j approuvés", "de": "%s %s: %s Tage genehmigt", "it": "%s %s: %s giorni approvati"},
	"bal.pending":  {"en": ", %s pending", "fr": ", %s j en attente", "de": ", %s ausstehend", "it": ", %s in attesa"},
	"bal.left":     {"en": " · **%s left** of %d", "fr": " · **reste %s j** sur %d", "de": " · **%s übrig** von %d", "it": " · **%s rimanenti** su %d"},

	// --- pinned message & command ---
	"pin.text":      {"en": "🌴 **Leave & absences**: make your requests here. Your manager receives the request and you get their answer by direct message.", "fr": "🌴 **Congés et absences** : faites vos demandes ici. Votre responsable reçoit la demande et vous recevez sa réponse en message privé.", "de": "🌴 **Ferien und Abwesenheiten**: Stellen Sie Ihre Anträge hier. Ihre vorgesetzte Person erhält den Antrag, und Sie bekommen die Antwort per Direktnachricht.", "it": "🌴 **Ferie e assenze**: fai qui le tue richieste. Il tuo responsabile riceve la richiesta e tu ricevi la risposta in un messaggio diretto."},
	"pin.new":       {"en": "➕ Request leave", "fr": "➕ Demander un congé", "de": "➕ Abwesenheit beantragen", "it": "➕ Richiedi un'assenza"},
	"cmd.desc":      {"en": "Request leave, see my requests and who is away", "fr": "Demander un congé, voir mes demandes et les absences", "de": "Abwesenheit beantragen, eigene Anträge und Abwesenheiten ansehen", "it": "Richiedi un'assenza, vedi le tue richieste e chi è assente"},
	"cmd.hint":      {"en": "[new | mine | team | approve | button | help]", "fr": "[demande | mes | absents | valider | bouton | aide]", "de": "[neu | meine | abwesend | genehmigen | knopf | hilfe]", "it": "[nuova | mie | assenti | approva | pulsante | aiuto]"},
	"cmd.onlyMgr":   {"en": "Only managers can approve requests.", "fr": "Seuls les responsables peuvent valider les demandes.", "de": "Nur Vorgesetzte können Anträge genehmigen.", "it": "Solo i responsabili possono approvare le richieste."},
	"cmd.onlyAdmin": {"en": "Only a channel admin can do this.", "fr": "Seul un admin du canal peut faire ça.", "de": "Nur Kanal-Admins können das tun.", "it": "Solo un amministratore del canale può farlo."},
	"cmd.pinFail":   {"en": "Could not post the message: %s", "fr": "Impossible de poster le message : %s", "de": "Die Nachricht konnte nicht gepostet werden: %s", "it": "Impossibile pubblicare il messaggio: %s"},
	"cmd.pinned":    {"en": "The message with the buttons was posted and pinned in this channel.", "fr": "Le message avec les boutons a été posté et épinglé dans ce canal.", "de": "Die Nachricht mit den Schaltflächen wurde in diesem Kanal gepostet und angeheftet.", "it": "Il messaggio con i pulsanti è stato pubblicato e fissato in questo canale."},
	"bot.display":   {"en": "Leave", "fr": "Congés", "de": "Abwesenheiten", "it": "Assenze"},
	"help": {
		"en": "**Leave requests**\n- `/%[1]s`: menu\n- `/%[1]s new`: new request\n- `/%[1]s mine`: my requests (and cancel)\n- `/%[1]s team`: who is away in the coming weeks\n- `/%[1]s approve`: requests to approve (managers)\n- `/%[1]s button`: post and pin a “Request leave” button in this channel (channel admin)",
		"fr": "**Congés**\n- `/%[1]s` : menu\n- `/%[1]s demande` : nouvelle demande\n- `/%[1]s mes` : mes demandes (et annuler)\n- `/%[1]s absents` : qui est absent ces prochaines semaines\n- `/%[1]s valider` : demandes à valider (responsables)\n- `/%[1]s bouton` : poster et épingler un bouton « Demander un congé » dans ce canal (admin du canal)",
		"de": "**Abwesenheiten**\n- `/%[1]s`: Menü\n- `/%[1]s neu`: neuer Antrag\n- `/%[1]s meine`: meine Anträge (und stornieren)\n- `/%[1]s abwesend`: wer in den nächsten Wochen abwesend ist\n- `/%[1]s genehmigen`: zu genehmigende Anträge (Vorgesetzte)\n- `/%[1]s knopf`: eine Schaltfläche „Abwesenheit beantragen“ in diesem Kanal posten und anheften (Kanal-Admin)",
		"it": "**Assenze**\n- `/%[1]s`: menu\n- `/%[1]s nuova`: nuova richiesta\n- `/%[1]s mie`: le mie richieste (e annullare)\n- `/%[1]s assenti`: chi è assente nelle prossime settimane\n- `/%[1]s approva`: richieste da approvare (responsabili)\n- `/%[1]s pulsante`: pubblica e fissa un pulsante «Richiedi un'assenza» in questo canale (amministratore del canale)",
	},

	// --- dates ---
	"range":   {"en": "from %s to %s", "fr": "du %s au %s", "de": "vom %s bis %s", "it": "dal %s al %s"},
	"day1":    {"en": "%s day", "fr": "%s jour", "de": "%s Tag", "it": "%s giorno"},
	"dayN":    {"en": "%s days", "fr": "%s jours", "de": "%s Tage", "it": "%s giorni"},
	"at":      {"en": "%s at %s", "fr": "%s à %s", "de": "%s um %s", "it": "%s alle %s"},
	"example": {"en": "15.12.2026", "fr": "15.12.2026", "de": "15.12.2026", "it": "15.12.2026"},
}

// Subcommand words accepted in every language.
var subWords = map[string][]string{
	"new":    {"new", "request", "demande", "demander", "nouvelle", "neu", "antrag", "nuova", "richiesta"},
	"mine":   {"mine", "my", "list", "mes", "moi", "liste", "meine", "mie", "mia"},
	"queue":  {"approve", "pending", "queue", "valider", "attente", "genehmigen", "offen", "approva", "approvare"},
	"team":   {"team", "absences", "absents", "away", "equipe", "équipe", "abwesend", "abwesenheiten", "assenti", "assenze"},
	"button": {"button", "bouton", "knopf", "pulsante"},
	"help":   {"help", "aide", "hilfe", "aiuto"},
}

func subCommand(word string) string {
	word = strings.ToLower(word)
	for k, ws := range subWords {
		for _, w := range ws {
			if w == word {
				return k
			}
		}
	}
	return ""
}

var defaultTypesByLang = map[string][]string{
	"en": {"Vacation", "Sick leave"},
	"fr": {"Vacances", "Maladie"},
	"de": {"Ferien", "Krankheit"},
	"it": {"Vacanze", "Malattia"},
}

var triggerByLang = map[string]string{"en": "leave", "fr": "conges", "de": "ferien", "it": "ferie"}

var (
	daysLong = map[string][]string{
		"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		"fr": {"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
		"de": {"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		"it": {"domenica", "lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato"},
	}
	daysShort = map[string][]string{
		"en": {"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		"fr": {"dim", "lun", "mar", "mer", "jeu", "ven", "sam"},
		"de": {"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"},
		"it": {"dom", "lun", "mar", "mer", "gio", "ven", "sab"},
	}
	dateLayout = map[string]string{"en": "Jan 2, 2006", "fr": "02.01.2006", "de": "02.01.2006", "it": "02/01/2006"}
)
