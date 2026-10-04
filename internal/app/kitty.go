package app

import (
	"reflect"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// KittyFilter intercepts Kitty keyboard protocol CSI u sequences that
// BubbleTea doesn't recognise (they arrive as unexported unknownCSISequenceMsg)
// and converts them to standard BubbleTea key messages or custom messages.
//
// Terminals that do not support the Kitty protocol never emit these sequences,
// so the filter is a no-op in that case.
func KittyFilter(_ tea.Model, msg tea.Msg) tea.Msg {
	// unknownCSISequenceMsg is unexported []byte – detect via reflection.
	rv := reflect.ValueOf(msg)
	if rv.Kind() != reflect.Slice || rv.Type().Elem().Kind() != reflect.Uint8 {
		return msg
	}
	raw := rv.Bytes()
	if len(raw) < 4 || raw[0] != '\x1b' || raw[1] != '[' {
		return msg
	}
	// Only handle CSI … u  sequences (the Kitty CSI u encoding).
	if raw[len(raw)-1] != 'u' {
		return msg
	}
	params := string(raw[2 : len(raw)-1]) // strip ESC [ and u
	if converted := convertCSIU(params); converted != nil {
		return converted
	}
	return msg
}

// convertCSIU parses a CSI u parameter string and returns a BubbleTea message,
// or nil if the sequence is not one we need to handle.
//
// Format: codepoint(;modifier(:event_type)?)?
func convertCSIU(params string) tea.Msg {
	parts := strings.SplitN(params, ";", 2)
	cp, err := strconv.Atoi(strings.SplitN(parts[0], ":", 2)[0])
	if err != nil {
		return nil
	}

	modifier := 1 // default: no modifier bits
	if len(parts) > 1 {
		modStr := strings.SplitN(parts[1], ":", 2)[0]
		if m, e := strconv.Atoi(modStr); e == nil {
			modifier = m
		}
	}

	// Modifier-only keys → ShiftPressMsg
	// Left Shift = 57441, Right Shift = 57447
	if cp == 57441 || cp == 57447 {
		return ShiftPressMsg{}
	}

	// Decode modifier bits (encoded as 1 + bits).
	shift := (modifier-1)&0x1 != 0
	alt := (modifier-1)&0x2 != 0

	// Try to convert to a BubbleTea key.
	if k, ok := kittyToKey(cp, shift); ok {
		k.Alt = alt
		return tea.KeyMsg(k)
	}
	return nil
}

// kittyToKey maps a Kitty protocol codepoint (+ shift flag) to a BubbleTea Key.
func kittyToKey(cp int, shift bool) (tea.Key, bool) {
	if shift {
		if kt, ok := kittyShiftMap[cp]; ok {
			return tea.Key{Type: kt}, true
		}
	}
	if kt, ok := kittyBaseMap[cp]; ok {
		return tea.Key{Type: kt}, true
	}
	return tea.Key{}, false
}

// kittyBaseMap maps Kitty codepoints to BubbleTea KeyTypes (no Shift).
var kittyBaseMap = map[int]tea.KeyType{
	// Cursor / navigation
	57350: tea.KeyLeft,
	57351: tea.KeyRight,
	57352: tea.KeyUp,
	57353: tea.KeyDown,
	57354: tea.KeyPgUp,
	57355: tea.KeyPgDown,
	57356: tea.KeyHome,
	57357: tea.KeyEnd,
	57348: tea.KeyInsert,
	57349: tea.KeyDelete,
	// Function keys
	57364: tea.KeyF1,
	57365: tea.KeyF2,
	57366: tea.KeyF3,
	57367: tea.KeyF4,
	57368: tea.KeyF5,
	57369: tea.KeyF6,
	57370: tea.KeyF7,
	57371: tea.KeyF8,
	57372: tea.KeyF9,
	57373: tea.KeyF10,
	57374: tea.KeyF11,
	57375: tea.KeyF12,
	// Tab
	9: tea.KeyTab,
}

// kittyShiftMap maps Kitty codepoints to the BubbleTea shifted KeyType.
var kittyShiftMap = map[int]tea.KeyType{
	// Shift + cursor
	57350: tea.KeyShiftLeft,
	57351: tea.KeyShiftRight,
	57352: tea.KeyShiftUp,
	57353: tea.KeyShiftDown,
	57356: tea.KeyShiftHome,
	57357: tea.KeyShiftEnd,
	// Shift + F1‑F8  →  F13‑F20  (the BubbleTea convention)
	57364: tea.KeyF13,
	57365: tea.KeyF14,
	57366: tea.KeyF15,
	57367: tea.KeyF16,
	57368: tea.KeyF17,
	57369: tea.KeyF18,
	57370: tea.KeyF19,
	57371: tea.KeyF20,
	// Shift + Tab
	9: tea.KeyShiftTab,
}
