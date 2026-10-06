package main

// Party roster layout references A2Tools; see THIRD_PARTY.md.
import (
	"encoding/binary"
	"unicode/utf8"
)

type playerStats struct {
	GearScore   uint32 `json:"gearScore"`
	CombatPower uint64 `json:"combatPower"`
}

// Parse bounded roster records, joining by name only when the name is unique.
// Account IDs in this packet are not combat entity IDs.
func parseRosterStats(p []byte) (map[string]playerStats, bool) {
	if len(p) < 6 {
		return nil, false
	}
	o := 4
	n := int(p[o])
	o++
	if n < 1 || n > 40 || o+n+18 > len(p) || !utf8.Valid(p[o:o+n]) {
		return nil, false
	}
	o += n
	size := int(p[o])
	o++
	if size < 1 || size > 12 {
		return nil, false
	}
	o += 17 // dungeon, padding, leader account ID, padding
	count, rest, ok := readVar(p[o:])
	if !ok || count < 1 || count > uint32(size) {
		return nil, false
	}
	o = len(p) - len(rest)
	result := map[string]playerStats{}
	duplicate := map[string]bool{}
	for index := uint32(0); index < count; index++ {
		if o+11 > len(p) {
			return nil, false
		}
		slot := p[o+1]
		server := binary.LittleEndian.Uint16(p[o+8:])
		n = int(p[o+10])
		if n == 0 && p[o] == 0 {
			return result, len(result) > 0
		}
		if slot < 1 || int(slot) > size || server == 0 || server > 9999 || n < 1 || n > 40 || o+11+n+12 > len(p) {
			return nil, false
		}
		nameBytes := p[o+11 : o+11+n]
		if !utf8.Valid(nameBytes) {
			return nil, false
		}
		name := string(nameBytes)
		o += 11 + n
		level := binary.LittleEndian.Uint32(p[o+4:])
		gear := binary.LittleEndian.Uint32(p[o+8:])
		o += 12
		if level < 1 || level > 200 || gear > 1000000 {
			return nil, false
		}
		anchor := -1
		for a := o; a <= o+10 && a+2 <= len(p); a++ {
			if binary.LittleEndian.Uint16(p[a:]) == server {
				anchor = a
				break
			}
		}
		if anchor < 0 || anchor+13 > len(p) {
			return nil, false
		}
		power := binary.LittleEndian.Uint64(p[anchor+5:])
		o = anchor + 13
		if power > 100000000 {
			return nil, false
		}
		if _, exists := result[name]; exists {
			duplicate[name] = true
			delete(result, name)
		} else if !duplicate[name] {
			result[name] = playerStats{gear, power}
		}
		if index+1 == count {
			return result, true
		}
		next := -1
		for a := o; a <= o+32 && a+11 <= len(p); a++ {
			world := binary.LittleEndian.Uint16(p[a+8:])
			length := int(p[a+10])
			if p[a+1] == slot+1 && world > 0 && world <= 9999 && length <= 40 && a+11+length <= len(p) && utf8.Valid(p[a+11:a+11+length]) {
				next = a
				break
			}
		}
		if next < 0 {
			return nil, false
		}
		o = next
	}
	return nil, false
}

func (s *scope) observeRosterStats(m aMessage) {
	// The roster may be embedded in a larger decoded server frame.
	p := m.payload
	for at := 0; at < len(p); at++ {
		body := []byte(nil)
		if at == 0 && m.opcode == 0x9702 {
			body = p
		} else if at+2 < len(p) && p[at] == 2 && p[at+1] == 0x97 {
			body = p[at+2:]
		}
		if body == nil {
			continue
		}
		stats, ok := parseRosterStats(body)
		if !ok {
			continue
		}
		s.stats = stats
		s.classRevision++ // Publish metadata even when no new damage arrives.
		return
	}
}
