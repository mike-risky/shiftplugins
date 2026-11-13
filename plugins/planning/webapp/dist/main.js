/* SPDX-License-Identifier: GPL-3.0-or-later
 * Copyright (C) 2026 Smafe */
/* Planning plugin webapp — plain JS, uses the React instance provided by Mattermost. */
(function () {
    'use strict';

    var PLUGIN_ID = 'com.shiftplugins.planning';
    var React = window.React;
    var h = React.createElement;
    var useState = React.useState;
    var useEffect = React.useEffect;
    var useCallback = React.useCallback;


    // ---------- i18n (English default; follows each user's Mattermost language) ----------
    var STORE = null;
    var DICT = {
        months: {en: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
            fr: ['janvier', 'février', 'mars', 'avril', 'mai', 'juin', 'juillet', 'août', 'septembre', 'octobre', 'novembre', 'décembre'],
            de: ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'],
            it: ['gennaio', 'febbraio', 'marzo', 'aprile', 'maggio', 'giugno', 'luglio', 'agosto', 'settembre', 'ottobre', 'novembre', 'dicembre']},
        wdShort: {en: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'], fr: ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'],
            de: ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'], it: ['Lun', 'Mar', 'Mer', 'Gio', 'Ven', 'Sab', 'Dom']},
        wdLong: {en: ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
            fr: ['dimanche', 'lundi', 'mardi', 'mercredi', 'jeudi', 'vendredi', 'samedi'],
            de: ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'],
            it: ['domenica', 'lunedì', 'martedì', 'mercoledì', 'giovedì', 'venerdì', 'sabato']},
    };
    var T = {
        error: {en: 'Error {0}', fr: 'Erreur {0}', de: 'Fehler {0}', it: 'Errore {0}'},
        editShift: {en: 'Edit shift', fr: 'Modifier le shift', de: 'Schicht bearbeiten', it: 'Modifica turno'},
        newShift: {en: 'New shift', fr: 'Nouveau shift', de: 'Neue Schicht', it: 'Nuovo turno'},
        name: {en: 'Name', fr: 'Nom', de: 'Name', it: 'Nome'},
        namePh: {en: 'e.g. Morning, Evening, On call…', fr: 'ex. Matin, Soir, Permanence…', de: 'z. B. Früh, Spät, Bereitschaft…', it: 'es. Mattina, Sera, Reperibilità…'},
        start: {en: 'Start (optional)', fr: 'Début (optionnel)', de: 'Beginn (optional)', it: 'Inizio (facoltativo)'},
        end: {en: 'End', fr: 'Fin', de: 'Ende', it: 'Fine'},
        people: {en: 'Number of people', fr: 'Nombre de personnes', de: 'Anzahl Personen', it: 'Numero di persone'},
        repeat: {en: 'Repeat', fr: 'Répéter', de: 'Wiederholen', it: 'Ripeti'},
        rNone: {en: 'No, just this day', fr: 'Non, seulement ce jour', de: 'Nein, nur dieser Tag', it: 'No, solo questo giorno'},
        rDaily: {en: 'Every day', fr: 'Tous les jours', de: 'Jeden Tag', it: 'Ogni giorno'},
        rWeekdays: {en: 'Monday to Friday', fr: 'Du lundi au vendredi', de: 'Montag bis Freitag', it: 'Dal lunedì al venerdì'},
        rWeekly: {en: 'Every week ({0})', fr: 'Chaque semaine ({0})', de: 'Jede Woche ({0})', it: 'Ogni settimana ({0})'},
        until: {en: 'Until', fr: 'Jusqu\'au', de: 'Bis', it: 'Fino al'},
        save: {en: 'Save', fr: 'Enregistrer', de: 'Speichern', it: 'Salva'},
        createN: {en: 'Create {0} shifts', fr: 'Créer {0} shifts', de: '{0} Schichten erstellen', it: 'Crea {0} turni'},
        create: {en: 'Create', fr: 'Créer', de: 'Erstellen', it: 'Crea'},
        cancel: {en: 'Cancel', fr: 'Annuler', de: 'Abbrechen', it: 'Annulla'},
        locked: {en: 'Locked', fr: 'Verrouillé', de: 'Gesperrt', it: 'Bloccato'},
        swapPending: {en: 'Swap waiting for approval', fr: 'Échange en attente de validation', de: 'Tausch wartet auf Genehmigung', it: 'Scambio in attesa di approvazione'},
        swapOffered: {en: 'Shift offered for swap', fr: 'Shift proposé à l\'échange', de: 'Schicht zum Tausch angeboten', it: 'Turno proposto per lo scambio'},
        remove: {en: 'Remove', fr: 'Retirer', de: 'Entfernen', it: 'Rimuovi'},
        free1: {en: '1 place left', fr: '1 place libre', de: '1 Platz frei', it: '1 posto libero'},
        freeN: {en: '{0} places left', fr: '{0} places libres', de: '{0} Plätze frei', it: '{0} posti liberi'},
        take: {en: 'Take this shift', fr: 'Prendre ce shift', de: 'Schicht übernehmen', it: 'Prendi questo turno'},
        release: {en: 'Release', fr: 'Libérer', de: 'Freigeben', it: 'Libera'},
        lockedByAdmin: {en: 'Locked by an admin', fr: 'Verrouillé par un admin', de: 'Von einem Admin gesperrt', it: 'Bloccato da un admin'},
        offer: {en: '🔄 Offer a swap', fr: '🔄 Proposer un échange', de: '🔄 Tausch anbieten', it: '🔄 Proponi uno scambio'},
        cancelOffer: {en: 'Cancel my offer', fr: 'Annuler mon offre', de: 'Mein Angebot zurückziehen', it: 'Annulla la mia offerta'},
        waitApproval: {en: '⏳ {0} is waiting for an admin to approve', fr: '⏳ {0} attend la validation d\'un admin', de: '⏳ {0} wartet auf die Genehmigung eines Admins', it: '⏳ {0} attende l\'approvazione di un admin'},
        claim: {en: '🔄 Take over from {0}', fr: '🔄 Reprendre la place de {0}', de: '🔄 Von {0} übernehmen', it: '🔄 Prendi il posto di {0}'},
        replaces: {en: '{0} replaces {1}?', fr: '{0} remplace {1} ?', de: '{0} ersetzt {1}?', it: '{0} sostituisce {1}?'},
        approve: {en: 'Approve', fr: 'Approuver', de: 'Genehmigen', it: 'Approva'},
        refuse: {en: 'Refuse', fr: 'Refuser', de: 'Ablehnen', it: 'Rifiuta'},
        unlock: {en: '🔓 Unlock', fr: '🔓 Déverrouiller', de: '🔓 Entsperren', it: '🔓 Sblocca'},
        lock: {en: '🔒 Lock', fr: '🔒 Verrouiller', de: '🔒 Sperren', it: '🔒 Blocca'},
        edit: {en: 'Edit', fr: 'Modifier', de: 'Bearbeiten', it: 'Modifica'},
        del: {en: 'Delete', fr: 'Supprimer', de: 'Löschen', it: 'Elimina'},
        delConfirm: {en: 'Delete this shift?', fr: 'Supprimer ce shift ?', de: 'Diese Schicht löschen?', it: 'Eliminare questo turno?'},
        signUpPh: {en: 'Sign someone up…', fr: 'Inscrire quelqu\'un…', de: 'Jemanden eintragen…', it: 'Iscrivi qualcuno…'},
        signUp: {en: 'Sign up', fr: 'Inscrire', de: 'Eintragen', it: 'Iscrivi'},
        prevMonth: {en: 'Previous month', fr: 'Mois précédent', de: 'Vorheriger Monat', it: 'Mese precedente'},
        nextMonth: {en: 'Next month', fr: 'Mois suivant', de: 'Nächster Monat', it: 'Mese successivo'},
        today: {en: 'Today', fr: 'Aujourd\'hui', de: 'Heute', it: 'Oggi'},
        lockMonthQ: {en: 'Lock every shift of this month?', fr: 'Verrouiller tous les shifts de ce mois ?', de: 'Alle Schichten dieses Monats sperren?', it: 'Bloccare tutti i turni di questo mese?'},
        unlockMonthQ: {en: 'Unlock every shift of this month?', fr: 'Déverrouiller tous les shifts de ce mois ?', de: 'Alle Schichten dieses Monats entsperren?', it: 'Sbloccare tutti i turni di questo mese?'},
        lockMonth: {en: '🔒 Lock month', fr: '🔒 Verrouiller le mois', de: '🔒 Monat sperren', it: '🔒 Blocca il mese'},
        unlockMonth: {en: '🔓 Unlock month', fr: '🔓 Déverrouiller le mois', de: '🔓 Monat entsperren', it: '🔓 Sblocca il mese'},
        disableQ: {en: 'Disable the planning in this channel? Shifts are kept.', fr: 'Désactiver le planning sur ce canal ? Les shifts sont conservés.', de: 'Schichtplan in diesem Kanal deaktivieren? Die Schichten bleiben erhalten.', it: 'Disattivare il planning in questo canale? I turni vengono conservati.'},
        disable: {en: 'Disable', fr: 'Désactiver', de: 'Deaktivieren', it: 'Disattiva'},
        close: {en: 'Close', fr: 'Fermer', de: 'Schließen', it: 'Chiudi'},
        loading: {en: 'Loading…', fr: 'Chargement…', de: 'Wird geladen…', it: 'Caricamento…'},
        notEnabled: {en: 'The planning is not enabled in this channel.', fr: 'Le planning n\'est pas activé sur ce canal.', de: 'Der Schichtplan ist in diesem Kanal nicht aktiviert.', it: 'Il planning non è attivo in questo canale.'},
        enableHere: {en: 'Enable planning here', fr: 'Activer le planning ici', de: 'Schichtplan hier aktivieren', it: 'Attiva il planning qui'},
        askAdmin: {en: 'Ask a channel admin to enable it.', fr: 'Demandez à un admin du canal de l\'activer.', de: 'Bitten Sie einen Kanal-Admin, ihn zu aktivieren.', it: 'Chiedi a un admin del canale di attivarlo.'},
        noShiftAdmin: {en: 'No shifts this day. Add one below.', fr: 'Aucun shift ce jour. Ajoutez-en un ci-dessous.', de: 'Keine Schichten an diesem Tag. Fügen Sie unten eine hinzu.', it: 'Nessun turno in questo giorno. Aggiungine uno qui sotto.'},
        noShift: {en: 'No shifts planned this day.', fr: 'Aucun shift prévu ce jour.', de: 'Keine Schichten an diesem Tag geplant.', it: 'Nessun turno previsto in questo giorno.'},
        addShift: {en: '+ Add a shift', fr: '+ Ajouter un shift', de: '+ Schicht hinzufügen', it: '+ Aggiungi un turno'},
        tooltip: {en: 'Shift planning', fr: 'Planning des shifts', de: 'Schichtplan', it: 'Planning dei turni'},
        title: {en: 'Planning', fr: 'Planning', de: 'Schichtplan', it: 'Planning'},
    };
    function lang() {
        var l = '';
        try {
            var st = STORE.getState();
            var me = st.entities.users.profiles[st.entities.users.currentUserId];
            l = (me && me.locale) || st.entities.general.config.DefaultClientLocale || '';
        } catch (e) { /* ignore */ }
        l = String(l).slice(0, 2).toLowerCase();
        return T.today[l] ? l : 'en';
    }
    function t(key) {
        var m = T[key] || {}, s = m[lang()] || m.en || key;
        var args = arguments;
        return s.replace(/\{(\d)\}/g, function (_, i) { return args[+i + 1]; });
    }
    function months() { return DICT.months[lang()]; }
    function wdLong() { return DICT.wdLong[lang()]; }

    // ---------- tiny global "open" store ----------
    var listeners = [];
    var openState = {channelId: null};
    function setOpen(channelId) {
        openState = {channelId: channelId};
        listeners.forEach(function (l) { l(openState); });
    }

    // ---------- helpers ----------
    function pad(n) { return (n < 10 ? '0' : '') + n; }
    function ymd(d) { return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()); }
    function ym(d) { return d.getFullYear() + '-' + pad(d.getMonth() + 1); }
    function parseDate(s) { var p = s.split('-'); return new Date(+p[0], +p[1] - 1, +p[2]); }
    function longDate(s) {
        var d = parseDate(s), l = lang(), wd = wdLong()[d.getDay()], mo = months()[d.getMonth()];
        if (l === 'en') { return wd + ', ' + mo + ' ' + d.getDate(); }
        if (l === 'de') { return wd + ', ' + d.getDate() + '. ' + mo; }
        return wd + ' ' + d.getDate() + ' ' + mo;
    }
    function timeLabel(s) { return s.start ? s.start + (s.end ? '–' + s.end : '') : ''; }

    function baseUrl() {
        return (window.basename || '') + '/plugins/' + PLUGIN_ID + '/api/v1';
    }
    function csrf() {
        var m = document.cookie.match(/(?:^|;\s*)MMCSRF=([^;]+)/);
        return m ? m[1] : '';
    }
    function api(method, path, body) {
        return fetch(baseUrl() + path, {
            method: method,
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
                'X-CSRF-Token': csrf(),
            },
            body: body ? JSON.stringify(body) : undefined,
        }).then(function (r) {
            return r.json().catch(function () { return {}; }).then(function (j) {
                if (!r.ok) { throw new Error(j.error || t('error', r.status)); }
                return j;
            });
        });
    }

    function computeDates(from, repeat, until) {
        if (repeat === 'none' || !until) { return [from]; }
        var out = [];
        var d = parseDate(from);
        var end = parseDate(until);
        while (d <= end && out.length < 120) {
            var wd = d.getDay();
            if (repeat === 'daily' || (repeat === 'weekdays' && wd >= 1 && wd <= 5) ||
                (repeat === 'weekly' && wd === parseDate(from).getDay())) {
                out.push(ymd(d));
            }
            d.setDate(d.getDate() + 1);
        }
        return out;
    }

    // ---------- styles ----------
    var CSS = '' +
        '.pl-overlay{position:fixed;inset:0;background:rgba(0,0,0,.45);z-index:2000;display:flex;align-items:center;justify-content:center}' +
        '.pl-modal{background:var(--center-channel-bg,#fff);color:var(--center-channel-color,#1d1d1b);width:min(1180px,96vw);height:min(860px,94vh);border-radius:12px;box-shadow:0 20px 60px rgba(0,0,0,.35);display:flex;flex-direction:column;overflow:hidden;font-size:14px}' +
        '.pl-head{display:flex;align-items:center;gap:8px;padding:14px 18px;border-bottom:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.12)}' +
        '.pl-title{font-size:18px;font-weight:600;margin-right:auto}' +
        '.pl-btn{border:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.2);background:transparent;color:inherit;border-radius:6px;padding:6px 10px;cursor:pointer;font-size:13px;line-height:1.2}' +
        '.pl-btn:hover{background:rgba(var(--center-channel-color-rgb,29,29,27),.08)}' +
        '.pl-btn:disabled{opacity:.45;cursor:default}' +
        '.pl-primary{background:var(--button-bg,#1d1d1b);color:var(--button-color,#fff);border-color:transparent}' +
        '.pl-primary:hover{background:var(--button-bg,#1d1d1b);filter:brightness(1.1)}' +
        '.pl-danger{color:var(--error-text,#d24b4e);border-color:rgba(210,75,78,.4)}' +
        '.pl-x{font-size:20px;padding:2px 10px;border:none}' +
        '.pl-body{display:flex;flex:1;min-height:0}' +
        '.pl-cal{flex:1;display:flex;flex-direction:column;padding:10px 12px;min-width:0}' +
        '.pl-wd{display:grid;grid-template-columns:repeat(7,1fr);gap:4px;font-size:12px;opacity:.6;padding:0 2px 4px}' +
        '.pl-grid{flex:1;display:grid;grid-template-columns:repeat(7,1fr);grid-auto-rows:1fr;gap:4px;min-height:0}' +
        '.pl-day{border:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.12);border-radius:8px;padding:4px;overflow:hidden;cursor:pointer;display:flex;flex-direction:column;gap:2px;min-height:0}' +
        '.pl-day:hover{border-color:rgba(var(--center-channel-color-rgb,29,29,27),.35)}' +
        '.pl-day.out{opacity:.35}' +
        '.pl-day.sel{outline:2px solid var(--button-bg,#1d1d1b);outline-offset:-1px}' +
        '.pl-num{font-size:12px;font-weight:600}' +
        '.pl-day.today .pl-num{background:var(--button-bg,#1d1d1b);color:var(--button-color,#fff);border-radius:10px;padding:0 6px;align-self:flex-start}' +
        '.pl-chip{font-size:11px;border-radius:4px;padding:1px 4px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;border:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.25)}' +
        '.pl-chip.open{border-style:dashed}' +
        '.pl-chip.full{background:rgba(var(--center-channel-color-rgb,29,29,27),.08)}' +
        '.pl-chip.mine{background:#74FAC1;color:#1d1d1b;border-color:#74FAC1;font-weight:600}' +
        '.pl-side{width:340px;border-left:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.12);padding:14px;overflow:auto;display:flex;flex-direction:column;gap:10px}' +
        '.pl-side h3{margin:0;font-size:16px}.pl-side h3::first-letter{text-transform:uppercase}' +
        '.pl-card{border:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.15);border-radius:8px;padding:10px;display:flex;flex-direction:column;gap:6px}' +
        '.pl-card.mine{border-color:#74FAC1;box-shadow:inset 3px 0 0 #74FAC1}' +
        '.pl-row{display:flex;align-items:center;gap:6px;flex-wrap:wrap}' +
        '.pl-sp{flex:1}' +
        '.pl-muted{opacity:.6;font-size:12px}' +
        '.pl-person{display:inline-flex;align-items:center;gap:4px;background:rgba(var(--center-channel-color-rgb,29,29,27),.08);border-radius:10px;padding:2px 8px;font-size:12px}' +
        '.pl-person button{border:none;background:none;color:inherit;cursor:pointer;padding:0;opacity:.6}' +
        '.pl-form{display:flex;flex-direction:column;gap:6px}' +
        '.pl-form label{font-size:12px;opacity:.75}' +
        '.pl-in{width:100%;box-sizing:border-box;padding:6px 8px;border-radius:6px;border:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.25);background:var(--center-channel-bg,#fff);color:inherit;font-size:13px}' +
        '.pl-err{background:rgba(210,75,78,.12);color:var(--error-text,#d24b4e);padding:8px 12px;border-radius:6px;margin:8px 18px 0}' +
        '.pl-empty{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:12px;opacity:.85;text-align:center;padding:20px}' +
        '@media (max-width:800px){.pl-body{flex-direction:column}.pl-side{width:auto;border-left:none;border-top:1px solid rgba(var(--center-channel-color-rgb,29,29,27),.12);max-height:45%}}';

    function Style() { return h('style', null, CSS); }

    // ---------- components ----------
    function ShiftForm(props) {
        var init = props.shift || {};
        var _a = useState(init.label || ''), label = _a[0], setLabel = _a[1];
        var _b = useState(init.start || ''), start = _b[0], setStart = _b[1];
        var _c = useState(init.end || ''), end = _c[0], setEnd = _c[1];
        var _d = useState(init.places || 1), places = _d[0], setPlaces = _d[1];
        var _e = useState('none'), repeat = _e[0], setRepeat = _e[1];
        var _f = useState(''), until = _f[0], setUntil = _f[1];
        var editing = Boolean(props.shift);
        var count = editing ? 1 : computeDates(props.date, repeat, until).length;

        function submit(e) {
            e.preventDefault();
            var body = {label: label, start: start, end: end, places: Number(places)};
            if (editing) {
                body.action = 'update';
                body.shift_id = props.shift.id;
            } else {
                body.action = 'create';
                body.dates = computeDates(props.date, repeat, until);
            }
            props.onSubmit(body);
        }

        return h('form', {className: 'pl-form pl-card', onSubmit: submit},
            h('strong', null, editing ? t('editShift') : t('newShift')),
            h('label', null, t('name')),
            h('input', {className: 'pl-in', value: label, placeholder: t('namePh'), maxLength: 40, onChange: function (e) { setLabel(e.target.value); }, autoFocus: true}),
            h('div', {className: 'pl-row'},
                h('div', {style: {flex: 1}}, h('label', null, t('start')), h('input', {className: 'pl-in', type: 'time', value: start, onChange: function (e) { setStart(e.target.value); }})),
                h('div', {style: {flex: 1}}, h('label', null, t('end')), h('input', {className: 'pl-in', type: 'time', value: end, disabled: !start, onChange: function (e) { setEnd(e.target.value); }}))
            ),
            h('label', null, t('people')),
            h('input', {className: 'pl-in', type: 'number', min: 1, max: 10, value: places, onChange: function (e) { setPlaces(e.target.value); }}),
            !editing && h('label', null, t('repeat')),
            !editing && h('select', {className: 'pl-in', value: repeat, onChange: function (e) { setRepeat(e.target.value); }},
                h('option', {value: 'none'}, t('rNone')),
                h('option', {value: 'daily'}, t('rDaily')),
                h('option', {value: 'weekdays'}, t('rWeekdays')),
                h('option', {value: 'weekly'}, t('rWeekly', wdLong()[parseDate(props.date).getDay()]))
            ),
            !editing && repeat !== 'none' && h('label', null, t('until')),
            !editing && repeat !== 'none' && h('input', {className: 'pl-in', type: 'date', min: props.date, value: until, onChange: function (e) { setUntil(e.target.value); }}),
            h('div', {className: 'pl-row'},
                h('button', {className: 'pl-btn pl-primary', type: 'submit', disabled: props.busy || (repeat !== 'none' && !until)},
                    editing ? t('save') : (count > 1 ? t('createN', count) : t('create'))),
                h('button', {className: 'pl-btn', type: 'button', onClick: props.onCancel}, t('cancel'))
            )
        );
    }

    function ShiftCard(props) {
        var s = props.shift, data = props.data, act = props.act;
        var _a = useState(''), pick = _a[0], setPick = _a[1];
        var mine = s.users.indexOf(data.me) >= 0;
        var full = s.users.length >= s.places;
        var free = s.places - s.users.length;
        var name = function (id) { var u = data.users[id]; return u ? u.name : '?'; };
        var offers = s.offers || [];
        var offerOf = function (id) { for (var i = 0; i < offers.length; i++) { if (offers[i].from === id) { return offers[i]; } } return null; };
        var swaps = data.swaps && s.date >= data.today;
        var myOffer = offerOf(data.me);

        return h('div', {className: 'pl-card' + (mine ? ' mine' : '')},
            h('div', {className: 'pl-row'},
                h('strong', null, s.label),
                timeLabel(s) && h('span', {className: 'pl-muted'}, timeLabel(s)),
                h('span', {className: 'pl-sp'}),
                s.locked && h('span', {title: t('locked')}, '🔒'),
                h('span', {className: 'pl-muted'}, s.users.length + '/' + s.places)
            ),
            h('div', {className: 'pl-row'},
                s.users.map(function (id) {
                    var o = offerOf(id);
                    var tag = o ? (o.to ? ' ⏳→ ' + name(o.to) : ' 🔄') : '';
                    return h('span', {key: id, className: 'pl-person', title: o ? (o.to ? t('swapPending') : t('swapOffered')) : ''}, name(id) + tag,
                        data.admin && h('button', {title: t('remove'), onClick: function () { act({action: 'remove', shift_id: s.id, user_id: id}); }}, '×'));
                }),
                free > 0 && h('span', {className: 'pl-muted'}, (free > 1 ? t('freeN', free) : t('free1')))
            ),
            h('div', {className: 'pl-row'},
                !mine && !full && !s.locked && h('button', {className: 'pl-btn pl-primary', disabled: props.busy, onClick: function () { act({action: 'take', shift_id: s.id}); }}, t('take')),
                mine && !s.locked && h('button', {className: 'pl-btn', disabled: props.busy, onClick: function () { act({action: 'release', shift_id: s.id}); }}, t('release')),
                s.locked && !data.admin && h('span', {className: 'pl-muted'}, t('lockedByAdmin')),
                mine && swaps && !myOffer && h('button', {className: 'pl-btn', disabled: props.busy, onClick: function () { act({action: 'offer', shift_id: s.id}); }}, t('offer')),
                mine && swaps && myOffer && !myOffer.to && h('button', {className: 'pl-btn', disabled: props.busy, onClick: function () { act({action: 'cancel_offer', shift_id: s.id}); }}, t('cancelOffer')),
                mine && myOffer && myOffer.to && h('span', {className: 'pl-muted'}, t('waitApproval', name(myOffer.to)))
            ),
            swaps && !mine && offers.filter(function (o) { return !o.to; }).map(function (o) {
                return h('div', {key: 'c' + o.from, className: 'pl-row'},
                    h('button', {className: 'pl-btn pl-primary', disabled: props.busy, onClick: function () { act({action: 'claim', shift_id: s.id, from_user: o.from}); }}, t('claim', name(o.from))));
            }),
            data.admin && offers.filter(function (o) { return o.to; }).map(function (o) {
                return h('div', {key: 'a' + o.from, className: 'pl-row'},
                    h('span', {className: 'pl-muted'}, t('replaces', name(o.to), name(o.from))),
                    h('button', {className: 'pl-btn pl-primary', disabled: props.busy, onClick: function () { act({action: 'approve', shift_id: s.id, from_user: o.from}); }}, t('approve')),
                    h('button', {className: 'pl-btn pl-danger', disabled: props.busy, onClick: function () { act({action: 'refuse', shift_id: s.id, from_user: o.from}); }}, t('refuse')));
            }),
            data.admin && h('div', {className: 'pl-row'},
                h('button', {className: 'pl-btn', disabled: props.busy, onClick: function () { act({action: s.locked ? 'unlock' : 'lock', shift_id: s.id}); }}, s.locked ? t('unlock') : t('lock')),
                h('button', {className: 'pl-btn', onClick: function () { props.onEdit(s); }}, t('edit')),
                h('button', {className: 'pl-btn pl-danger', disabled: props.busy, onClick: function () {
                    if (window.confirm(t('delConfirm'))) { act({action: 'delete', shift_id: s.id}); }
                }}, t('del'))
            ),
            data.admin && !full && h('div', {className: 'pl-row'},
                h('select', {className: 'pl-in', style: {flex: 1}, value: pick, onChange: function (e) { setPick(e.target.value); }},
                    h('option', {value: ''}, t('signUpPh')),
                    data.members.filter(function (m) { return s.users.indexOf(m.id) < 0; }).map(function (m) {
                        return h('option', {key: m.id, value: m.id}, m.name + ' (@' + m.username + ')');
                    })
                ),
                h('button', {className: 'pl-btn', disabled: !pick || props.busy, onClick: function () { act({action: 'assign', shift_id: s.id, user_id: pick}); setPick(''); }}, t('signUp'))
            )
        );
    }

    function Planning(props) {
        var channelId = props.channelId;
        var today = ymd(new Date());
        var _a = useState(function () { var d = new Date(); return new Date(d.getFullYear(), d.getMonth(), 1); }), cursor = _a[0], setCursor = _a[1];
        var _b = useState(null), data = _b[0], setData = _b[1];
        var _c = useState(''), error = _c[0], setError = _c[1];
        var _d = useState(false), busy = _d[0], setBusy = _d[1];
        var _e = useState(today), selected = _e[0], setSelected = _e[1];
        var _f = useState(null), form = _f[0], setForm = _f[1]; // null | 'new' | shift
        var month = ym(cursor);

        var load = useCallback(function () {
            setError('');
            return api('GET', '/month?channel_id=' + channelId + '&month=' + month)
                .then(setData)
                .catch(function (e) { setError(e.message); });
        }, [channelId, month]);

        useEffect(function () { setData(null); load(); }, [load]);

        function act(body) {
            setBusy(true);
            setError('');
            body.channel_id = channelId;
            body.month = body.dates ? body.dates[0].slice(0, 7) : month;
            return api('POST', '/action', body)
                .then(function (d) { if (body.month === month) { setData(d); } else { load(); } setForm(null); })
                .catch(function (e) { setError(e.message); })
                .then(function () { setBusy(false); });
        }

        function shift(delta) {
            setCursor(new Date(cursor.getFullYear(), cursor.getMonth() + delta, 1));
            setForm(null);
        }

        // Build a Monday-first 6-week grid.
        var first = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
        var offset = (first.getDay() + 6) % 7;
        var cells = [];
        for (var i = 0; i < 42; i++) {
            cells.push(new Date(cursor.getFullYear(), cursor.getMonth(), 1 - offset + i));
        }
        if (cells[35].getMonth() !== cursor.getMonth()) { cells = cells.slice(0, 35); }

        var byDate = {};
        if (data && data.shifts) {
            data.shifts.forEach(function (s) { (byDate[s.date] = byDate[s.date] || []).push(s); });
        }
        var selShifts = byDate[selected] || [];
        var selInMonth = selected.slice(0, 7) === month;
        var anyUnlocked = data && data.shifts.some(function (s) { return !s.locked; });

        var head = h('div', {className: 'pl-head'},
            h('span', {className: 'pl-title'}, '📅 ' + t('title') + ' · ' + months()[cursor.getMonth()].charAt(0).toUpperCase() + months()[cursor.getMonth()].slice(1) + ' ' + cursor.getFullYear()),
            h('button', {className: 'pl-btn', onClick: function () { shift(-1); }, title: t('prevMonth')}, '‹'),
            h('button', {className: 'pl-btn', onClick: function () { var d = new Date(); setCursor(new Date(d.getFullYear(), d.getMonth(), 1)); setSelected(today); }}, t('today')),
            h('button', {className: 'pl-btn', onClick: function () { shift(1); }, title: t('nextMonth')}, '›'),
            data && data.enabled && data.admin && data.shifts.length > 0 && h('button', {className: 'pl-btn', disabled: busy, onClick: function () {
                var a = anyUnlocked ? 'lock_month' : 'unlock_month';
                if (window.confirm(anyUnlocked ? t('lockMonthQ') : t('unlockMonthQ'))) { act({action: a}); }
            }}, anyUnlocked ? t('lockMonth') : t('unlockMonth')),
            data && data.enabled && data.admin && h('button', {className: 'pl-btn pl-danger', disabled: busy, onClick: function () {
                if (window.confirm(t('disableQ'))) { act({action: 'disable'}); }
            }}, t('disable')),
            h('button', {className: 'pl-btn pl-x', onClick: props.onClose, title: t('close')}, '×')
        );

        var body;
        if (!data) {
            body = h('div', {className: 'pl-empty'}, error ? null : t('loading'));
        } else if (!data.enabled) {
            body = h('div', {className: 'pl-empty'},
                h('div', {style: {fontSize: 40}}, '📅'),
                h('div', null, t('notEnabled')),
                data.admin ?
                    h('button', {className: 'pl-btn pl-primary', disabled: busy, onClick: function () { act({action: 'enable'}); }}, t('enableHere')) :
                    h('div', {className: 'pl-muted'}, t('askAdmin'))
            );
        } else {
            body = h('div', {className: 'pl-body'},
                h('div', {className: 'pl-cal'},
                    h('div', {className: 'pl-wd'}, DICT.wdShort[lang()].map(function (w) { return h('div', {key: w}, w); })),
                    h('div', {className: 'pl-grid'}, cells.map(function (d) {
                        var key = ymd(d);
                        var list = key.slice(0, 7) === month ? (byDate[key] || []) : [];
                        var cls = 'pl-day' + (d.getMonth() !== cursor.getMonth() ? ' out' : '') + (key === today ? ' today' : '') + (key === selected ? ' sel' : '');
                        return h('div', {key: key, className: cls, onClick: function () {
                            if (d.getMonth() !== cursor.getMonth()) { setCursor(new Date(d.getFullYear(), d.getMonth(), 1)); }
                            setSelected(key);
                            setForm(null);
                        }},
                        h('span', {className: 'pl-num'}, d.getDate()),
                        list.map(function (s) {
                            var mine = s.users.indexOf(data.me) >= 0;
                            var full = s.users.length >= s.places;
                            var c = 'pl-chip ' + (mine ? 'mine' : (full ? 'full' : 'open'));
                            var txt = (s.locked ? '🔒 ' : '') + ((s.offers || []).some(function (o) { return !o.to; }) ? '🔄 ' : '') + (s.start ? s.start + ' ' : '') + s.label + ' ' + s.users.length + '/' + s.places;
                            var who = s.users.map(function (id) { return (data.users[id] || {}).name; }).join(', ');
                            return h('div', {key: s.id, className: c, title: txt + (who ? ' — ' + who : '')}, txt);
                        }));
                    }))
                ),
                h('div', {className: 'pl-side'},
                    h('h3', null, longDate(selected)),
                    !selInMonth && h('div', {className: 'pl-muted'}, t('loading')),
                    selInMonth && selShifts.length === 0 && !form && h('div', {className: 'pl-muted'}, data.admin ? t('noShiftAdmin') : t('noShift')),
                    selInMonth && selShifts.map(function (s) {
                        if (form && form !== 'new' && form.id === s.id) {
                            return h(ShiftForm, {key: s.id, date: selected, shift: s, busy: busy, onSubmit: act, onCancel: function () { setForm(null); }});
                        }
                        return h(ShiftCard, {key: s.id, shift: s, data: data, act: act, busy: busy, onEdit: setForm});
                    }),
                    data.admin && selInMonth && (form === 'new' ?
                        h(ShiftForm, {date: selected, busy: busy, onSubmit: act, onCancel: function () { setForm(null); }}) :
                        h('button', {className: 'pl-btn', onClick: function () { setForm('new'); }}, t('addShift')))
                )
            );
        }

        return h('div', {className: 'pl-modal', onClick: function (e) { e.stopPropagation(); }},
            head,
            error && h('div', {className: 'pl-err'}, error),
            body
        );
    }

    function Root() {
        var _a = useState(openState), st = _a[0], setSt = _a[1];
        useEffect(function () {
            listeners.push(setSt);
            function onKey(e) { if (e.key === 'Escape' && openState.channelId) { setOpen(null); } }
            document.addEventListener('keydown', onKey);
            return function () {
                listeners = listeners.filter(function (l) { return l !== setSt; });
                document.removeEventListener('keydown', onKey);
            };
        }, []);
        if (!st.channelId) { return null; }
        var close = function () { setOpen(null); };
        return h('div', {className: 'pl-overlay', onClick: close},
            h(Style),
            h(Planning, {key: st.channelId, channelId: st.channelId, onClose: close})
        );
    }

    function CalendarIcon() {
        return h('svg', {width: 18, height: 18, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round'},
            h('rect', {x: 3, y: 4, width: 18, height: 18, rx: 2}),
            h('line', {x1: 16, y1: 2, x2: 16, y2: 6}),
            h('line', {x1: 8, y1: 2, x2: 8, y2: 6}),
            h('line', {x1: 3, y1: 10, x2: 21, y2: 10})
        );
    }

    function PluginClass() {}
    PluginClass.prototype.initialize = function (registry, store) {
        STORE = store;
        function currentChannel() {
            try { return store.getState().entities.channels.currentChannelId; } catch (e) { return null; }
        }
        registry.registerRootComponent(Root);
        registry.registerChannelHeaderButtonAction(
            h(CalendarIcon),
            function (channel) { setOpen((channel && channel.id) || currentChannel()); },
            t('title'),
            t('tooltip')
        );
        registry.registerSlashCommandWillBePostedHook(function (message, args) {
            if (message.trim().toLowerCase() === '/planning') {
                setOpen((args && args.channel_id) || currentChannel());
                return Promise.resolve({});
            }
            return Promise.resolve({message: message, args: args});
        });
    };

    window.registerPlugin(PLUGIN_ID, new PluginClass());
})();
