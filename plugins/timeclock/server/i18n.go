// SPDX-License-Identifier: MIT
// Copyright (C) 2026 Smafe

package main

// Translations. Each person sees the plugin in their own Mattermost language
// (Profile → Settings → Display → Language). English is the fallback.
// Shared messages (the pinned post) use the server's default client language.

import (
	"fmt"
	"strings"
	"time"
)

var langs = []string{"en", "fr", "de", "it"}

type msg map[string]string // lang -> text (fmt format)

var dayNames = map[string][]string{
	"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
	"fr": {"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
	"de": {"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
	"it": {"domenica", "lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato"},
}

var dayAbbr = map[string][]string{
	"en": {"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	"fr": {"dim", "lun", "mar", "mer", "jeu", "ven", "sam"},
	"de": {"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"},
	"it": {"dom", "lun", "mar", "mer", "gio", "ven", "sab"},
}

var monthNames = map[string][]string{
	"en": {"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	"fr": {"", "janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
	"de": {"", "Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
	"it": {"", "gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre"},
}

func normLang(locale string) string {
	l := strings.ToLower(locale)
	if len(l) > 2 {
		l = l[:2]
	}
	for _, x := range langs {
		if x == l {
			return x
		}
	}
	return "en"
}

// langOf: the person's Mattermost language.
func (p *Plugin) langOf(userID string) string {
	if u, err := p.API.GetUser(userID); err == nil && u.Locale != "" {
		return normLang(u.Locale)
	}
	return p.serverLang()
}

// serverLang: default client language of the server (for shared posts).
func (p *Plugin) serverLang() string {
	if c := p.API.GetConfig(); c != nil && c.LocalizationSettings.DefaultClientLocale != nil {
		return normLang(*c.LocalizationSettings.DefaultClientLocale)
	}
	return "en"
}

func tr(l, key string, a ...any) string {
	m, ok := msgs[key]
	if !ok {
		return key
	}
	s := m[l]
	if s == "" {
		s = m["en"]
	}
	if len(a) > 0 {
		return fmt.Sprintf(s, a...)
	}
	return s
}

func dateLong(l string, t time.Time) string {
	d, m := dayNames[l][t.Weekday()], monthNames[l][t.Month()]
	switch l {
	case "en":
		return fmt.Sprintf("%s, %s %d", d, m, t.Day())
	case "de":
		return fmt.Sprintf("%s, %d. %s", d, t.Day(), m)
	default:
		return fmt.Sprintf("%s %d %s", d, t.Day(), m)
	}
}

func dayShort(l string, t time.Time) string {
	if l == "de" {
		return fmt.Sprintf("%s %d.", dayAbbr[l][t.Weekday()], t.Day())
	}
	return fmt.Sprintf("%s %d", dayAbbr[l][t.Weekday()], t.Day())
}

func monthLabel(l, month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return fmt.Sprintf("%s %d", monthNames[l][t.Month()], t.Year())
}

var msgs = map[string]msg{
	// ---------- buttons ----------
	"btn.in":      {"en": "▶️ Clock in", "fr": "▶️ Je commence", "de": "▶️ Einstempeln", "it": "▶️ Inizio"},
	"btn.toggle":  {"en": "⏸️ Break / Resume", "fr": "⏸️ Pause / Reprise", "de": "⏸️ Pause / Weiter", "it": "⏸️ Pausa / Riprendi"},
	"btn.out":     {"en": "⏹️ Clock out", "fr": "⏹️ Je termine", "de": "⏹️ Ausstempeln", "it": "⏹️ Fine"},
	"btn.mine":    {"en": "📋 My hours", "fr": "📋 Mes heures", "de": "📋 Meine Stunden", "it": "📋 Le mie ore"},
	"btn.pause":   {"en": "⏸️ Break", "fr": "⏸️ Pause", "de": "⏸️ Pause", "it": "⏸️ Pausa"},
	"btn.resume":  {"en": "▶️ Resume", "fr": "▶️ Reprendre", "de": "▶️ Weiter", "it": "▶️ Riprendi"},
	"btn.forgot":  {"en": "✏️ I forgot to clock out", "fr": "✏️ J'ai oublié de pointer la fin", "de": "✏️ Ausstempeln vergessen", "it": "✏️ Ho dimenticato di timbrare la fine"},
	"btn.team":    {"en": "👥 Team", "fr": "👥 Équipe", "de": "👥 Team", "it": "👥 Squadra"},
	"btn.csv":     {"en": "⬇️ CSV file", "fr": "⬇️ Fichier CSV", "de": "⬇️ CSV-Datei", "it": "⬇️ File CSV"},
	"btn.edit":    {"en": "✏️ Fix a day", "fr": "✏️ Corriger une journée", "de": "✏️ Tag korrigieren", "it": "✏️ Correggi un giorno"},
	"btn.back":    {"en": "👥 Back to team", "fr": "👥 Retour à l'équipe", "de": "👥 Zurück zum Team", "it": "👥 Torna alla squadra"},
	"btn.clock":   {"en": "⏱️ Time clock", "fr": "⏱️ Pointer", "de": "⏱️ Stempeln", "it": "⏱️ Timbra"},
	"btn.csvall":  {"en": "⬇️ Export team CSV", "fr": "⬇️ Export CSV de l'équipe", "de": "⬇️ Team-CSV exportieren", "it": "⬇️ Esporta CSV squadra"},
	"btn.detail":  {"en": "See details for…", "fr": "Voir le détail de…", "de": "Details anzeigen für…", "it": "Vedi dettagli di…"},
	"btn.endtime": {"en": "✏️ Enter my end time", "fr": "✏️ Indiquer mon heure de fin", "de": "✏️ Meine Endzeit eingeben", "it": "✏️ Indica l'ora di fine"},
	"btn.outnow":  {"en": "⏹️ Clock out now", "fr": "⏹️ Je termine maintenant", "de": "⏹️ Jetzt ausstempeln", "it": "⏹️ Fine adesso"},
	"btn.still":   {"en": "I'm still working", "fr": "Je travaille encore", "de": "Ich arbeite noch", "it": "Sto ancora lavorando"},

	// ---------- pinned post / panel ----------
	"title": {"en": "⏱️ **Time clock**", "fr": "⏱️ **Pointage**", "de": "⏱️ **Zeiterfassung**", "it": "⏱️ **Timbratura**"},
	"pin.text": {
		"en": "Tap a button to clock in or out. Only you see the reply. Your hours are visible only to you and your managers. You can also type `/pointage` in any channel.",
		"fr": "Touchez un bouton pour pointer. La réponse n'est visible que par vous. Vos heures ne sont visibles que par vous et les responsables. Vous pouvez aussi taper `/pointage` dans n'importe quel canal.",
		"de": "Tippen Sie auf eine Schaltfläche, um ein- oder auszustempeln. Nur Sie sehen die Antwort. Ihre Stunden sehen nur Sie und die Verantwortlichen. Sie können auch `/pointage` in jedem Kanal eingeben.",
		"it": "Tocca un pulsante per timbrare. Solo tu vedi la risposta. Le tue ore sono visibili solo a te e ai responsabili. Puoi anche digitare `/pointage` in qualsiasi canale.",
	},
	"status.off":    {"en": "⚪ You are **not clocked in**.", "fr": "⚪ Vous n'êtes **pas en service**.", "de": "⚪ Sie sind **nicht eingestempelt**.", "it": "⚪ **Non sei in servizio**."},
	"status.paused": {"en": "⏸️ **On break** since %s (shift started %s).", "fr": "⏸️ **En pause** depuis %s (début du service %s).", "de": "⏸️ **In Pause** seit %s (Dienstbeginn %s).", "it": "⏸️ **In pausa** dalle %s (inizio servizio %s)."},
	"status.stale":  {"en": "⚠️ Clock-in started on %s at %s and **never closed**.", "fr": "⚠️ Pointage commencé le %s à %s et **jamais terminé**.", "de": "⚠️ Eingestempelt am %s um %s und **nie ausgestempelt**.", "it": "⚠️ Timbratura iniziata %s alle %s e **mai chiusa**."},
	"status.on":     {"en": "🟢 **Clocked in** since %s.", "fr": "🟢 **En service** depuis %s.", "de": "🟢 **Eingestempelt** seit %s.", "it": "🟢 **In servizio** dalle %s."},
	"panel.totals":  {"en": "\nToday: **%s** · This month: **%s**", "fr": "\nAujourd'hui : **%s** · Ce mois : **%s**", "de": "\nHeute: **%s** · Diesen Monat: **%s**", "it": "\nOggi: **%s** · Questo mese: **%s**"},

	// ---------- clock actions ----------
	"in.ok":          {"en": "▶️ Start recorded at **%s**. Have a good shift!", "fr": "▶️ Début enregistré à **%s**. Bon travail !", "de": "▶️ Beginn um **%s** erfasst. Gutes Schaffen!", "it": "▶️ Inizio registrato alle **%s**. Buon lavoro!"},
	"in.already":     {"en": "You have been clocked in since %s.", "fr": "Vous êtes déjà en service depuis %s.", "de": "Sie sind bereits seit %s eingestempelt.", "it": "Sei già in servizio dalle %s."},
	"in.alreadyp":    {"en": "You have been clocked in since %s (on break). Tap “Resume” to continue.", "fr": "Vous êtes déjà en service depuis %s (en pause). Touchez « Reprendre » pour continuer.", "de": "Sie sind seit %s eingestempelt (in Pause). Tippen Sie auf „Weiter“.", "it": "Sei in servizio dalle %s (in pausa). Tocca «Riprendi» per continuare."},
	"missing":        {"en": "⚠️ Your clock-in of %s (start %s) was never closed. It is marked “missing end” and counts 0 h until corrected.", "fr": "⚠️ Votre pointage du %s (début %s) n'a jamais été terminé. Il est marqué « fin manquante » et compte 0 h jusqu'à correction.", "de": "⚠️ Ihre Zeiterfassung vom %s (Beginn %s) wurde nie abgeschlossen. Sie ist als „Ende fehlt“ markiert und zählt 0 Std. bis zur Korrektur.", "it": "⚠️ La tua timbratura di %s (inizio %s) non è mai stata chiusa. È segnata «fine mancante» e conta 0 h fino alla correzione."},
	"notin.start":    {"en": "You are not clocked in. Tap “Clock in” first.", "fr": "Vous n'êtes pas en service. Touchez d'abord « Je commence ».", "de": "Sie sind nicht eingestempelt. Tippen Sie zuerst auf „Einstempeln“.", "it": "Non sei in servizio. Tocca prima «Inizio»."},
	"notin":          {"en": "You are not clocked in.", "fr": "Vous n'êtes pas en service.", "de": "Sie sind nicht eingestempelt.", "it": "Non sei in servizio."},
	"stale.use":      {"en": "Your clock-in started on %s at %s and was never closed. Enter your real end time with “I forgot to clock out”.", "fr": "Votre pointage a commencé le %s à %s et n'a pas été terminé. Indiquez votre heure de fin avec « J'ai oublié de pointer la fin ».", "de": "Ihre Zeiterfassung begann am %s um %s und wurde nie abgeschlossen. Geben Sie Ihre echte Endzeit über „Ausstempeln vergessen“ ein.", "it": "La tua timbratura è iniziata %s alle %s e non è mai stata chiusa. Indica l'ora di fine con «Ho dimenticato di timbrare la fine»."},
	"pause.already":  {"en": "You have been on a break since %s.", "fr": "Vous êtes déjà en pause depuis %s.", "de": "Sie sind bereits seit %s in Pause.", "it": "Sei già in pausa dalle %s."},
	"pause.not":      {"en": "You are not on a break.", "fr": "Vous n'êtes pas en pause.", "de": "Sie sind nicht in Pause.", "it": "Non sei in pausa."},
	"pause.ok":       {"en": "⏸️ Break started at **%s**. Tap “Resume” when you're back.", "fr": "⏸️ Pause commencée à **%s**. Touchez « Reprendre » au retour.", "de": "⏸️ Pause ab **%s**. Tippen Sie auf „Weiter“, wenn Sie zurück sind.", "it": "⏸️ Pausa iniziata alle **%s**. Tocca «Riprendi» al rientro."},
	"resume.ok":      {"en": "▶️ Back at **%s** (break of %s).", "fr": "▶️ Reprise à **%s** (pause de %s).", "de": "▶️ Weiter um **%s** (Pause %s).", "it": "▶️ Ripresa alle **%s** (pausa di %s)."},
	"notfound":       {"en": "Clock-in not found, please try again.", "fr": "Pointage introuvable, réessayez.", "de": "Zeiterfassung nicht gefunden, bitte erneut versuchen.", "it": "Timbratura non trovata, riprova."},
	"out.ok":         {"en": "⏹️ End recorded at **%s**. This period: %s", "fr": "⏹️ Fin enregistrée à **%s**. Cette période : %s", "de": "⏹️ Ende um **%s** erfasst. Dieser Abschnitt: %s", "it": "⏹️ Fine registrata alle **%s**. Questo periodo: %s"},
	"out.pause":      {"en": " (break %s)", "fr": " (pause %s)", "de": " (Pause %s)", "it": " (pausa %s)"},
	"out.day":        {"en": ". Today's total: **%s**.", "fr": ". Total du jour : **%s**.", "de": ". Heute total: **%s**.", "it": ". Totale di oggi: **%s**."},
	"out.stale":      {"en": "Your clock-in started on %s at %s, more than %d h ago. Enter your real end time with “I forgot to clock out”.", "fr": "Votre pointage a commencé le %s à %s, il y a plus de %d h. Indiquez votre vraie heure de fin avec « J'ai oublié de pointer la fin ».", "de": "Ihre Zeiterfassung begann am %s um %s, vor mehr als %d Std. Geben Sie Ihre echte Endzeit über „Ausstempeln vergessen“ ein.", "it": "La tua timbratura è iniziata %s alle %s, più di %d h fa. Indica l'ora di fine reale con «Ho dimenticato di timbrare la fine»."},
	"close.notfound": {"en": "Clock-in not found.", "fr": "Pointage introuvable.", "de": "Zeiterfassung nicht gefunden.", "it": "Timbratura non trovata."},
	"close.done":     {"en": "This clock-in is already closed.", "fr": "Ce pointage est déjà terminé.", "de": "Diese Zeiterfassung ist bereits abgeschlossen.", "it": "Questa timbratura è già chiusa."},
	"close.before":   {"en": "The end time must be after the start (%s).", "fr": "L'heure de fin doit être après le début (%s).", "de": "Die Endzeit muss nach dem Beginn liegen (%s).", "it": "L'ora di fine deve essere dopo l'inizio (%s)."},
	"close.future":   {"en": "The end time can't be in the future.", "fr": "L'heure de fin ne peut pas être dans le futur.", "de": "Die Endzeit darf nicht in der Zukunft liegen.", "it": "L'ora di fine non può essere nel futuro."},

	// ---------- monthly views ----------
	"month.self":    {"en": "My hours", "fr": "Mes heures", "de": "Meine Stunden", "it": "Le mie ore"},
	"month.head":    {"en": "📋 **%s** · %s\nTotal: **%s** over %d day(s)", "fr": "📋 **%s** · %s\nTotal : **%s** sur %d jour(s)", "de": "📋 **%s** · %s\nTotal: **%s** an %d Tag(en)", "it": "📋 **%s** · %s\nTotale: **%s** su %d giorno/i"},
	"month.missing": {"en": " · ⚠️ %d missing end(s)", "fr": " · ⚠️ %d fin(s) manquante(s)", "de": " · ⚠️ %d fehlende(s) Ende(n)", "it": " · ⚠️ %d fine/i mancante/i"},
	"month.none":    {"en": "_No clock-ins this month._", "fr": "_Aucun pointage ce mois-ci._", "de": "_Keine Zeiterfassung in diesem Monat._", "it": "_Nessuna timbratura questo mese._"},
	"team.line":     {"en": "**%s** · %s · %d day(s)", "fr": "**%s** · %s · %d jour(s)", "de": "**%s** · %s · %d Tag(e)", "it": "**%s** · %s · %d giorno/i"},
	"team.head":     {"en": "👥 **Team** · %s · total %s", "fr": "👥 **Équipe** · %s · total %s", "de": "👥 **Team** · %s · total %s", "it": "👥 **Squadra** · %s · totale %s"},
	"team.on":       {"en": "\n🟢 Clocked in: ", "fr": "\n🟢 En service : ", "de": "\n🟢 Eingestempelt: ", "it": "\n🟢 In servizio: "},
	"team.nobody":   {"en": "\n⚪ Nobody is clocked in right now.", "fr": "\n⚪ Personne en service en ce moment.", "de": "\n⚪ Zurzeit ist niemand eingestempelt.", "it": "\n⚪ Nessuno in servizio in questo momento."},
	"duty.since":    {"en": "%s (since %s", "fr": "%s (depuis %s", "de": "%s (seit %s", "it": "%s (dalle %s"},
	"duty.paused":   {"en": ", on break", "fr": ", en pause", "de": ", in Pause", "it": ", in pausa"},
	"deleted":       {"en": "(deleted)", "fr": "(supprimé)", "de": "(gelöscht)", "it": "(eliminato)"},

	// ---------- commands / misc ----------
	"help": {
		"en": "**/pointage** opens your time clock. Other commands:\n- `/pointage start`, `/pointage break`, `/pointage resume`, `/pointage end`\n- `/pointage hours [month]`: your hours for the month (e.g. `/pointage hours 10.2026`)\n- `/pointage export [month]`: CSV file (your hours, or the whole team for a manager)\n- `/pointage team [month]`: team hours (managers)\n- `/pointage fix`: fix a day (managers)\n- `/pointage enable` / `disable` / `button`: pinned message with the buttons in this channel (admins)",
		"fr": "**/pointage** ouvre votre pointage. Autres commandes :\n- `/pointage debut`, `/pointage pause`, `/pointage reprise`, `/pointage fin`\n- `/pointage heures [mois]` : vos heures du mois (ex. `/pointage heures 10.2026`)\n- `/pointage export [mois]` : fichier CSV (vos heures, ou toute l'équipe pour un responsable)\n- `/pointage equipe [mois]` : heures de l'équipe (responsables)\n- `/pointage corriger` : corriger une journée (responsables)\n- `/pointage activer` / `desactiver` / `bouton` : message épinglé avec les boutons dans ce canal (admins)",
		"de": "**/pointage** öffnet Ihre Zeiterfassung. Weitere Befehle:\n- `/pointage beginn`, `/pointage pause`, `/pointage weiter`, `/pointage ende`\n- `/pointage stunden [Monat]`: Ihre Stunden des Monats (z. B. `/pointage stunden 10.2026`)\n- `/pointage export [Monat]`: CSV-Datei (Ihre Stunden oder das ganze Team für Verantwortliche)\n- `/pointage team [Monat]`: Stunden des Teams (Verantwortliche)\n- `/pointage korrigieren`: einen Tag korrigieren (Verantwortliche)\n- `/pointage aktivieren` / `deaktivieren` / `knopf`: angeheftete Nachricht mit den Schaltflächen in diesem Kanal (Admins)",
		"it": "**/pointage** apre la tua timbratura. Altri comandi:\n- `/pointage inizio`, `/pointage pausa`, `/pointage riprendi`, `/pointage fine`\n- `/pointage ore [mese]`: le tue ore del mese (es. `/pointage ore 10.2026`)\n- `/pointage export [mese]`: file CSV (le tue ore, o tutta la squadra per un responsabile)\n- `/pointage squadra [mese]`: ore della squadra (responsabili)\n- `/pointage correggi`: correggere un giorno (responsabili)\n- `/pointage attiva` / `disattiva` / `pulsante`: messaggio fissato con i pulsanti in questo canale (admin)",
	},
	"cmd.desc":      {"en": "Clock in, break, clock out and see your hours", "fr": "Pointer (début, pause, fin) et voir ses heures", "de": "Ein-/Ausstempeln, Pause und Stunden ansehen", "it": "Timbrare (inizio, pausa, fine) e vedere le ore"},
	"cmd.hint":      {"en": "[start | break | end | hours | team]", "fr": "[debut | pause | fin | heures | equipe]", "de": "[beginn | pause | ende | stunden | team]", "it": "[inizio | pausa | fine | ore | squadra]"},
	"badmonth":      {"en": "Invalid month. Example: `/pointage %s 10.2026`", "fr": "Mois invalide. Exemple : `/pointage %s 10.2026`", "de": "Ungültiger Monat. Beispiel: `/pointage %s 10.2026`", "it": "Mese non valido. Esempio: `/pointage %s 10.2026`"},
	"managers.only": {"en": "Managers only.", "fr": "Réservé aux responsables.", "de": "Nur für Verantwortliche.", "it": "Solo per i responsabili."},
	"fix.only":      {"en": "Only a manager can fix hours. Please ask one.", "fr": "Seul un responsable peut corriger les heures. Demandez-lui.", "de": "Nur Verantwortliche können Stunden korrigieren. Bitte fragen Sie nach.", "it": "Solo un responsabile può correggere le ore. Chiedi a lui."},
	"chadmin.only":  {"en": "Only a channel admin can do that.", "fr": "Seul un admin du canal peut faire ça.", "de": "Nur Kanal-Admins können das tun.", "it": "Solo un amministratore del canale può farlo."},
	"enabled":       {"en": "⏱️ The time clock message has been posted and pinned in this channel.", "fr": "⏱️ Le message de pointage a été posté et épinglé dans ce canal.", "de": "⏱️ Die Zeiterfassungs-Nachricht wurde in diesem Kanal gepostet und angeheftet.", "it": "⏱️ Il messaggio di timbratura è stato pubblicato e fissato in questo canale."},
	"disabled":      {"en": "The time clock message has been removed from this channel. Hours are kept and `/pointage` still works.", "fr": "Le message de pointage a été retiré de ce canal. Les heures sont conservées et `/pointage` marche toujours.", "de": "Die Zeiterfassungs-Nachricht wurde aus diesem Kanal entfernt. Die Stunden bleiben erhalten und `/pointage` funktioniert weiterhin.", "it": "Il messaggio di timbratura è stato rimosso da questo canale. Le ore sono conservate e `/pointage` funziona ancora."},
	"unknown.cmd":   {"en": "Unknown command.", "fr": "Commande inconnue.", "de": "Unbekannter Befehl.", "it": "Comando sconosciuto."},
	"unknown.act":   {"en": "Unknown action.", "fr": "Action inconnue.", "de": "Unbekannte Aktion.", "it": "Azione sconosciuta."},
	"error":         {"en": "Error: %s", "fr": "Erreur : %s", "de": "Fehler: %s", "it": "Errore: %s"},

	// ---------- dialogs ----------
	"dlg.notrigger": {"en": "Can't open the form here.", "fr": "Impossible d'ouvrir le formulaire ici.", "de": "Das Formular kann hier nicht geöffnet werden.", "it": "Impossibile aprire il modulo qui."},
	"edit.title":    {"en": "Fix a day", "fr": "Corriger une journée", "de": "Tag korrigieren", "it": "Correggi un giorno"},
	"edit.intro":    {"en": "Replaces all of this person's clock-ins for that day.", "fr": "Remplace tous les pointages de cette personne pour ce jour-là.", "de": "Ersetzt alle Zeiterfassungen dieser Person an diesem Tag.", "it": "Sostituisce tutte le timbrature di questa persona per quel giorno."},
	"edit.person":   {"en": "Person", "fr": "Personne", "de": "Person", "it": "Persona"},
	"f.date":        {"en": "Date", "fr": "Date", "de": "Datum", "it": "Data"},
	"f.dateph":      {"en": "DD.MM.YYYY", "fr": "JJ.MM.AAAA", "de": "TT.MM.JJJJ", "it": "GG.MM.AAAA"},
	"f.start":       {"en": "Start", "fr": "Début", "de": "Beginn", "it": "Inizio"},
	"f.end":         {"en": "End", "fr": "Fin", "de": "Ende", "it": "Fine"},
	"f.starthelp":   {"en": "Leave Start and End empty to delete this day.", "fr": "Laissez Début et Fin vides pour effacer cette journée.", "de": "Beginn und Ende leer lassen, um diesen Tag zu löschen.", "it": "Lascia vuoti Inizio e Fine per cancellare questo giorno."},
	"f.endhelp":     {"en": "If the end is earlier than the start, it counts for the next day (night shift).", "fr": "Si la fin est plus tôt que le début, elle compte pour le lendemain (service de nuit).", "de": "Liegt das Ende vor dem Beginn, zählt es für den nächsten Tag (Nachtschicht).", "it": "Se la fine è prima dell'inizio, conta per il giorno dopo (turno di notte)."},
	"f.pause":       {"en": "Break (minutes)", "fr": "Pause (minutes)", "de": "Pause (Minuten)", "it": "Pausa (minuti)"},
	"f.note":        {"en": "Note", "fr": "Remarque", "de": "Bemerkung", "it": "Nota"},
	"f.save":        {"en": "Save", "fr": "Enregistrer", "de": "Speichern", "it": "Salva"},
	"e.person":      {"en": "Choose a person.", "fr": "Choisissez une personne.", "de": "Wählen Sie eine Person.", "it": "Scegli una persona."},
	"e.personnf":    {"en": "Person not found.", "fr": "Personne introuvable.", "de": "Person nicht gefunden.", "it": "Persona non trovata."},
	"e.date":        {"en": "Format DD.MM.YYYY, e.g. 03.11.2026", "fr": "Format JJ.MM.AAAA, par ex. 03.11.2026", "de": "Format TT.MM.JJJJ, z. B. 03.11.2026", "it": "Formato GG.MM.AAAA, es. 03.11.2026"},
	"e.time":        {"en": "Invalid time (HH:MM).", "fr": "Heure invalide (HH:MM).", "de": "Ungültige Uhrzeit (HH:MM).", "it": "Ora non valida (HH:MM)."},
	"e.pause":       {"en": "Number of minutes, e.g. 30.", "fr": "Nombre de minutes, par ex. 30.", "de": "Anzahl Minuten, z. B. 30.", "it": "Numero di minuti, es. 30."},
	"e.future":      {"en": "The end can't be in the future.", "fr": "La fin ne peut pas être dans le futur.", "de": "Das Ende darf nicht in der Zukunft liegen.", "it": "La fine non può essere nel futuro."},
	"e.pauselong":   {"en": "The break is longer than the day.", "fr": "La pause est plus longue que la journée.", "de": "Die Pause ist länger als der Tag.", "it": "La pausa è più lunga della giornata."},
	"e.ondut":       {"en": "This person is clocked in right now. Fix this day after their shift ends.", "fr": "Cette personne est en service en ce moment. Corrigez cette journée après la fin du service.", "de": "Diese Person ist gerade eingestempelt. Korrigieren Sie diesen Tag nach Dienstende.", "it": "Questa persona è in servizio ora. Correggi questo giorno dopo la fine del servizio."},
	"e.form":        {"en": "Invalid form, please try again.", "fr": "Formulaire invalide, réessayez.", "de": "Ungültiges Formular, bitte erneut versuchen.", "it": "Modulo non valido, riprova."},
	"edit.cleared":  {"en": "day deleted", "fr": "journée effacée", "de": "Tag gelöscht", "it": "giorno cancellato"},
	"edit.what":     {"en": "%s–%s, break %s, total **%s**", "fr": "%s–%s, pause %s, total **%s**", "de": "%s–%s, Pause %s, total **%s**", "it": "%s–%s, pausa %s, totale **%s**"},
	"edit.done":     {"en": "✏️ %s, **%s**: %s.", "fr": "✏️ Journée du %s de **%s** : %s.", "de": "✏️ %s, **%s**: %s.", "it": "✏️ %s, **%s**: %s."},
	"edit.dm":       {"en": "✏️ **%s** fixed your hours for %s: %s.", "fr": "✏️ **%s** a corrigé votre pointage du %s : %s.", "de": "✏️ **%s** hat Ihre Zeiterfassung vom %s korrigiert: %s.", "it": "✏️ **%s** ha corretto la tua timbratura di %s: %s."},
	"forgot.none":   {"en": "You have no open clock-in.", "fr": "Vous n'avez pas de pointage en cours.", "de": "Sie haben keine offene Zeiterfassung.", "it": "Non hai timbrature aperte."},
	"forgot.title":  {"en": "Forgotten end time", "fr": "Heure de fin oubliée", "de": "Vergessene Endzeit", "it": "Ora di fine dimenticata"},
	"forgot.intro":  {"en": "You clocked in on **%s at %s**. What time did you finish?", "fr": "Vous avez commencé le **%s à %s**. À quelle heure avez-vous terminé ?", "de": "Sie haben am **%s um %s** eingestempelt. Wann haben Sie aufgehört?", "it": "Hai iniziato **%s alle %s**. A che ora hai finito?"},
	"forgot.date":   {"en": "End date", "fr": "Date de fin", "de": "Enddatum", "it": "Data di fine"},
	"forgot.time":   {"en": "End time", "fr": "Heure de fin", "de": "Endzeit", "it": "Ora di fine"},
	"forgot.ok":     {"en": "⏹️ End recorded: %s at **%s**. This period: %s.", "fr": "⏹️ Fin enregistrée : %s à **%s**. Cette période : %s.", "de": "⏹️ Ende erfasst: %s um **%s**. Dieser Abschnitt: %s.", "it": "⏹️ Fine registrata: %s alle **%s**. Questo periodo: %s."},
	"dlg.unknown":   {"en": "Unknown form.", "fr": "Formulaire inconnu.", "de": "Unbekanntes Formular.", "it": "Modulo sconosciuto."},

	// ---------- export ----------
	"csv.cols":    {"en": "Name;Username;Date;Day;Start;End;Break (h:mm);Worked (h:mm);Worked (minutes);Remark", "fr": "Nom;Utilisateur;Date;Jour;Début;Fin;Pause (h:mm);Travail (h:mm);Travail (minutes);Remarque", "de": "Name;Benutzer;Datum;Tag;Beginn;Ende;Pause (h:mm);Arbeit (h:mm);Arbeit (Minuten);Bemerkung", "it": "Nome;Utente;Data;Giorno;Inizio;Fine;Pausa (h:mm);Lavoro (h:mm);Lavoro (minuti);Nota"},
	"csv.totcols": {"en": "Name;Username;Days worked;Total (h:mm);Total (minutes);Missing ends", "fr": "Nom;Utilisateur;Jours travaillés;Total (h:mm);Total (minutes);Fins manquantes", "de": "Name;Benutzer;Arbeitstage;Total (h:mm);Total (Minuten);Fehlende Enden", "it": "Nome;Utente;Giorni lavorati;Totale (h:mm);Totale (minuti);Fini mancanti"},
	"csv.total":   {"en": "TOTAL %s", "fr": "TOTAL %s", "de": "TOTAL %s", "it": "TOTALE %s"},
	"r.missing":   {"en": "missing end", "fr": "fin manquante", "de": "Ende fehlt", "it": "fine mancante"},
	"r.open":      {"en": "in progress", "fr": "en cours", "de": "läuft", "it": "in corso"},
	"r.self":      {"en": "end entered afterwards", "fr": "fin saisie après coup", "de": "Ende nachträglich erfasst", "it": "fine inserita dopo"},
	"r.by":        {"en": "fixed by %s", "fr": "corrigé par %s", "de": "korrigiert von %s", "it": "corretto da %s"},
	"csv.none":    {"en": "No clock-ins in %s.", "fr": "Aucun pointage en %s.", "de": "Keine Zeiterfassung im %s.", "it": "Nessuna timbratura a %s."},
	"csv.dmerr":   {"en": "⚠️ Can't open the private conversation: %s", "fr": "⚠️ Impossible d'ouvrir la conversation privée : %s", "de": "⚠️ Private Unterhaltung kann nicht geöffnet werden: %s", "it": "⚠️ Impossibile aprire la conversazione privata: %s"},
	"csv.upload":  {"en": "⚠️ Can't send the file: %s", "fr": "⚠️ Envoi du fichier impossible : %s", "de": "⚠️ Datei kann nicht gesendet werden: %s", "it": "⚠️ Impossibile inviare il file: %s"},
	"csv.of":      {"en": "Time clock of %s", "fr": "Pointage de %s", "de": "Zeiterfassung von %s", "it": "Timbrature di %s"},
	"csv.ofv":     {"en": "Time clock of %s", "fr": "Pointage d'%s", "de": "Zeiterfassung von %s", "it": "Timbrature di %s"},
	"csv.team":    {"en": "Team time clock", "fr": "Pointage de l'équipe", "de": "Zeiterfassung des Teams", "it": "Timbrature della squadra"},
	"csv.post":    {"en": "⬇️ %s · %s (generated %s).", "fr": "⬇️ %s · %s (généré le %s).", "de": "⬇️ %s · %s (erstellt am %s).", "it": "⬇️ %s · %s (generato il %s)."},
	"csv.sent":    {"en": "⬇️ The CSV file has been sent to you in a private message from @%s.", "fr": "⬇️ Le fichier CSV vous a été envoyé en message privé par @%s.", "de": "⬇️ Die CSV-Datei wurde Ihnen per Direktnachricht von @%s gesendet.", "it": "⬇️ Il file CSV ti è stato inviato in un messaggio privato da @%s."},
	"csv.file":    {"en": "timeclock", "fr": "pointage", "de": "zeiterfassung", "it": "timbrature"},
	"remind":      {"en": "⏰ You have been clocked in since %s at %s (%d h). Did you forget to clock out?", "fr": "⏰ Vous êtes en service depuis le %s à %s (%d h). Avez-vous oublié de pointer la fin ?", "de": "⏰ Sie sind seit %s um %s eingestempelt (%d Std.). Haben Sie das Ausstempeln vergessen?", "it": "⏰ Sei in servizio da %s alle %s (%d h). Hai dimenticato di timbrare la fine?"},
}
