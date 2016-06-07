package main

import (
	"time"

	xp "github.com/BurntSushi/xgb/xproto"
)

const (
	// wmKeysym is the key to trigger taowm actions. For other possible
	// values, such as xkSuperL for the 'Windows' key that is typically
	// between the left Control and Alt keys, see keysym.go.
	wmKeysym = xkCapsLock

	// colorXxx are taowm's text and border colors. We assume 24-bit RGB.
	colorBaseUnfocused  = 0x1f3f1f
	colorBaseFocused    = 0x3f7f3f
	colorPulseUnfocused = 0x3f7f3f
	colorPulseFocused   = 0x7fff7f
	colorQuitUnfocused  = 0x7f1f1f
	colorQuitFocused    = 0xff3f3f

	// fontXxx are the font metrics for X11's default font (fixed).
	// fontHeight1 is the vertical offset for the first line of text.
	fontHeight  = 16
	fontHeight1 = 9
	fontWidth   = 6

	// pulseXxx are the animation durations.
	pulseFrameDuration = 50 * time.Millisecond
	pulseTotalDuration = 1000 * time.Millisecond

	// quitDuration is the grace period, when quitting, for programs to exit
	// cleanly.
	quitDuration = 60 * time.Second
)

// xSettings is the key/value pairs to announce via the XSETTINGS mechanism.
// In particular, these include font and theme configuration parameters picked
// up by GTK+ programs such as gnome-terminal.
//
// The dump_xsettings program from http://code.google.com/p/xsettingsd/ will
// show the XSETTINGS key/value pairs set by other desktop environments such
// as GNOME.
//
// If this array is empty, then taowm will not try to own the XSETTINGS list,
// allowing another program such as gnome-settings-daemon to do so.
var xSettings = [...]struct {
	name  string
	value interface{}
}{
	{"Net/IconThemeName", "Tango"},
	{"Net/ThemeName", "Clearlooks"},
	{"Xft/Antialias", 1},
	{"Xft/DPI", 96 * 1024}, // Hard-code 96 DPI, the same as what gnome-settings-daemon does.
	{"Xft/Hinting", 1},
	{"Xft/HintStyle", "hintslight"},
	{"Xft/RGBA", "none"},
}

const doAudioActions = true

// actions lists the action to be performed for each key press. The do function
// returns whether to pulsate the frames' borders to acknowledge the key press.
//
// The map keys are X11 keysyms as int32s. The unary +/^ means whether the
// shift modifier needs to be absent/present.
var actions = map[int32]struct {
	do  func(*workspace, interface{}) bool
	arg interface{}
}{
	+' ':      {doExec, []string{"google-chrome"}},
	^' ':      {doExec, []string{"google-chrome", "--incognito"}},
	^'|':      {doExec, []string{"gnome-screensaver-command", "-l"}},
	+xkReturn: {doExec, []string{"gnome-terminal"}},
	^xkReturn: {doExec, []string{"dmenu_run", "-nb", "#0f0f0f", "-nf", "#3f7f3f",
		"-sb", "#0f0f0f", "-sf", "#7fff7f", "-l", "10"}},

	+xkAudioLowerVolume: {doAudio, []string{"pactl", "set-sink-volume", "0", "--", "-5%"}},
	+xkAudioRaiseVolume: {doAudio, []string{"pactl", "set-sink-volume", "0", "--", "+5%"}},
	+xkAudioMute:        {doAudio, []string{"pactl", "set-sink-mute", "0", "toggle"}},

	+xkBackspace: {doWindowDelete, nil},
	^xkEscape:    {doQuit, nil},

	+'`':          {doScreen, next},
	^'~':          {doScreen, prev},
	+xkTab:        {doFrame, next},
	^xkISOLeftTab: {doFrame, prev},

	+'q': {doList, listWorkspaces},
	+'w': {doWorkspaceMigrate, nil},
	+'e': {doWorkspace, prev},
	^'E': {doWorkspaceNudge, prev},
	+'r': {doWorkspace, next},
	^'R': {doWorkspaceNudge, next},
	+'t': {doWorkspaceNew, nil},
	^'T': {doWorkspaceDelete, nil},

	+'a': {doList, listWindows},
	+'s': {doWindowSelect, false},
	^'S': {doWindowSelect, true},
	+'d': {doWindow, prev},
	^'D': {doWindowNudge, prev},
	+'f': {doWindow, next},
	^'F': {doWindowNudge, next},
	+'g': {doFullscreen, nil},
	^'G': {doHide, nil},

	+'-': {doSplit, horizontal},
	+'=': {doSplit, vertical},
	^'+': {doMerge, nil},

	+'1': {doWindowN, 0},
	+'2': {doWindowN, 1},
	+'3': {doWindowN, 2},
	+'4': {doWindowN, 3},
	+'5': {doWindowN, 4},
	+'6': {doWindowN, 5},
	+'7': {doWindowN, 6},
	+'8': {doWindowN, 7},
	+'9': {doWindowN, 8},
	+'0': {doWindowN, 9},

	+xkF1:  {doWorkspaceN, 0},
	+xkF2:  {doWorkspaceN, 1},
	+xkF3:  {doWorkspaceN, 2},
	+xkF4:  {doWorkspaceN, 3},
	+xkF5:  {doWorkspaceN, 4},
	+xkF6:  {doWorkspaceN, 5},
	+xkF7:  {doWorkspaceN, 6},
	+xkF8:  {doWorkspaceN, 7},
	+xkF9:  {doWorkspaceN, 8},
	+xkF10: {doWorkspaceN, 9},
	+xkF11: {doWorkspaceN, 10},
	+xkF12: {doWorkspaceN, 11},

	+'i': {doSynthetic, xp.Button(4)},
	^'I': {doSynthetic, xp.Button(4)},
	+'m': {doSynthetic, xp.Button(5)},
	^'M': {doSynthetic, xp.Button(5)},
	+'y': {doSynthetic, xp.Keysym(xkHome)},
	^'Y': {doSynthetic, xp.Keysym(xkHome)},
	+'u': {doSynthetic, xp.Keysym(xkPageUp)},
	^'U': {doSynthetic, xp.Keysym(xkPageUp)},
	+'h': {doSynthetic, xp.Keysym(xkLeft)},
	^'H': {doSynthetic, xp.Keysym(xkLeft)},
	+'j': {doSynthetic, xp.Keysym(xkDown)},
	^'J': {doSynthetic, xp.Keysym(xkDown)},
	+'k': {doSynthetic, xp.Keysym(xkUp)},
	^'K': {doSynthetic, xp.Keysym(xkUp)},
	+'l': {doSynthetic, xp.Keysym(xkRight)},
	^'L': {doSynthetic, xp.Keysym(xkRight)},
	+'b': {doSynthetic, xp.Keysym(xkEnd)},
	^'B': {doSynthetic, xp.Keysym(xkEnd)},
	+'n': {doSynthetic, xp.Keysym(xkPageDown)},
	^'N': {doSynthetic, xp.Keysym(xkPageDown)},
	+',': {doSynthetic, xp.Keysym(xkBackspace)},
	^'<': {doSynthetic, xp.Keysym(xkBackspace)},
	+'.': {doSynthetic, xp.Keysym(xkDelete)},
	^'>': {doSynthetic, xp.Keysym(xkDelete)},

	+'/': {doProgramAction, paTabNew},
	^'?': {doProgramAction, paTabClose},
	+'c': {doProgramAction, paTabPrev},
	+'v': {doProgramAction, paTabNext},
	+'o': {doProgramAction, paCopy},
	^'O': {doProgramAction, paCut},
	+'p': {doProgramAction, paPaste},
	^'P': {doProgramAction, paPasteSpecial},
	+'z': {doProgramAction, paZoomIn},
	^'Z': {doProgramAction, paZoomReset},
	+'x': {doProgramAction, paZoomOut},
}

// programAction is an action for a particular program to invoke, as opposed
// to a window management action or generic left/down/up/right synthetic key.
type programAction int

const (
	paTabNew programAction = iota
	paTabClose
	paTabPrev
	paTabNext
	paCut
	paCopy
	paPaste
	paPasteSpecial
	paZoomIn
	paZoomOut
	paZoomReset
	nProgramActions
)

// programActions defines the program-specific synthetic key combination to
// send to perform a generic program action. For example, the 'copy' action is
// Control-C for some programs and Control-Shift-C for others.
//
// The map keys are based on a window's WM_CLASS. To configure a program that
// isn't listed here, run "xprop | grep WM_CLASS", click on a window from that
// program, and use the first quoted value as the map key here.
var programActions = map[string][nProgramActions]struct {
	state  uint16
	keysym xp.Keysym
}{
	"gnome-terminal-server": {
		paTabNew:       {xp.ModMaskControl | xp.ModMaskShift, 'T'},
		paTabClose:     {xp.ModMaskControl | xp.ModMaskShift, 'W'},
		paTabPrev:      {xp.ModMaskControl, xkPageUp},
		paTabNext:      {xp.ModMaskControl, xkPageDown},
		paCut:          {xp.ModMaskControl | xp.ModMaskShift, 'C'},
		paCopy:         {xp.ModMaskControl | xp.ModMaskShift, 'C'},
		paPaste:        {xp.ModMaskControl | xp.ModMaskShift, 'V'},
		paPasteSpecial: {xp.ModMaskControl | xp.ModMaskShift, 'V'},
		paZoomIn:       {xp.ModMaskControl | xp.ModMaskShift, '+'},
		paZoomOut:      {xp.ModMaskControl, '-'},
		paZoomReset:    {xp.ModMaskControl, '0'},
	},
	"google-chrome": {
		paTabNew:       {xp.ModMaskControl, 't'},
		paTabClose:     {xp.ModMaskControl, 'w'},
		paTabPrev:      {xp.ModMaskControl, xkPageUp},
		paTabNext:      {xp.ModMaskControl, xkPageDown},
		paCut:          {xp.ModMaskControl, 'x'},
		paCopy:         {xp.ModMaskControl, 'c'},
		paPaste:        {xp.ModMaskControl, 'v'},
		paPasteSpecial: {xp.ModMaskControl | xp.ModMaskShift, 'V'},
		paZoomIn:       {xp.ModMaskControl | xp.ModMaskShift, '+'},
		paZoomOut:      {xp.ModMaskControl, '-'},
		paZoomReset:    {xp.ModMaskControl, '0'},
	},
}
