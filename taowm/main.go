package main

import (
	"log"
	"os"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	xp "github.com/BurntSushi/xgb/xproto"
)

var (
	xConn    *xgb.Conn
	rootXWin xp.Window

	eventTime xp.Timestamp
	keyRootX  int16
	keyRootY  int16
	keyState  uint16

	// proactiveChan carries X operations that happen of the program's
	// own accord, such as animations. These are sent to the main goroutine
	// from other goroutines. In comparison, examples of reactive operations
	// are responding to window creation and key presses.
	proactiveChan = make(chan func())
)

type checker interface {
	Check() error
}

var checkers []checker

func check(c checker) {
	checkers = append(checkers, c)
}

func sendClientMessage(xWin xp.Window, atom xp.Atom) {
	check(xp.SendEventChecked(xConn, false, xWin, xp.EventMaskNoEvent,
		string(xp.ClientMessageEvent{
			Format: 32,
			Window: xWin,
			Type:   atomWMProtocols,
			Data: xp.ClientMessageDataUnionData32New([]uint32{
				uint32(atom),
				uint32(eventTime),
				0,
				0,
				0,
			}),
		}.Bytes()),
	))
}

func handleConfigureRequest(e xp.ConfigureRequestEvent) {
	mask, values := uint16(0), []uint32(nil)
	if w := findWindow(func(w *window) bool { return w.xWin == e.Window }); w != nil {
		cne := xp.ConfigureNotifyEvent{
			Event:  w.xWin,
			Window: w.xWin,
			X:      w.rect.X,
			Y:      w.rect.Y,
			Width:  w.rect.Width,
			Height: w.rect.Height,
		}
		check(xp.SendEventChecked(xConn, false, w.xWin,
			xp.EventMaskStructureNotify, string(cne.Bytes())))
		return
	}
	if e.ValueMask&xp.ConfigWindowX != 0 {
		mask |= xp.ConfigWindowX
		values = append(values, uint32(e.X))
	}
	if e.ValueMask&xp.ConfigWindowY != 0 {
		mask |= xp.ConfigWindowY
		values = append(values, uint32(e.Y))
	}
	if e.ValueMask&xp.ConfigWindowWidth != 0 {
		mask |= xp.ConfigWindowWidth
		values = append(values, uint32(e.Width))
	}
	if e.ValueMask&xp.ConfigWindowHeight != 0 {
		mask |= xp.ConfigWindowHeight
		values = append(values, uint32(e.Height))
	}
	if e.ValueMask&xp.ConfigWindowBorderWidth != 0 {
		mask |= xp.ConfigWindowBorderWidth
		values = append(values, uint32(e.BorderWidth))
	}
	if e.ValueMask&xp.ConfigWindowSibling != 0 {
		mask |= xp.ConfigWindowSibling
		values = append(values, uint32(e.Sibling))
	}
	if e.ValueMask&xp.ConfigWindowStackMode != 0 {
		mask |= xp.ConfigWindowStackMode
		values = append(values, uint32(e.StackMode))
	}
	check(xp.ConfigureWindowChecked(xConn, e.Window, mask, values))
}

func manage(xWin xp.Window, mapRequest bool) {
	callFocus := false
	w := findWindow(func(w *window) bool { return w.xWin == xWin })
	if w == nil {
		wmDeleteWindow, wmTakeFocus := false, false
		if prop, err := xp.GetProperty(xConn, false, xWin, atomWMProtocols,
			xp.GetPropertyTypeAny, 0, 64).Reply(); err != nil {
			log.Println(err)
		} else {
			for v := prop.Value; len(v) >= 4; v = v[4:] {
				switch xp.Atom(u32(v)) {
				case atomWMDeleteWindow:
					wmDeleteWindow = true
				case atomWMTakeFocus:
					wmTakeFocus = true
				}
			}
		}

		transientFor := (*window)(nil)
		if prop, err := xp.GetProperty(xConn, false, xWin, atomWMTransientFor,
			xp.GetPropertyTypeAny, 0, 64).Reply(); err != nil {
			log.Println(err)
		} else if v := prop.Value; len(v) == 4 {
			transientForXWin := xp.Window(u32(v))
			transientFor = findWindow(func(w *window) bool {
				return w.xWin == transientForXWin
			})
		}

		k := screens[0].workspace
		if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
			log.Println(err)
		} else {
			k = screenContaining(p.RootX, p.RootY).workspace
		}
		w = &window{
			transientFor: transientFor,
			xWin:         xWin,
			rect: xp.Rectangle{
				X:      offscreenXY,
				Y:      offscreenXY,
				Width:  1,
				Height: 1,
			},
			wmDeleteWindow: wmDeleteWindow,
			wmTakeFocus:    wmTakeFocus,
		}
		f := k.focusedFrame
		previous := k.dummyWindow.link[prev]
		if transientFor != nil {
			previous = transientFor
		} else if f.window != nil {
			previous = f.window
		}
		w.link[next] = previous.link[next]
		w.link[prev] = previous
		w.link[next].link[prev] = w
		w.link[prev].link[next] = w

		if transientFor != nil && transientFor.frame != nil {
			f = transientFor.frame
			f.window, transientFor.frame = nil, nil
		} else if f.window != nil {
			f = k.mainFrame.firstEmptyFrame()
		}
		if f != nil {
			f.window, w.frame = w, f
			callFocus = f == k.focusedFrame
		} else {
			pulseChan <- time.Now()
		}

		check(xp.ChangeWindowAttributesChecked(xConn, xWin, xp.CwEventMask,
			[]uint32{xp.EventMaskEnterWindow | xp.EventMaskStructureNotify},
		))
		w.configure()
		if transientFor != nil {
			transientFor.hasTransientFor = true
			transientFor.configure()
		}
	}
	if mapRequest {
		check(xp.MapWindowChecked(xConn, xWin))
	}
	if callFocus {
		focus(w)
	}
	makeLists()
	pulseChan <- time.Now()
}

func unmanage(xWin xp.Window) {
	w := findWindow(func(w *window) bool { return w.xWin == xWin })
	if w == nil {
		return
	}
	if quitting && findWindow(func(w *window) bool { return true }) == nil {
		os.Exit(0)
	}
	if w.hasTransientFor {
		for {
			w1 := findWindow(func(w2 *window) bool { return w2.transientFor == w })
			if w1 == nil {
				break
			}
			w1.transientFor = nil
		}
	}
	if f := w.frame; f != nil {
		k := f.workspace
		replacement := (*window)(nil)
		if w.transientFor != nil && w.transientFor.frame == nil {
			replacement = w.transientFor
		} else {
			bestOffscreenSeqNum := uint32(0)
			for w1 := w.link[next]; w1 != w; w1 = w1.link[next] {
				if w1.offscreenSeqNum <= bestOffscreenSeqNum {
					continue
				}
				if k.fullscreen {
					if w1.frame == k.focusedFrame {
						continue
					}
				} else if w1.frame != nil {
					continue
				}
				replacement, bestOffscreenSeqNum = w1, w1.offscreenSeqNum
			}
		}
		if replacement != nil {
			if f0 := replacement.frame; f0 != nil {
				f0.window, replacement.frame = nil, nil
			}
			f.window, replacement.frame = replacement, f
			replacement.configure()
			if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
				log.Println(err)
			} else if contains(f.rect, p.RootX, p.RootY) {
				focus(replacement)
			}
		} else {
			f.window = nil
			if k.fullscreen && f == k.focusedFrame {
				doFullscreen(k, nil)
			}
		}
	}
	w.link[next].link[prev] = w.link[prev]
	w.link[prev].link[next] = w.link[next]
	*w = window{}
	makeLists()
	pulseChan <- time.Now()
}

type xEventOrError struct {
	event xgb.Event
	error xgb.Error
}

func main() {
	var err error
	xConn, err = xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	if err = xinerama.Init(xConn); err != nil {
		log.Fatal(err)
	}
	xSetup := xp.Setup(xConn)
	if len(xSetup.Roots) != 1 {
		log.Fatalf("X setup has unsupported number of roots: %d", len(xSetup.Roots))
	}
	rootXWin = xSetup.Roots[0].Root

	becomeTheWM()
	initAtoms()
	initDesktop(&xSetup.Roots[0])
	initKeyboardMapping()
	initScreens()

	// Manage any existing windows.
	tree, err := xp.QueryTree(xConn, rootXWin).Reply()
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range tree.Children {
		if c == desktopXWin {
			continue
		}
		attrs, err := xp.GetWindowAttributes(xConn, c).Reply()
		if err != nil {
			continue
		}
		if attrs.OverrideRedirect || attrs.MapState == xp.MapStateUnmapped {
			continue
		}
		manage(c, false)
	}

	// Process X events.
	eeChan := make(chan xEventOrError)
	go func() {
		for {
			e, err := xConn.WaitForEvent()
			eeChan <- xEventOrError{e, err}
		}
	}()
	for {
		for i, c := range checkers {
			if err := c.Check(); err != nil {
				log.Println(err)
			}
			checkers[i] = nil
		}
		checkers = checkers[:0]

		select {
		case f := <-proactiveChan:
			f()
		case ee := <-eeChan:
			if ee.error != nil {
				log.Println(ee.error)
				continue
			}
			switch e := ee.event.(type) {
			case xp.ButtonPressEvent:
				eventTime = e.Time
				handleButtonPress(e)
			case xp.ButtonReleaseEvent:
				eventTime = e.Time
			case xp.ClientMessageEvent:
				// No-op.
			case xp.ConfigureNotifyEvent:
				// No-op.
			case xp.ConfigureRequestEvent:
				handleConfigureRequest(e)
			case xp.DestroyNotifyEvent:
				// No-op.
			case xp.EnterNotifyEvent:
				eventTime = e.Time
				handleEnterNotify(e)
			case xp.ExposeEvent:
				handleExpose(e)
			case xp.KeyPressEvent:
				eventTime, keyRootX, keyRootY, keyState = e.Time, e.RootX, e.RootY, e.State
				handleKeyPress(e)
			case xp.KeyReleaseEvent:
				eventTime, keyRootX, keyRootY, keyState = e.Time, 0, 0, 0
			case xp.MapNotifyEvent:
				// No-op.
			case xp.MappingNotifyEvent:
				// No-op.
			case xp.MapRequestEvent:
				manage(e.Window, true)
			case xp.MotionNotifyEvent:
				eventTime = e.Time
				handleMotionNotify(e)
			case xp.UnmapNotifyEvent:
				unmanage(e.Window)
			default:
				log.Printf("unhandled event: %v", ee.event)
			}
		}
	}
}
