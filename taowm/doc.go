/*
Taowm is The Acutely Opinionated Window Manager. It is a minimalist, keyboard
driven, low distraction, tiling window manager for someone who uses a computer
primarily to run just two GUI programs: a web browser and a terminal emulator.


INSTALLATION

To install taowm:
	1. Install Go (as per http://golang.org/doc/install or get it from
	   your distribution).
	2. Run "go get code.google.com/p/taowm/taowm".

This will install taowm in your $GOPATH, or under $GOROOT/bin if $GOPATH is
empty. Run "go help gopath" to read more about $GOPATH.

Taowm is designed to run from an Xsession session. Add this line to the end of
your ~/.xsession file:
	/path/to/your/taowm
where the path is wherever "go get" or "go install" wrote to. Again, run
"go help gopath" for more information.

Log out and log back in with the "Xsession" option. Some systems, such as
Ubuntu 12.04 "Precise", do not offer an Xsession option by default. To enable
it, create a new file /usr/share/xsessions/custom.desktop that contains:
	[Desktop Entry]
	Name=Xsession
	Exec=/etc/X11/Xsession


USAGE

Taowm starts with each screen divided into two side-by-side frames, outlined in
green. Frames can frame windows, but they can also be empty: closing a frame's
window will not collapse that frame. The frame that contains the mouse pointer
is the focused frame, and its border is brighter than other frames. Its window
(if it contains one) will have the keyboard focus.

Taowm is primarily keyboard driven, and all keyboard shortcuts involve first
holding down the Caps Lock key, similar to how holding down the Control key
followed by the 'N' key, in your web browser, creates a new browser window. The
default Caps Lock behavior, CHANGING ALL TYPED LETTERS TO UPPER CASE, is
disabled.

Caps Lock and the Space key will open a new web browser window. Caps Lock and
the Enter key will open a new terminal emulator window. Caps Lock and the '\'
backslash key will lock the screen. Caps Lock and the Backspace key will close
the window in the focused frame. Caps Lock and the Tab key will cycle through
the frames.

To quit taowm and return to the log in screen, hold down Caps Lock and the
Shift key and hit the Escape key three times in quick succession. Normally,
this will quit immediately. Some programs may ask for something before closing,
such as a file name to write unsaved data to. In this case, taowm will quit in
60 seconds or whenever all such programs have closed, instead of quitting
immediately, and the frame borders will turn red.

If there are more windows than frames, then Caps Lock and the 'D' or 'F' key
will cycle through hidden windows. Caps Lock and a number key like '1', '2',
etc. will move the 1st, 2nd, etc. window to the focused frame. Caps Lock and
the 'A' key will show a list of windows: the one currently in the focused
frame is marked with a '+', other windows in other frames are marked with a
'-', hidden windows that have not been seen yet are marked with an '@', and
hidden windows that have been seen before are unmarked. In particular, newly
created windows will not automatically be shown. Taowm prevents new windows
from popping up and 'stealing' keyboard focus, a problem if the password you
are typing into your terminal emulator accidentally gets written to a chat
window that popped up at the wrong time. Instead, if there isn't an empty frame
to accept a new window, taowm keeps that window hidden (and marked with an '@'
in the window list) until you are ready to deal with it. If there are any such
windows that have not been seen yet, the green frame borders will pulsate to
remind you. Selected windows are also marked with a '#'; selection is described
below.

Caps Lock and the 'G' key will toggle the focused frame in occupying the entire
screen. Caps Lock and Shift and the 'G' key will hide the window in the focused
frame. Caps Lock and the '-' key, the '=' key or Shift and the '+' key will
split the current frame horizontally, vertically, or merge a frame to undo a
frame split respectively.

A screen contains workspaces like a frame contains windows. Caps Lock and the
'T' key will create a new workspace, hiding the current one. Caps Lock and the
'E' or 'R' key will cycle through hidden workspaces. Caps Lock and Shift and
the 'T' key will delete the current workspace, provided that it holds no
windows and there is another hidden workspace to switch to. Caps Lock and the
'Q' key will show a list of workspaces (and their windows). Caps Lock and the
'`' key will cycle through the screens. Caps Lock and the F1 key, F2 key, etc.
will move the 1st, 2nd, etc. workspace to the current screen. Caps Lock and the
'S' key will select a window, or unselect a selected window. More than one
window may be selected at a time. Caps Lock and Shift and the 'S' key will
select or unselect all windows in the current workspace. Caps Lock and the 'W'
key will migrate all selected windows to the current workspace and unselect
them.

Taowm also provides alternative ways to navigate within a program's window.
Caps Lock and the 'H', 'J', 'K' or 'L' keys are equivalent to pressing the
Left, Down, Up or Right arrow keys. Similarly, Caps Lock and the 'Y', 'U',
'B' or 'N' keys are equivalent to Home, Page Up, End or Page Down. The 'I' or
'M' keys are equivalent to a mouse wheel scrolling up or down, and the ',' or
'.' keys are equivalent to the Backspace or Delete keys.

Taowm provides similar shortcuts for other common actions. Caps Lock and the
'O' or 'P' keys will copy or paste, '/' or Shift-and-'?' will open or close a
tab in the current window, 'C' or 'V' will cycle through tabs, 'Z' or 'X' will
zoom in or out. By default, these keys will only work with the google-chrome
web browser and the gnome-terminal terminal emulator. Making these work with
other programs will require some customization.


CUSTOMIZATION

Customizing the keyboard shortcuts, web browser, terminal emulator, colors,
etc., is done by editing config.go and re-compiling (and re-installing): run
"go install code.google.com/p/taowm/taowm".


DEVELOPMENT

When working on taowm, it can be run in a nested X server such as Xephyr. From
the code.google.com/p/taowm/taowm directory under $GOPATH:
	Xephyr :9 2>/dev/null &
	DISPLAY=:9 go run *.go


DISCUSSION

The taowm mailing list is at http://groups.google.com/group/taowm


LEGAL

Taowm is copyright 2013 The Taowm Authors. All rights reserved. Use of this
source code is governed by a BSD-style license that can be found in the
LICENSE file.
*/
package main
