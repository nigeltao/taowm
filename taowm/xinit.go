package main

import (
	"bytes"
	"log"

	"github.com/BurntSushi/xgb/xinerama"
	xp "github.com/BurntSushi/xgb/xproto"
)

var (
	atomWMClass        xp.Atom
	atomWMDeleteWindow xp.Atom
	atomWMName         xp.Atom
	atomWMProtocols    xp.Atom
	atomWMTakeFocus    xp.Atom
	atomWMTransientFor xp.Atom

	desktopXWin   xp.Window
	desktopXGC    xp.Gcontext
	desktopWidth  uint16
	desktopHeight uint16

	keysyms [256][2]xp.Keysym
)

func becomeTheWM() {
	if err := xp.ChangeWindowAttributesChecked(xConn, rootXWin, xp.CwEventMask, []uint32{
		xp.EventMaskButtonPress |
			xp.EventMaskButtonRelease |
			xp.EventMaskPointerMotion |
			xp.EventMaskSubstructureRedirect,
	}).Check(); err != nil {
		if _, ok := err.(xp.AccessError); ok {
			log.Fatal("could not become the window manager. Is another window manager running?")
		}
		log.Fatal(err)
	}
}

func initAtoms() {
	atomWMClass = internAtom("WM_CLASS")
	atomWMDeleteWindow = internAtom("WM_DELETE_WINDOW")
	atomWMName = internAtom("WM_NAME")
	atomWMProtocols = internAtom("WM_PROTOCOLS")
	atomWMTakeFocus = internAtom("WM_TAKE_FOCUS")
	atomWMTransientFor = internAtom("WM_TRANSIENT_FOR")
}

func internAtom(name string) xp.Atom {
	r, err := xp.InternAtom(xConn, false, uint16(len(name)), name).Reply()
	if err != nil {
		log.Fatal(err)
	}
	return r.Atom
}

func initDesktop(xScreen *xp.ScreenInfo) {
	xFont, err := xp.NewFontId(xConn)
	if err != nil {
		log.Fatal(err)
	}
	xCursor, err := xp.NewCursorId(xConn)
	if err != nil {
		log.Fatal(err)
	}
	err = xp.OpenFontChecked(xConn, xFont, uint16(len("cursor")), "cursor").Check()
	if err != nil {
		log.Fatal(err)
	}
	const xcLeftPtr = 68 // XC_left_ptr from cursorfont.h.
	err = xp.CreateGlyphCursorChecked(
		xConn, xCursor, xFont, xFont, xcLeftPtr, xcLeftPtr+1,
		0xffff, 0xffff, 0xffff, 0, 0, 0).Check()
	if err != nil {
		log.Fatal(err)
	}
	err = xp.CloseFontChecked(xConn, xFont).Check()
	if err != nil {
		log.Fatal(err)
	}

	desktopXWin, err = xp.NewWindowId(xConn)
	if err != nil {
		log.Fatal(err)
	}
	desktopXGC, err = xp.NewGcontextId(xConn)
	if err != nil {
		log.Fatal(err)
	}
	desktopWidth = xScreen.WidthInPixels
	desktopHeight = xScreen.HeightInPixels

	if err := xp.CreateWindowChecked(
		xConn, xScreen.RootDepth, desktopXWin, xScreen.Root,
		0, 0, desktopWidth, desktopHeight, 0,
		xp.WindowClassInputOutput,
		xScreen.RootVisual,
		xp.CwOverrideRedirect|xp.CwEventMask,
		[]uint32{
			1,
			xp.EventMaskExposure,
		},
	).Check(); err != nil {
		log.Fatal(err)
	}

	if len(xSettings) != 0 {
		initXSettings()
	}

	if err := xp.ConfigureWindowChecked(
		xConn,
		desktopXWin,
		xp.ConfigWindowStackMode,
		[]uint32{
			xp.StackModeBelow,
		},
	).Check(); err != nil {
		log.Fatal(err)
	}

	if err := xp.ChangeWindowAttributesChecked(
		xConn,
		desktopXWin,
		xp.CwBackPixel|xp.CwCursor,
		[]uint32{
			xScreen.BlackPixel,
			uint32(xCursor),
		},
	).Check(); err != nil {
		log.Fatal(err)
	}

	if err := xp.CreateGCChecked(
		xConn,
		desktopXGC,
		xp.Drawable(xScreen.Root),
		0,
		nil,
	).Check(); err != nil {
		log.Fatal(err)
	}

	if err := xp.MapWindowChecked(xConn, desktopXWin).Check(); err != nil {
		log.Fatal(err)
	}
}

func initKeyboardMapping() {
	const (
		keyLo = 8
		keyHi = 255
	)
	km, err := xp.GetKeyboardMapping(xConn, keyLo, keyHi-keyLo+1).Reply()
	if err != nil {
		log.Fatal(err)
	}
	n := int(km.KeysymsPerKeycode)
	if n < 2 {
		log.Fatalf("too few keysyms per keycode: %d", n)
	}
	wmKeycode := xp.Keycode(0)
	for i := keyLo; i <= keyHi; i++ {
		keysyms[i][0] = km.Keysyms[(i-keyLo)*n+0]
		keysyms[i][1] = km.Keysyms[(i-keyLo)*n+1]
		if keysyms[i][0] == wmKeysym || keysyms[i][1] == wmKeysym {
			wmKeycode = xp.Keycode(i)
		}
	}
	if wmKeycode == 0 {
		log.Fatalf("could not find the window manager key %s", keysymString(wmKeysym))
	}
	if err := xp.GrabKeyChecked(xConn, false, rootXWin, xp.ModMaskAny, wmKeycode,
		xp.GrabModeAsync, xp.GrabModeAsync).Check(); err != nil {
		log.Fatal(err)
	}

	// Disable Caps Lock if it is the wmKeysym.
	if wmKeysym == xkCapsLock {
		// First, remap the Caps_Lock key code to Hyper_R.
		// This is the equivalent of: xmodmap -e "keysym Caps_Lock = Hyper_R".
		i0 := int(wmKeycode-keyLo+0) * n
		i1 := int(wmKeycode-keyLo+1) * n
		newKeysyms := make([]xp.Keysym, n)
		copy(newKeysyms, km.Keysyms[i0:i1])
		for i := range newKeysyms {
			if newKeysyms[i] == xkCapsLock {
				newKeysyms[i] = xkHyperR
			}
		}
		if err := xp.ChangeKeyboardMappingChecked(xConn,
			1, wmKeycode, uint8(n), newKeysyms).Check(); err != nil {
			log.Fatal(err)
		}

		// Second, clear the lock modifier mapping.
		// This is the equivalent of: xmodmap -e "clear lock".
		mm, err := xp.GetModifierMapping(xConn).Reply()
		if err != nil {
			log.Fatal(err)
		}
		masks := [...]int{
			xp.ModMaskShift,
			xp.ModMaskLock,
			xp.ModMaskControl,
			xp.ModMask1,
			xp.ModMask2,
			xp.ModMask3,
			xp.ModMask4,
			xp.ModMask5,
		}
		for i, mask := range masks {
			if len(mm.Keycodes) < (i+1)*int(mm.KeycodesPerModifier) {
				break
			}
			if mask == xp.ModMaskLock {
				for j := 0; j < int(mm.KeycodesPerModifier); j++ {
					mm.Keycodes[i*int(mm.KeycodesPerModifier)+j] = 0
				}
				break
			}
		}
		sm, err := xp.SetModifierMapping(xConn, mm.KeycodesPerModifier, mm.Keycodes).Reply()
		if err != nil {
			log.Fatal(err)
		}
		if sm.Status != xp.MappingStatusSuccess {
			log.Fatal("could not set modifier mapping")
		}
	}
}

func findKeycode(keysym xp.Keysym) (keycode xp.Keycode, shift bool) {
	for i, k := range keysyms {
		if k[0] == keysym {
			return xp.Keycode(i), false
		}
		if k[1] == keysym {
			return xp.Keycode(i), true
		}
	}
	return 0, false
}

func initScreens() {
	xine, err := xinerama.QueryScreens(xConn).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if len(xine.ScreenInfo) > 0 {
		screens = make([]*screen, len(xine.ScreenInfo))
		for i, si := range xine.ScreenInfo {
			screens[i] = &screen{
				rect: xp.Rectangle{
					X:      si.XOrg,
					Y:      si.YOrg,
					Width:  si.Width - 1,
					Height: si.Height - 1,
				},
			}
		}
	} else {
		screens = make([]*screen, 1)
		screens[0] = &screen{
			rect: xp.Rectangle{
				X:      0,
				Y:      0,
				Width:  desktopWidth - 1,
				Height: desktopHeight - 1,
			},
		}
	}
	for _, s := range screens {
		s.workspace = newWorkspace(s)
	}
}

func initXSettings() {
	a0 := internAtom("_XSETTINGS_S0")
	if err := xp.SetSelectionOwnerChecked(xConn, desktopXWin, a0,
		xp.TimeCurrentTime).Check(); err != nil {
		log.Printf("could not set xsettings: %v", err)
		return
	}
	a1 := internAtom("_XSETTINGS_SETTINGS")
	encoded := makeEncodedXSettings()
	if err := xp.ChangePropertyChecked(xConn, xp.PropModeReplace, desktopXWin, a1, a1,
		8, uint32(len(encoded)), encoded).Check(); err != nil {
		log.Printf("could not set xsettings: %v", err)
		return
	}
}

func makeEncodedXSettings() []byte {
	b := new(bytes.Buffer)
	b.WriteString("\x00\x00\x00\x00") // Zero means little-endian.
	b.WriteString("\x00\x00\x00\x00") // Serial number.
	writeUint32(b, uint32(len(xSettings)))
	for _, s := range xSettings {
		switch s.value.(type) {
		case int:
			b.WriteString("\x00\x00")
		case string:
			b.WriteString("\x01\x00")
		default:
			log.Fatalf("unsupported XSettings type %T", s.value)
		}
		writeUint16(b, uint16(len(s.name)))
		b.WriteString(s.name)
		if x := len(s.name) % 4; x != 0 {
			b.WriteString("\x00\x00\x00\x00"[:4-x]) // Padding.
		}
		b.WriteString("\x00\x00\x00\x00") // Serial number.
		switch v := s.value.(type) {
		case int:
			writeUint32(b, uint32(v))
		case string:
			writeUint32(b, uint32(len(v)))
			b.WriteString(v)
			if x := len(v) % 4; x != 0 {
				b.WriteString("\x00\x00\x00\x00"[:4-x]) // Padding.
			}
		}
	}
	return b.Bytes()
}

func writeUint16(b *bytes.Buffer, u uint16) {
	b.WriteByte(byte(u << 0))
	b.WriteByte(byte(u << 8))
}

func writeUint32(b *bytes.Buffer, u uint32) {
	b.WriteByte(byte(u << 0))
	b.WriteByte(byte(u << 8))
	b.WriteByte(byte(u << 16))
	b.WriteByte(byte(u << 24))
}

func u32(b []byte) uint32 {
	return uint32(b[0])<<0 | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}