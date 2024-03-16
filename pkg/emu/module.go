package emu

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/cfoust/cy/pkg/geom"
)

// TODO(cfoust): 05/19/23 combine this with above
const (
	AttrReverse = 1 << iota
	AttrUnderline
	AttrBold
	AttrGfx
	AttrItalic
	AttrBlink
)

const (
	cursorDefault = 1 << iota
	cursorWrapNext
	cursorOrigin
)

// ModeFlag represents various terminal mode states.
type ModeFlag uint32

// Terminal modes
const (
	ModeWrap ModeFlag = 1 << iota
	ModeInsert
	ModeAppKeypad
	ModeAltScreen
	ModeCRLF
	ModeMouseButton
	ModeMouseMotion
	ModeReverse
	ModeKeyboardLock
	ModeHide
	ModeEcho
	ModeAppCursor
	ModeMouseSgr
	Mode8bit
	ModeBlink
	ModeFBlink
	ModeFocus
	ModeMouseX10
	ModeMouseMany
	ModeMouseMask = ModeMouseButton | ModeMouseMotion | ModeMouseX10 | ModeMouseMany
)

// ChangeFlag represents possible state changes of the terminal.
type ChangeFlag uint32

// WriteID represents the unique ID of a single contiguous Write() to the
// terminal.
type WriteID uint32

// Terminal changes to occur in VT.ReadState
const (
	ChangedScreen ChangeFlag = 1 << iota
	ChangedTitle
)

type Glyph struct {
	Char        rune
	Mode        int16
	FG, BG      Color
	Transparent bool
	Write       WriteID
}

func (g Glyph) IsEmpty() bool {
	return g.Char == ' '
}

func (g Glyph) IsDefault() bool {
	return g.Mode&attrBlank != 0
}

func EmptyGlyph() Glyph {
	return Glyph{
		Char: ' ',
		FG:   DefaultFG,
		BG:   DefaultBG,
	}
}

type Line []Glyph

func (l Line) String() (str string) {
	for i := 0; i < len(l); i++ {
		str += string(l[i].Char)
	}

	return str
}

func (l Line) IsWrapped() bool {
	if len(l) == 0 {
		return false
	}

	return l[len(l)-1].Mode == attrWrap
}

func (l Line) Length() int {
	return getLineLength(l)
}

func (l Line) Clone() Line {
	return copyLine(l)
}

type CursorStyle int

const (
	CursorStyleBlock CursorStyle = iota
	CursorStyleSteadyBlock
	CursorStyleUnderline
	CursorStyleBlinkUnderline
	CursorStyleBar
	CursorStyleBlinkBar
)

type Cursor struct {
	Attr  Glyph
	X, Y  int
	State uint8
	Style CursorStyle
}

type Cell struct {
	geom.Vec2
	Glyph
}

// Terminal represents the virtual terminal emulator.
type Terminal interface {
	// View displays the virtual terminal.
	View

	// Parse parses input and writes terminal changes to state.
	Parse(p []byte) (n int)

	// Write does the same as Parse, but locks first.
	io.Writer
}

// View represents the view of the virtual terminal emulator.
type View interface {
	// String dumps the virtual terminal contents.
	fmt.Stringer

	// Size returns the size of the virtual terminal.
	Size() (cols, rows int)

	// Resize changes the size of the virtual terminal.
	Resize(cols, rows int)

	// Mode returns the current terminal mode.
	Mode() ModeFlag

	// Title represents the title of the console window.
	Title() string

	// Cell returns the glyph containing the character code, foreground color, and
	// background color at position (x, y) relative to the top left of the terminal.
	Cell(x, y int) Glyph

	// Cursor returns the current position of the cursor.
	Cursor() Cursor

	// CursorVisible returns the visible state of the cursor.
	CursorVisible() bool

	// Screen gets all of the lines on the screen.
	Screen() []Line

	// History returns the scrollback buffer.
	History() []Line

	// The location in history of the top-left cell of the screen. The `R`
	// field refers to the line in history and the `C` refers to a column
	// in that line that the cell contains. For example, when only one line
	// is in the scrollback buffer and it does not wrap onto the screen,
	// Root() will return [1, 0].
	Root() geom.Vec2

	// Flow returns `count` lines from the perspective of `from` respecting
	// the bounds of the given viewport. A negative `count` will return -1
	// * `count` lines from before where `from` begins. If there are not
	// enough lines before or after `from` to satisfy the request, fewer
	// lines than `count` will be returned.
	//
	// For example, with a history that looks like this:
	// ```
	// abcdefg
	// !abc
	// foobarbaz
	// ```
	// where "!" represents `from`, a `count` of 2 with a viewport width of 3 would
	// return:
	// ```
	// abc
	// foo
	// ```
	//
	// A count of -2 and a viewport width of 3 would return:
	// ```
	// bcd
	// efg
	// ```
	Flow(viewport, root geom.Vec2) ([]FlowLine, geom.Vec2, bool)

	Changes() *Dirty
}

type TerminalOption func(*TerminalInfo)

type TerminalInfo struct {
	w              io.Writer
	cols, rows     int
	disableHistory bool
}

func WithWriter(w io.Writer) TerminalOption {
	return func(info *TerminalInfo) {
		info.w = w
	}
}

func WithSize(size geom.Vec2) TerminalOption {
	return func(info *TerminalInfo) {
		info.cols = size.C
		info.rows = size.R
	}
}

// Providing WithoutHistory disables the scrollback buffer, which drastically
// reduces the amount of memory a Terminal uses.
var WithoutHistory TerminalOption = func(info *TerminalInfo) {
	info.disableHistory = true
}

// New returns a new virtual terminal emulator.
func New(opts ...TerminalOption) Terminal {
	info := TerminalInfo{
		w:    ioutil.Discard,
		cols: geom.DEFAULT_SIZE.C,
		rows: geom.DEFAULT_SIZE.R,
	}
	for _, opt := range opts {
		opt(&info)
	}
	return newTerminal(info)
}
