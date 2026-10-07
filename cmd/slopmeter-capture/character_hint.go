package main

import (
	"github.com/nuriland/a2kit/game"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validCharacterHint(name string) bool {
	if len(name) > 72 || !utf8.ValidString(name) {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func (s *scope) setCharacterHint(name string) {
	if !validCharacterHint(name) {
		return
	}
	name = strings.TrimSpace(name)
	if name == s.characterHint {
		return
	}
	if s.hintedSelf {
		s.self = 0
		s.name = ""
		s.members = nil
		s.classes = nil
		s.stats = nil
		s.hintedSelf = false
	}
	s.characterHint = name
	if s.self == 0 && name != "" {
		var match game.Entity
		for id, known := range s.names {
			if known == name {
				if match != 0 {
					return
				}
				match = id
			}
		}
		if match != 0 {
			s.self = match
			s.name = name
			s.hintedSelf = true
		}
	}
}
func (s *scope) observeCharacterHint(event game.Event) {
	p, ok := event.(game.Player)
	if !ok {
		return
	}
	if p.Self {
		s.hintedSelf = false
		if p.Name != "" {
			s.characterHint = p.Name
		}
		return
	}
	if s.characterHint == "" || p.Name != s.characterHint || p.Entity == 0 {
		return
	}
	// A duplicate name cannot safely establish a session identity.
	for id, name := range s.names {
		if id != p.Entity && name == p.Name {
			if s.hintedSelf {
				s.self = 0
				s.name = ""
				s.hintedSelf = false
			}
			return
		}
	}
	if s.self == 0 {
		s.self = p.Entity
		s.name = p.Name
		s.hintedSelf = true
	}
}
