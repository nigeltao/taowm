package main

import (
	"log"

	xp "github.com/BurntSushi/xgb/xproto"
)

type orientation int

const (
	noOrientation orientation = iota
	horizontal
	vertical
)

type traversal int

const (
	next traversal = iota
	prev
)

type listing int

const (
	listNone listing = iota
	listWindows
	listWorkspaces
)

// offscreenXY is the most negative X/Y co-ordinate.
const offscreenXY = -1 << 15

func contains(r xp.Rectangle, x, y int16) bool {
	return r.X <= x && x <= r.X+int16(r.Width) &&
		r.Y <= y && y <= r.Y+int16(r.Height)
}

func screenContaining(x, y int16) *screen {
	for _, s := range screens {
		if contains(s.rect, x, y) {
			return s
		}
	}
	return screens[0]
}

var (
	screens        []*screen
	dummyWorkspace workspace // The anchor of a doubly-linked list of workspaces.
)

func init() {
	dummyWorkspace.link[next] = &dummyWorkspace
	dummyWorkspace.link[prev] = &dummyWorkspace
}

func findWindow(predicate func(*window) bool) *window {
	for k := dummyWorkspace.link[next]; k != &dummyWorkspace; k = k.link[next] {
		for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
			if predicate(w) {
				return w
			}
		}
	}
	return nil
}

type screen struct {
	workspace *workspace
	rect      xp.Rectangle
}

type workspace struct {
	link         [2]*workspace
	screen       *screen
	focusedFrame *frame
	mainFrame    frame
	dummyWindow  window // The anchor of a doubly-linked list of windows.
	fullscreen   bool
	listing      listing
	list         []interface{}
	index        int
}

type frame struct {
	parent      *frame
	prevSibling *frame
	nextSibling *frame
	firstChild  *frame
	lastChild   *frame
	orientation orientation
	workspace   *workspace
	window      *window
	rect        xp.Rectangle
}

type window struct {
	frame           *frame
	link            [2]*window
	transientFor    *window
	xWin            xp.Window
	rect            xp.Rectangle
	name            string
	offscreenSeqNum uint32
	hasTransientFor bool
	seen            bool
	selected        bool
	wmDeleteWindow  bool
	wmTakeFocus     bool
}

func (s *screen) repaint() {
	check(xp.ClearAreaChecked(xConn, true, desktopXWin,
		s.rect.X, s.rect.Y, s.rect.Width+1, s.rect.Height+1))
}

func newWorkspace(rect xp.Rectangle, previous *workspace) *workspace {
	k := &workspace{
		mainFrame: frame{
			rect: rect,
		},
		index: -1,
	}
	k.mainFrame.workspace = k
	k.dummyWindow.link[next] = &k.dummyWindow
	k.dummyWindow.link[prev] = &k.dummyWindow
	k.focusedFrame = &k.mainFrame

	k.link[next] = previous.link[next]
	k.link[prev] = previous
	k.link[next].link[prev] = k
	k.link[prev].link[next] = k

	k.mainFrame.split(horizontal)
	return k
}

func makeLists() {
	for _, s := range screens {
		if s.workspace.listing != listNone {
			s.workspace.makeList()
		}
	}
}

func (k *workspace) makeList() {
	switch k.listing {
	case listWindows:
		k.list = k.makeWindowList()
	case listWorkspaces:
		k.list = k.makeWorkspaceList()
	default:
		k.list = nil
	}
	k.index = -1
	if len(k.list) != 0 {
		if p, err := xp.QueryPointer(xConn, rootXWin).Reply(); err != nil {
			log.Println(err)
		} else {
			k.index = k.indexForPoint(p.RootX, p.RootY)
		}
	}
	k.configure()
	k.screen.repaint()
}

func (k *workspace) makeWindowList() (list []interface{}) {
	for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
		// TODO: listen instead of poll for name changes.
		w.name = w.property(atomWMName)
		if w.name == "" {
			w.name = "?"
		}
		list = append(list, w)
	}
	return list
}

func (k *workspace) makeWorkspaceList() (list []interface{}) {
	for k := dummyWorkspace.link[next]; k != &dummyWorkspace; k = k.link[next] {
		list = append(list, k)
		list = append(list, k.makeWindowList()...)
	}
	return list
}

func (k *workspace) indexForPoint(rootX, rootY int16) int {
	r := k.focusedFrame.rect
	if k.fullscreen || k.listing == listWorkspaces {
		r = k.mainFrame.rect
	}
	x := int(rootX - r.X)
	y := int(rootY - r.Y)
	if x <= 0 || int(r.Width) <= x || y <= 0 || int(r.Height) <= y {
		return -1
	}
	i := int(y/fontHeight) - 1
	if i < 0 || len(k.list) <= i {
		return -1
	}
	if k.listing == listWorkspaces {
		for ; i >= 0; i-- {
			if _, ok := k.list[i].(*workspace); ok {
				return i
			}
		}
	}
	return i
}

func (k *workspace) configure() {
	for w := k.dummyWindow.link[next]; w != &k.dummyWindow; w = w.link[next] {
		w.configure()
	}
}

func (k *workspace) drawFrameBorders() {
	if k.fullscreen || k.listing == listWorkspaces {
		return
	}
	setForeground(colorUnfocused)
	rects := k.mainFrame.appendRectangles(nil)
	check(xp.PolyRectangleChecked(xConn, xp.Drawable(desktopXWin), desktopXGC, rects))
	setForeground(colorFocused)
	k.focusedFrame.drawBorder()
}

func (k *workspace) focusFrame(f *frame) {
	if f == nil {
		return
	}
	if k.focusedFrame != f {
		if !k.fullscreen && k.listing != listWorkspaces {
			setForeground(colorUnfocused)
			k.focusedFrame.drawBorder()
			setForeground(colorFocused)
			f.drawBorder()
		}
		k.focusedFrame = f
	}
	focus(f.window)
}

func (k *workspace) frameContaining(x, y int16) *frame {
	if k.fullscreen || k.listing == listWorkspaces || contains(k.focusedFrame.rect, x, y) {
		return k.focusedFrame
	}
	return k.mainFrame.frameContaining(x, y)
}

func (k *workspace) layout() {
	if k.screen != nil {
		k.mainFrame.rect = k.screen.rect
	} else {
		k.mainFrame.rect = xp.Rectangle{X: offscreenXY, Y: offscreenXY, Width: 256, Height: 256}
	}
	k.mainFrame.layout()
}

func (f *frame) frameContaining(x, y int16) *frame {
	if contains(f.rect, x, y) {
		if f.firstChild == nil {
			return f
		}
		for c := f.firstChild; c != nil; c = c.nextSibling {
			if g := c.frameContaining(x, y); g != nil {
				return g
			}
		}
	}
	return nil
}

func (f *frame) firstDescendent() *frame {
	for f.firstChild != nil {
		f = f.firstChild
	}
	return f
}

func (f *frame) lastDescendent() *frame {
	for f.lastChild != nil {
		f = f.lastChild
	}
	return f
}

func (f *frame) firstEmptyFrame() *frame {
	if f.firstChild != nil {
		for c := f.firstChild; c != nil; c = c.nextSibling {
			if ret := c.firstEmptyFrame(); ret != nil {
				return ret
			}
		}
	} else if f.window == nil {
		return f
	}
	return nil
}

func (f *frame) numChildren() (n int) {
	for c := f.firstChild; c != nil; c = c.nextSibling {
		n++
	}
	return n
}

func (f *frame) appendRectangles(r []xp.Rectangle) []xp.Rectangle {
	if f.firstChild != nil {
		for c := f.firstChild; c != nil; c = c.nextSibling {
			r = c.appendRectangles(r)
		}
		return r
	}
	return append(r, f.rect)
}

func (f *frame) split(o orientation) {
	if f.parent != nil && f.parent.orientation == o {
		g := &frame{
			parent:      f.parent,
			workspace:   f.workspace,
			prevSibling: f,
			nextSibling: f.nextSibling,
		}
		if f.nextSibling != nil {
			f.nextSibling.prevSibling = g
		} else {
			f.parent.lastChild = g
		}
		f.nextSibling = g
		if f.window != nil {
			f.workspace.focusedFrame = g
		}
		f.parent.layout()
		return
	}

	f.orientation = o
	f.firstChild = &frame{
		parent:    f,
		workspace: f.workspace,
	}
	f.lastChild = &frame{
		parent:    f,
		workspace: f.workspace,
	}
	f.firstChild.nextSibling = f.lastChild
	f.lastChild.prevSibling = f.firstChild
	if f.workspace.focusedFrame == f {
		f.workspace.focusedFrame = f.firstChild
	}
	if w := f.window; w != nil {
		f.window = nil
		f.firstChild.window = w
		w.frame = f.firstChild
	}
	f.layout()
}

func (f *frame) layout() {
	if f.orientation == noOrientation {
		if f.window != nil {
			f.window.configure()
		}
		return
	}
	i, n := 0, f.numChildren()
	for c := f.firstChild; c != nil; i, c = i+1, c.nextSibling {
		c.rect = f.rect
		switch f.orientation {
		case horizontal:
			i0 := (i + 0) * int(f.rect.Width) / n
			i1 := (i + 1) * int(f.rect.Width) / n
			c.rect.X += int16(i0)
			c.rect.Width = uint16(i1 - i0)
		case vertical:
			i0 := (i + 0) * int(f.rect.Height) / n
			i1 := (i + 1) * int(f.rect.Height) / n
			c.rect.Y += int16(i0)
			c.rect.Height = uint16(i1 - i0)
		}
		c.layout()
	}
}

func (f *frame) traverse(t traversal) *frame {
	if f.parent == nil {
		return f
	}
	if t == next && f.nextSibling != nil {
		return f.nextSibling.firstDescendent()
	}
	if t == prev && f.prevSibling != nil {
		return f.prevSibling.lastDescendent()
	}
	f, from := f.parent, f
	for {
		switch from {
		case f.parent:
			if f.firstChild == nil {
				return f
			}
			if t == next {
				f, from = f.firstChild, f
			} else {
				f, from = f.lastChild, f
			}
		case f.firstChild:
			if f.prevSibling != nil {
				f, from = f.prevSibling, f
			} else if f.parent != nil {
				f, from = f.parent, f
			} else {
				f, from = f.lastChild, f
			}
		case f.lastChild:
			if f.nextSibling != nil {
				f, from = f.nextSibling, f
			} else if f.parent != nil {
				f, from = f.parent, f
			} else {
				f, from = f.firstChild, f
			}
		case f.prevSibling:
			return f.firstDescendent()
		case f.nextSibling:
			return f.lastDescendent()
		}
	}
	panic("unreachable")
}

func (f *frame) drawBorder() {
	check(xp.PolyRectangleChecked(xConn, xp.Drawable(desktopXWin), desktopXGC,
		[]xp.Rectangle{f.rect}))
}

var nextOffscreenSeqNum uint32 = 1

func (w *window) property(a xp.Atom) string {
	p, err := xp.GetProperty(xConn, false, w.xWin, a, xp.GetPropertyTypeAny, 0, 1<<32-1).Reply()
	if err != nil {
		log.Println(err)
	}
	return string(p.Value)
}

func (w *window) configure() {
	mask, values := uint16(0), []uint32(nil)
	r := xp.Rectangle{X: offscreenXY, Y: offscreenXY, Width: w.rect.Width, Height: w.rect.Height}
	if w.frame != nil && w.frame.workspace.screen != nil {
		k := w.frame.workspace
		if k.listing == listWorkspaces ||
			(k.listing == listWindows && k.focusedFrame == w.frame) {
			// No-op; r is offscreen.
		} else if k.fullscreen {
			if k.focusedFrame == w.frame {
				r.X = k.mainFrame.rect.X
				r.Y = k.mainFrame.rect.Y
				r.Width = k.mainFrame.rect.Width + 1
				r.Height = k.mainFrame.rect.Height + 1
			}
		} else {
			r.X = w.frame.rect.X + 2
			r.Y = w.frame.rect.Y + 2
			r.Width = w.frame.rect.Width - 3
			r.Height = w.frame.rect.Height - 3
		}
	}
	if w.seen && w.rect == r {
		return
	}
	w.rect = r
	if r.X != offscreenXY {
		w.seen = true
		mask = xp.ConfigWindowX |
			xp.ConfigWindowY |
			xp.ConfigWindowWidth |
			xp.ConfigWindowHeight |
			xp.ConfigWindowBorderWidth
		values = []uint32{
			uint32(uint16(r.X)),
			uint32(uint16(r.Y)),
			uint32(r.Width),
			uint32(r.Height),
			0,
		}
	} else {
		w.offscreenSeqNum = nextOffscreenSeqNum
		nextOffscreenSeqNum++
		mask = xp.ConfigWindowX | xp.ConfigWindowY
		values = []uint32{
			uint32(uint16(r.X)),
			uint32(uint16(r.Y)),
		}
	}
	check(xp.ConfigureWindowChecked(xConn, w.xWin, mask, values))
}
