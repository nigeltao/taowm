package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	xp "github.com/BurntSushi/xgb/xproto"
)

func doExec(_ *workspace, cmd1 interface{}) bool {
	cmd, ok := cmd1.([]string)
	if !ok {
		return false
	}
	if len(cmd) == 0 {
		return false
	}
	go func() {
		c := exec.Command(cmd[0], cmd[1:]...)
		if err := c.Start(); err != nil {
			log.Printf("could not start command %q: %v", cmd, err)
		}
		// Ignore any error from the program itself.
		c.Wait()
	}()
	return false
}

func doAudio(k *workspace, cmd1 interface{}) bool {
	if !doAudioActions {
		return false
	}
	doExec(k, cmd1)
	return true
}

func doScreen(k *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	i := -1
	for j, s := range screens {
		if s.workspace == k {
			i = j
			break
		}
	}
	if i < 0 {
		return true
	}
	if t == next {
		i = (i + 1) % len(screens)
	} else {
		i = (i + len(screens) - 1) % len(screens)
	}
	warpPointerTo(screens[i].workspace.focusedFrame)
	return true
}

func doFrame(k *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	if k.fullscreen || k.listing != listNone {
		return false
	}
	k.focusedFrame = k.focusedFrame.traverse(t)
	warpPointerTo(k.focusedFrame)
	return true
}

func warpPointerTo(f *frame) {
	f.workspace.focusFrame(f)
	check(xp.WarpPointerChecked(xConn, xp.WindowNone, rootXWin, 0, 0, 0, 0,
		f.rect.X+int16(f.rect.Width/2),
		f.rect.Y+int16(f.rect.Height/2),
	))
	makeLists()
}

func doWindow(k *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	f0 := k.focusedFrame
	dummy := &k.dummyWindow
	w0 := f0.window
	if w0 == nil {
		w0 = dummy
	}
	w1 := w0
	for {
		w1 = w1.link[t]
		if w1 == w0 {
			return true
		}
		if w1 == dummy {
			continue
		}
		if w1.frame == nil || k.fullscreen {
			break
		}
	}
	if w0 == dummy {
		w0 = nil
	}
	changeWindow(f0, w0, w1)
	return true
}

func doWindowN(k *workspace, n1 interface{}) bool {
	n, ok := n1.(int)
	if !ok {
		return false
	}
	f0, w0 := k.focusedFrame, k.focusedFrame.window
	w1 := k.dummyWindow.link[next]
	for ; n > 0 && w1 != &k.dummyWindow; n-- {
		w1 = w1.link[next]
	}
	if w1 == &k.dummyWindow || w1 == w0 {
		return true
	}
	changeWindow(f0, w0, w1)
	return true
}

func changeWindow(f0 *frame, w0, w1 *window) {
	if f0 == nil || w1 == nil {
		return
	}
	if w0 != w1 {
		if f1 := w1.frame; f1 != nil {
			if w0 != nil {
				f1.window, w0.frame = w0, f1
			} else {
				f1.window = nil
			}
		} else if w0 != nil {
			w0.frame = nil
		}
		f0.window, w1.frame = w1, f0
	}
	w1.configure()
	if w0 != nil {
		w0.configure()
	}
	focus(w1)
	makeLists()
}

func doWorkspace(k0 *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	k1 := k0
	for {
		k1 = k1.link[t]
		if k1 == k0 {
			return true
		}
		if k1 == &dummyWorkspace {
			continue
		}
		if k1.screen == nil {
			break
		}
	}
	changeWorkspace(k0.screen, k0, k1)
	return true
}

func doWorkspaceN(k0 *workspace, n1 interface{}) bool {
	n, ok := n1.(int)
	if !ok {
		return false
	}
	k1 := dummyWorkspace.link[next]
	for ; n > 0 && k1 != &dummyWorkspace; n-- {
		k1 = k1.link[next]
	}
	if k1 == &dummyWorkspace || k1 == k0 {
		return true
	}
	changeWorkspace(k0.screen, k0, k1)
	return true
}

func doWorkspaceNew(k0 *workspace, _ interface{}) bool {
	s := k0.screen
	changeWorkspace(s, k0, newWorkspace(s.rect, k0))
	return true
}

func changeWorkspace(s0 *screen, k0, k1 *workspace) {
	if s0 == nil || k0 == nil || k1 == nil {
		return
	}
	k0.listing = listNone
	s1 := k1.screen
	if k0 != k1 {
		if s1 != nil {
			s1.workspace, k0.screen = k0, s1
		} else {
			k0.screen = nil
		}
		s0.workspace, k1.screen = k1, s0
	}
	k1.layout()
	k0.layout()
	if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
		log.Println(err)
		k1.focusFrame(k1.mainFrame.firstDescendent())
	} else {
		k1.focusFrame(k1.frameContaining(p.RootX, p.RootY))
	}
	s0.repaint()
	if s1 != nil {
		s1.repaint()
	}
	makeLists()
}

func doWindowDelete(k *workspace, _ interface{}) bool {
	w := k.focusedFrame.window
	if w != nil && w.wmDeleteWindow {
		sendClientMessage(w.xWin, atomWMDeleteWindow)
	}
	return true
}

func doWorkspaceDelete(k0 *workspace, _ interface{}) bool {
	if k0.dummyWindow.link[next] != &k0.dummyWindow {
		// Workspace-delete fails if the workspace contains a window.
		return true
	}
	k1 := k0
	for {
		k1 = k1.link[next]
		if k1 == k0 {
			return true
		}
		if k1 == &dummyWorkspace {
			continue
		}
		if k1.screen == nil {
			break
		}
	}
	k0.link[prev].link[next] = k0.link[next]
	k0.link[next].link[prev] = k0.link[prev]
	s := k0.screen
	s.workspace, k1.screen = k1, s
	*k0 = workspace{}
	k1.layout()
	focus(k1.focusedFrame.window)
	s.repaint()
	makeLists()
	return true
}

func doList(k *workspace, l1 interface{}) bool {
	l, ok := l1.(listing)
	if !ok {
		return false
	}
	if k.listing != l {
		k.listing = l
	} else {
		k.listing = listNone
	}
	k.makeList()
	return false
}

func doWindowNudge(k *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	if w := k.focusedFrame.window; w != nil {
		wn, wp := w.link[next], w.link[prev]
		wn.link[prev] = wp
		wp.link[next] = wn
		if t == next {
			w.link[next] = wn.link[next]
			w.link[prev] = wn
		} else {
			w.link[next] = wp
			w.link[prev] = wp.link[prev]
		}
		w.link[next].link[prev] = w
		w.link[prev].link[next] = w
	}
	makeLists()
	return true
}

func doWorkspaceNudge(k *workspace, t1 interface{}) bool {
	t, ok := t1.(traversal)
	if !ok {
		return false
	}
	kn, kp := k.link[next], k.link[prev]
	kn.link[prev] = kp
	kp.link[next] = kn
	if t == next {
		k.link[next] = kn.link[next]
		k.link[prev] = kn
	} else {
		k.link[next] = kp
		k.link[prev] = kp.link[prev]
	}
	k.link[next].link[prev] = k
	k.link[prev].link[next] = k
	makeLists()
	return true
}

func doWindowSelect(k *workspace, b1 interface{}) bool {
	b, ok := b1.(bool)
	if !ok {
		return false
	}
	if b {
		allSelected := true
		for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
			if !w.selected {
				allSelected = false
				break
			}
		}
		for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
			w.selected = !allSelected
		}
	} else {
		if w := k.focusedFrame.window; w != nil {
			w.selected = !w.selected
		}
	}
	makeLists()
	return true
}

func doWorkspaceMigrate(k *workspace, _ interface{}) bool {
	previous := k.dummyWindow.link[prev]
	if k.focusedFrame.window != nil {
		previous = k.focusedFrame.window
	}
	var migrants []*window
	for k0 := dummyWorkspace.link[next]; k0 != &dummyWorkspace; k0 = k0.link[next] {
		migrants = migrants[:0]
		for w := k0.dummyWindow.link[next]; w != &k0.dummyWindow; w = w.link[next] {
			if !w.selected {
				continue
			}
			w.selected = false
			if k0 == k {
				continue
			}
			migrants = append(migrants, w)
		}
		for _, w := range migrants {
			if f := w.frame; f != nil {
				f.window, w.frame = nil, nil
			}
			wn, wp := w.link[next], w.link[prev]
			wn.link[prev] = wp
			wp.link[next] = wn
			wn, wp = previous.link[next], previous
			wn.link[prev] = w
			wp.link[next] = w
			w.link[next], w.link[prev] = wn, wp
			previous = w

			f := k.focusedFrame
			if f.window != nil {
				f = k.mainFrame.firstEmptyFrame()
			}
			if f != nil {
				f.window, w.frame = w, f
			}
			w.configure()
		}
		if k0.fullscreen && k0.focusedFrame.window == nil {
			if k0.screen != nil {
				doFullscreen(k0, nil)
			} else {
				k0.fullscreen = false
			}
		}
	}
	makeLists()
	return true
}

func doFullscreen(k *workspace, _ interface{}) bool {
	if !k.fullscreen && k.focusedFrame.window == nil {
		return true
	}
	k.fullscreen = !k.fullscreen
	if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
		log.Println(err)
	} else {
		k.focusFrame(k.frameContaining(p.RootX, p.RootY))
	}
	k.configure()
	if k.screen != nil {
		k.screen.repaint()
	}
	return true
}

func doHide(k *workspace, _ interface{}) bool {
	f := k.focusedFrame
	w := f.window
	if w != nil {
		f.window, w.frame = nil, nil
		w.configure()
	}
	if k.fullscreen {
		doFullscreen(k, nil)
	}
	makeLists()
	return true
}

func doMerge(k *workspace, _ interface{}) bool {
	if k.fullscreen || k.listing == listWorkspaces {
		return false
	}
	f := k.focusedFrame
	if f.parent == nil {
		// Merge fails if the frame is the main frame.
		return true
	}
	w := f.window
	if w != nil {
		f.window, w.frame = nil, nil
		w.configure()
	}

	if f.prevSibling != nil {
		f.prevSibling.nextSibling = f.nextSibling
		k.focusedFrame = f.prevSibling.lastDescendent()
	}
	if f.nextSibling != nil {
		f.nextSibling.prevSibling = f.prevSibling
		k.focusedFrame = f.nextSibling.firstDescendent()
	}
	parent := f.parent
	if parent.firstChild == f {
		parent.firstChild = f.nextSibling
	}
	if parent.lastChild == f {
		parent.lastChild = f.prevSibling
	}

	if f.parent.numChildren() == 1 {
		// Hoist the sibling frame into the parent frame.
		sibling := parent.firstChild
		parent.firstChild = sibling.firstChild
		parent.lastChild = sibling.lastChild
		for c := parent.firstChild; c != nil; c = c.nextSibling {
			c.parent = parent
		}
		parent.orientation = sibling.orientation
		if w := sibling.window; w != nil {
			parent.window, w.frame = w, parent
		}
		if k.focusedFrame == sibling {
			k.focusedFrame = parent.firstDescendent()
		}
		*f, *sibling = frame{}, frame{}
	}

	parent.layout()
	finishMergeSplit(k)
	return true
}

func doSplit(k *workspace, o1 interface{}) bool {
	o, ok := o1.(orientation)
	if !ok {
		return false
	}
	if k.fullscreen || k.listing == listWorkspaces {
		return false
	}
	k.focusedFrame.split(o)
	finishMergeSplit(k)
	return true
}

func finishMergeSplit(k *workspace) {
	if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
		log.Println(err)
		k.focusFrame(k.mainFrame.firstDescendent())
	} else {
		k.focusFrame(k.frameContaining(p.RootX, p.RootY))
	}
	k.screen.repaint()
	makeLists()
}

func doProgramAction(k *workspace, pa1 interface{}) bool {
	pa, ok := pa1.(programAction)
	if !ok {
		return false
	}
	w := k.focusedFrame.window
	if w == nil {
		return false
	}
	class := w.property(atomWMClass)
	if i := strings.Index(class, "\x00"); i >= 0 {
		class = class[:i]
	}
	a := programActions[class][pa]
	if a.keysym == 0 {
		return false
	}
	sendSynthetic(w, a.state, a.keysym)
	return true
}

func doSynthetic(k *workspace, buttonOrKeysym interface{}) bool {
	if w := k.focusedFrame.window; w != nil {
		sendSynthetic(w, keyState, buttonOrKeysym)
	}
	return false
}

func sendSynthetic(w *window, state uint16, buttonOrKeysym interface{}) {
	// {Button,Key}{Press,Release}Event types all have the same wire format,
	// except for the first byte (the X11 message type).
	e := xp.KeyPressEvent{
		Time:       eventTime,
		Root:       rootXWin,
		Event:      w.xWin,
		Child:      w.xWin,
		RootX:      keyRootX,
		RootY:      keyRootY,
		EventX:     keyRootX - w.rect.X,
		EventY:     keyRootY - w.rect.Y,
		State:      state,
		SameScreen: true,
	}
	var msg0, msg1 byte
	switch bk := buttonOrKeysym.(type) {
	case xp.Button:
		msg0 = xp.ButtonPress
		msg1 = xp.ButtonRelease
		e.Detail = xp.Keycode(bk)
	case xp.Keysym:
		msg0 = xp.KeyPress
		msg1 = xp.KeyRelease
		keycode, shift := findKeycode(bk)
		if keycode == 0 {
			return
		}
		e.Detail = keycode
		if shift {
			e.State |= xp.KeyButMaskShift
		}
	default:
		return
	}

	b := e.Bytes()
	b[0] = msg0
	check(xp.SendEventChecked(xConn, false, w.xWin, xp.EventMaskNoEvent, string(b)))
	b[0] = msg1
	check(xp.SendEventChecked(xConn, false, w.xWin, xp.EventMaskNoEvent, string(b)))
}

var (
	quitTimes [2]time.Time
	quitIndex int
	quitting  bool
)

func doQuit(_ *workspace, _ interface{}) bool {
	if quitting {
		return false
	}
	now := time.Now()
	since := now.Sub(quitTimes[quitIndex])
	quitTimes[quitIndex] = now
	quitIndex = (quitIndex + 1) % len(quitTimes)
	if since > 5*time.Second {
		return true
	}
	quitting = true

	waiting := false
	for k := dummyWorkspace.link[next]; k != &dummyWorkspace; k = k.link[next] {
		for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
			if w.wmDeleteWindow {
				waiting = true
				sendClientMessage(w.xWin, atomWMDeleteWindow)
			}
		}
	}
	if waiting {
		go func() {
			time.Sleep(quitDuration)
			os.Exit(0)
		}()
	} else {
		os.Exit(0)
	}
	return true
}

func focus(w *window) {
	xWin := desktopXWin
	if w != nil {
		xWin = w.xWin
		if w.wmTakeFocus {
			sendClientMessage(xWin, atomWMTakeFocus)
			return
		}
	}
	check(xp.SetInputFocusChecked(xConn, xp.InputFocusParent, xWin, eventTime))
}
