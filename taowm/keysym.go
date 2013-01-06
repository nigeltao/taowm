package main

// These constants come from /usr/include/X11/keysymdef.h.

import (
	xp "github.com/BurntSushi/xgb/xproto"
)

const (
	xkISOLeftTab = 0xfe20
	xkBackspace  = 0xff08
	xkTab        = 0xff09
	xkReturn     = 0xff0d
	xkEscape     = 0xff1b
	xkHome       = 0xff50
	xkLeft       = 0xff51
	xkUp         = 0xff52
	xkRight      = 0xff53
	xkDown       = 0xff54
	xkPageUp     = 0xff55
	xkPageDown   = 0xff56
	xkEnd        = 0xff57
	xkShiftL     = 0xffe1
	xkShiftR     = 0xffe2
	xkControlL   = 0xffe3
	xkControlR   = 0xffe4
	xkCapsLock   = 0xffe5
	xkShiftLock  = 0xffe6
	xkMetaL      = 0xffe7
	xkMetaR      = 0xffe8
	xkAltL       = 0xffe9
	xkAltR       = 0xffea
	xkSuperL     = 0xffeb
	xkSuperR     = 0xffec
	xkHyperL     = 0xffed
	xkHyperR     = 0xffee
	xkDelete     = 0xffff
)

func keysymString(keysym xp.Keysym) string {
	switch keysym {
	case xkShiftL:
		return "ShiftL"
	case xkShiftR:
		return "ShiftR"
	case xkControlL:
		return "ControlL"
	case xkControlR:
		return "ControlR"
	case xkCapsLock:
		return "CapsLock"
	case xkShiftLock:
		return "ShiftLock"
	case xkMetaL:
		return "MetaL"
	case xkMetaR:
		return "MetaR"
	case xkAltL:
		return "AltL"
	case xkAltR:
		return "AltR"
	case xkSuperL:
		return "SuperL"
	case xkSuperR:
		return "SuperR"
	case xkHyperL:
		return "HyperL"
	case xkHyperR:
		return "HyperR"
	}
	return "UnknownKeysym"
}
