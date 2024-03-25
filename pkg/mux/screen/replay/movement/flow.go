package movement

import (
	"github.com/cfoust/cy/pkg/emu"
	"github.com/cfoust/cy/pkg/geom"
)

type flowMovement struct {
	emu.Terminal

	// The location of the viewport in the history of the terminal's main
	// screen. See emu.Root().
	root geom.Vec2

	// The cursor's position relative to the viewport.
	cursor   geom.Vec2
	viewport geom.Size

	// Used to mimic the behavior in text editors wherein moving the cursor
	// up and down "sticks" to a certain column index wherever possible
	desiredCol int
}

var _ Movement = (*flowMovement)(nil)

func (f *flowMovement) ScrollYDelta(delta int) {
	isUp := delta < 0

	// Account for the fact that Flow() returns the root line as well
	if !isUp {
		delta++
	}

	result := f.Flow(geom.Vec2{
		C: f.viewport.C,
		R: delta,
	}, f.root)

	numLines := len(result.Lines)

	// Find the new root
	target := 0
	if !isUp {
		target = numLines - 1
	}

	targetLine := result.Lines[target]
	f.root = geom.Vec2{
		R: targetLine.R,
		C: targetLine.C0,
	}

	newRow := f.cursor.R
	if isUp {
		newRow += numLines
	} else {
		// -1 because we're skipping the root line
		newRow -= numLines - 1
	}

	newRow = geom.Clamp(newRow, 0, f.viewport.R-1)
	f.cursor = geom.Vec2{
		R: newRow,
		C: f.resolveScreenColumn(newRow),
	}
}

func (f *flowMovement) ScrollXDelta(delta int) {
	// no-op in this mode
}

func (f *flowMovement) HandleSeek() {
	f.root = f.Root()

	// First just flow the viewport; if the whole screen fits, do
	// nothing
	result := f.Flow(f.viewport, f.root)
	if result.CursorOK {
		f.cursor = geom.Vec2{
			R: result.Cursor.Y,
			C: result.Cursor.X,
		}
		f.desiredCol = f.cursor.C
		return
	}

	// Flow the screen no matter how big it is
	// By definition, cursor must be OK (we flow all lines)
	result = f.Flow(geom.Vec2{C: f.viewport.C}, f.root)
	f.cursor.C = result.Cursor.X
	f.desiredCol = f.cursor.C
	f.scrollToFlowLine(
		result.Lines[result.Cursor.Y].Root(),
		ScrollPositionCenter,
	)
}

// getFlowLine gets a line on the screen in flow mode. Providing a negative
// `row` returns lines from history.
func (f *flowMovement) getFlowLine(row int) (line emu.ScreenLine, ok bool) {
	if row >= 0 {
		// Include the root line
		row++
	}

	flow := f.Flow(geom.Vec2{
		R: row,
		C: f.viewport.C,
	}, f.root)
	if !flow.OK {
		return
	}

	if len(flow.Lines) < geom.Abs(row) {
		return
	}

	if row < 0 {
		line = flow.Lines[0]
	} else {
		line = flow.Lines[row-1]
	}

	ok = true
	return
}

// getLastFlow returns the last root representing the upper limit for the
// scrollable region the user can reach. Mostly this is the last physical line
// on the screen; it's used primarily to prevent the user from scrolling onto
// blank lines at the end of the terminal screen.
func (f *flowMovement) getLastFlow() int {
	screen := f.Flow(getTerminalSize(f.Terminal), f.Root())
	if len(screen.Lines) == 0 {
		return 0
	}

	// Return the row of the last non-empty physical line
	for row := len(screen.Lines) - 1; row >= 0; row-- {
		if !isLineEmpty(screen.Lines[row].Chars) {
			return screen.Lines[row].Root().R
		}
	}

	return 0
}

type ScrollPosition int

const (
	ScrollPositionTop ScrollPosition = iota
	ScrollPositionCenter
	ScrollPositionBottom
)

func (f *flowMovement) scrollToFlowLine(dest geom.Vec2, position ScrollPosition) {
	if dest.R < 0 || dest.C < 0 {
		return
	}

	if dest.R > f.getLastFlow() {
		return
	}

	// If the line is on the screen, we don't need to scroll
	viewport := f.Flow(f.viewport, f.root)
	for row, line := range viewport.Lines {
		if line.Root() != dest {
			continue
		}

		f.cursor.R = row
		break
	}

	var rows int
	switch position {
	case ScrollPositionCenter:
		rows = f.viewport.R/2 - 1
	case ScrollPositionBottom:
		rows = f.viewport.R - 1
	}

	rows = geom.Max(rows, 0)

	if rows == 0 {
		f.root = dest
		f.cursor.R = 0
		return
	}

	flow := f.Flow(geom.Vec2{
		C: f.viewport.C,
		R: -1 * rows,
	}, dest)
	if !flow.OK {
		return
	}

	lines := flow.Lines
	f.root = lines[0].Root()
	f.cursor.R = len(lines)
}

// Given a point in term space representing a desired cursor position, return
// the best available cursor position. This enables behavior akin to moving up
// and down in a text editor.
func (f *flowMovement) resolveScreenColumn(row int) int {
	result := f.Flow(f.viewport, f.root)
	if !result.OK {
		return 0
	}

	lines := result.Lines
	if row < 0 || row >= len(lines) {
		return 0
	}

	return resolveDesiredColumn(lines[row].Chars, f.desiredCol)
}

func (f *flowMovement) Resize(size geom.Vec2) {
	// TODO(cfoust): 03/25/24
}

func (f *flowMovement) Cursor() geom.Vec2 {
	result := f.Flow(f.viewport, f.root)
	for row, line := range result.Lines {
		if result.Cursor.Y != row {
			continue
		}

		numChars := line.C1 - line.C0
		cursor := geom.Vec2{
			R: line.R,
			C: result.Cursor.X,
		}

		// We need to return the address of a real cell
		// TODO(cfoust): 03/25/24 dual-width chars?
		if result.Cursor.X >= numChars {
			cursor.C = numChars - 1
		}

		return cursor
	}

	return geom.Vec2{}
}

func (f *flowMovement) ReadString(start, end geom.Vec2) (result string) {
	return ""
}
