// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import "testing"

func TestParseDate(t *testing.T) {
	for in, want := range map[string]string{"15.12.2026": "2026-12-15", "15.12.26": "2026-12-15", "2026-12-15": "2026-12-15", "1/2/2027": "2027-02-01", "31.02.2026": "", "abc": ""} {
		got, ok := parseDate(in)
		if (want == "") == ok || got != want {
			t.Errorf("%s: %q %v", in, got, ok)
		}
	}
	if countDays("2026-12-14", "2026-12-20", PartFull, false) != 5 || countDays("2026-12-14", "2026-12-20", PartFull, true) != 7 || countDays("2026-12-14", "2026-12-14", PartAM, false) != 0.5 || countDays("2026-12-19", "2026-12-20", PartFull, false) != 0 {
		t.Error("countDays")
	}
	if rangeLabel("fr", "2026-12-14", "2026-12-18", PartFull) != "du lun 14.12.2026 au ven 18.12.2026" {
		t.Error(rangeLabel("fr", "2026-12-14", "2026-12-18", PartFull))
	}
	if rangeLabel("en", "2026-12-14", "2026-12-14", PartAM) != "Monday Dec 14, 2026 (morning)" || daysLabel("de", 0.5) != "0,5 Tag" || rangeLabel("it", "2026-12-14", "2026-12-18", PartFull) != "dal lun 14/12/2026 al ven 18/12/2026" {
		t.Error(rangeLabel("en", "2026-12-14", "2026-12-14", PartAM), daysLabel("de", 0.5), rangeLabel("it", "2026-12-14", "2026-12-18", PartFull))
	}
	for k, m := range catalog {
		for _, l := range langs {
			if m[l] == "" {
				t.Errorf("missing %s %s", k, l)
			}
		}
	}
}
