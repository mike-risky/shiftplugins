// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Smafe

package main

import (
	"regexp"
	"testing"
)

var verbRe = regexp.MustCompile(`%[sd]`)

func TestTranslations(t *testing.T) {
	for k, m := range msgs {
		n := len(verbRe.FindAllString(m["en"], -1))
		for _, l := range langs {
			s, ok := m[l]
			if !ok || s == "" {
				t.Errorf("%s: missing %s", k, l)
				continue
			}
			if got := verbRe.FindAllString(s, -1); len(got) != n {
				t.Errorf("%s/%s: %d verbs, en has %d", k, l, len(got), n)
			} else {
				for i, v := range verbRe.FindAllString(m["en"], -1) {
					if got[i] != v {
						t.Errorf("%s/%s: verb order differs", k, l)
					}
				}
			}
		}
	}
}
