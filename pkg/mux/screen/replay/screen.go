package replay

import (
	"context"

	"github.com/cfoust/cy/pkg/bind"
	"github.com/cfoust/cy/pkg/emu"
	"github.com/cfoust/cy/pkg/geom"
	P "github.com/cfoust/cy/pkg/io/protocol"
	"github.com/cfoust/cy/pkg/sessions"
	"github.com/cfoust/cy/pkg/sessions/search"
	"github.com/cfoust/cy/pkg/taro"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

type Replay struct {
	render *taro.Renderer
	binds  *bind.Engine[bind.Action]

	// the size of the terminal
	terminal emu.Terminal

	viewport geom.Size

	// the offset of the displayed event in `events`
	index  int
	events []sessions.Event

	// Selection mode occurs when the user moves the cursor or scrolls the
	// window
	isSelectionMode bool

	// The offset of the viewport relative to the top-left corner of the
	// underlying terminal.
	//
	// offset.R is in the range
	// [-1 * number of scrollback lines, min(-(height of terminal - height of viewport), 0)]
	// positive indices mean the viewport is inside of the scrollback buffer
	// negative indices mean the viewport is viewing only part of the terminal's screen
	//
	// offset.C is in the range [0, max(width of terminal - width of viewport, 0)]
	//
	// For example:
	// * offset.R == 1: the viewport shows the first scrollback line
	offset, minOffset, maxOffset geom.Vec2

	// The cursor's position relative to the viewport.
	cursor geom.Vec2
	// Used to emulate the behavior in text editors wherein moving the
	// cursor up and down "sticks" to a certain column index wherever
	// possible
	desiredCol int

	isSearching bool
	isForward   bool
	isWaiting   bool
	searchInput textinput.Model
	matches     []search.SearchResult
}

var _ taro.Model = (*Replay)(nil)

func (r *Replay) quit() (taro.Model, tea.Cmd) {
	return r, tea.Quit
}

// Translate a coordinate in the reference frame of the terminal to a point in
// the viewport.
func (r *Replay) termToViewport(point geom.Vec2) geom.Vec2 {
	return point.Sub(r.offset)
}

func (r *Replay) viewportToTerm(point geom.Vec2) geom.Vec2 {
	return point.Add(r.offset)
}

func (r *Replay) getTerminalCursor() geom.Vec2 {
	cursor := r.terminal.Cursor()
	return geom.Vec2{
		R: cursor.Y,
		C: cursor.X,
	}
}

func (r *Replay) getTerminalSize() geom.Vec2 {
	cols, rows := r.terminal.Size()
	return geom.Vec2{
		R: rows,
		C: cols,
	}
}

func (r *Replay) setViewport(oldViewport, newViewport geom.Size) (taro.Model, tea.Cmd) {
	r.viewport = newViewport
	r.recalculateViewport()

	if r.isSelectionMode {
		r.center(r.cursor)
	} else {
		r.center(r.getTerminalCursor())
	}

	return r, nil
}

func (r *Replay) setOffsetY(offset int) {
	r.offset.R = geom.Clamp(offset, r.minOffset.R, r.maxOffset.R)
}

func (r *Replay) setOffsetX(offset int) {
	r.offset.C = geom.Clamp(offset, r.minOffset.C, r.maxOffset.C)
}

// Center the viewport on a point in the reference frame of the terminal.
func (r *Replay) center(point geom.Vec2) {
	r.setOffsetX(point.C - (r.viewport.C / 2))
	r.setOffsetY(point.R - (r.viewport.R / 2))
}

// Calculate the bounds of `{min,max}Offset` and ensure `offset` falls between them.
func (r *Replay) recalculateViewport() {
	// the cursor and desiredCol are viewport-relative, so we must
	// transform to term space and back
	cursor := r.viewportToTerm(r.cursor)
	desiredCol := r.desiredCol + r.offset.C

	termSize := r.getTerminalSize()
	r.minOffset = geom.Vec2{
		R: -len(r.terminal.History()),
		C: 0, // always, but for clarity
	}
	r.maxOffset = geom.Vec2{
		R: geom.Max(termSize.R-r.viewport.R, 0),
		C: geom.Max(termSize.C-r.viewport.C, 0),
	}
	r.setOffsetY(r.offset.R)
	r.setOffsetX(r.offset.C)

	r.cursor = r.termToViewport(cursor)
	r.desiredCol = desiredCol - r.offset.C
}

// Given a point in term space representing a desired cursor position, return
// the best available cursor position. This enables behavior akin to moving up
// and down in a text editor.
func (r *Replay) getColumn(point geom.Vec2) int {
	var row emu.Line
	screen := r.terminal.Screen()
	history := r.terminal.History()
	if point.R < 0 {
		row = history[len(history)+point.R]
	} else {
		row = screen[point.R]
	}

	occupancy := make([]bool, len(row))
	for i := 0; i < len(row); i++ {
		if row[i].IsEmpty() {
			continue
		}

		// handle wide runes
		r := row[i].Char
		w := runewidth.RuneWidth(r)
		for j := 0; j < w; j++ {
			occupancy[i+j] = true
		}
		i += geom.Max(w-1, 0)
	}

	// desiredCol occupied -> return that col
	if occupancy[point.C] {
		return point.C
	}

	var haveBefore, haveAfter bool
	// check for occupied cells before and after the desired column
	for i := 0; i < len(row); i++ {
		if i == point.C || !occupancy[i] {
			continue
		}

		if i > point.C {
			haveAfter = true
		} else {
			haveBefore = true
		}
	}

	// the line is empty, just go to col 0
	if !haveBefore && !haveAfter {
		return 0
	}

	// point.C is before last non-whitespace and after first
	// non-whitespace: remain in place
	if haveBefore && haveAfter {
		return point.C
	}

	// first non-whitespace is after point.C: last column before first
	// non-whitespace
	if haveAfter && !haveBefore {
		for i := point.C + 1; i < len(row); i-- {
			if occupancy[i] {
				return i - 1
			}
		}

		return point.C
	}

	// last non-whitespace is before point.C: last non-whitespace column
	if haveBefore && !haveAfter {
		for i := point.C; i >= 0; i-- {
			if !row[i].IsEmpty() {
				return i
			}
		}
	}

	return 0
}

func (r *Replay) setScroll(offset int) {
	r.isSelectionMode = true
	before := r.viewportToTerm(r.cursor)
	r.setOffsetY(offset)
	after := r.termToViewport(before)

	// cursor is below viewport; move it to bottom
	if after.R >= r.viewport.R {
		r.cursor.R = geom.Max(r.viewport.R-1, 0)
	} else if after.R < 0 {
		r.cursor.R = 0
	} else {
		r.cursor.R = after.R
	}

	r.cursor.C = r.getColumn(geom.Vec2{
		R: r.cursor.R + r.offset.R,
		C: r.desiredCol,
	})
}

// Move the terminal from event index `from` to `to`.
func (r *Replay) setIndex(index int) {
	numEvents := len(r.events)
	// Allow for negative indices from end of stream
	if index < 0 {
		index = geom.Clamp(numEvents+index, 0, numEvents-1)
	}

	from := geom.Clamp(r.index, 0, numEvents-1)
	to := geom.Clamp(index, 0, numEvents-1)

	if from == to {
		return
	}

	// Don't reapply the change at the current offset
	if to > from && from != 0 {
		from++
	} else if from > to {
		r.terminal = emu.New()
		from = 0
	}

	for i := from; i <= to; i++ {
		event := r.events[i]
		switch e := event.Message.(type) {
		case P.OutputMessage:
			r.terminal.Write(e.Data)
		case P.SizeMessage:
			r.terminal.Resize(
				e.Columns,
				e.Rows,
			)
		}
	}

	r.index = to
	r.recalculateViewport()
	termCursor := r.getTerminalCursor()
	termSize := r.getTerminalSize()

	r.isSelectionMode = false

	// reset scroll offset whenever we move in time
	r.offset.R = 0
	r.offset.C = 0

	// Center the cursor if the viewport is smaller than the terminal's
	// viewport
	if r.viewport.C < termSize.C || r.viewport.R < termSize.R {
		r.center(termCursor)
	}

	r.cursor = r.termToViewport(termCursor)
	r.desiredCol = r.cursor.C
}

func (r *Replay) Init() tea.Cmd {
	return textinput.Blink
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (r *Replay) Update(msg tea.Msg) (taro.Model, tea.Cmd) {
	_, rows := r.terminal.Size()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return r.setViewport(
			r.viewport,
			geom.Size{
				R: msg.Height,
				C: msg.Width,
			},
		)
	case SearchResult:
		r.isSearching = false
		// TODO(cfoust): 10/13/23 handle error
		r.matches = msg.results

		if !msg.isForward {
			reverse(r.matches)
		}

		if len(r.matches) > 0 {
			r.setIndex(r.matches[0].Begin.Index)
		}

		return r, nil
	}

	if r.isSearching {
		switch msg := msg.(type) {
		case Action:
			switch msg.Type {
			case ActionQuit:
				r.isSearching = false
				return r, nil
			}
		case taro.KeyMsg:
			switch msg.Type {
			case taro.KeyEnter:
				value := r.searchInput.Value()
				r.searchInput.Reset()

				isForward := r.isForward
				events := r.events[r.index:]
				if !isForward {
					events = r.events[:r.index]
				}

				r.isWaiting = true

				return r, func() tea.Msg {
					res, err := search.Search(events, value)
					return SearchResult{
						isForward: isForward,
						results:   res,
						err:       err,
					}
				}
			}
		}
		var cmd tea.Cmd
		inputMsg := msg
		if key, ok := msg.(taro.KeyMsg); ok {
			inputMsg = key.ToTea()
		}
		r.searchInput, cmd = r.searchInput.Update(inputMsg)
		return r, cmd
	}

	switch msg := msg.(type) {
	case taro.MouseMsg:
		switch msg.Type {
		case taro.MouseWheelUp:
			r.setScroll(r.offset.R - 1)
		case taro.MouseWheelDown:
			r.setScroll(r.offset.R + 1)
		}
	case taro.KeyMsg:
		// Pass unmatched keys into the binding engine; because of how
		// text input works, :replay bindings have to be activated
		// selectively
		return r, func() tea.Msg {
			r.binds.InputMessage(msg)
			return nil
		}
	case Action:
		switch msg.Type {
		case ActionQuit:
			return r.quit()
		case ActionTimeBeginning:
			r.setIndex(0)
		case ActionTimeEnd:
			r.setIndex(-1)
		case ActionTimeSearchForward, ActionTimeSearchBackward:
			r.isSearching = true
			r.isForward = msg.Type == ActionTimeSearchForward
			r.searchInput.Reset()
		case ActionTimeStepBack:
			r.setIndex(r.index - 1)
		case ActionTimeStepForward:
			r.setIndex(r.index + 1)
		case ActionScrollUpHalf:
			r.setScroll(r.offset.R - (rows / 2))
		case ActionScrollDownHalf:
			r.setScroll(r.offset.R + (rows / 2))
		case ActionScrollUp:
			r.setScroll(r.offset.R - 1)
		case ActionScrollDown:
			r.setScroll(r.offset.R + 1)
		}
	}

	return r, nil
}

func newReplay(
	events []sessions.Event,
	binds *bind.Engine[bind.Action],
) *Replay {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20
	ti.Prompt = ""
	m := &Replay{
		render:      taro.NewRenderer(),
		events:      events,
		terminal:    emu.New(),
		searchInput: ti,
		binds:       binds,
	}
	m.setIndex(-1)
	return m
}

func New(
	ctx context.Context,
	recorder *sessions.Recorder,
	replayBinds *bind.BindScope,
	replayEvents chan<- bind.BindEvent,
) *taro.Program {
	events := recorder.Events()

	engine := bind.NewEngine[bind.Action]()
	engine.SetScopes(replayBinds)
	go engine.Poll(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-engine.Recv():
				if bindEvent, ok := event.(bind.BindEvent); ok {
					replayEvents <- bindEvent
				}
			}
		}
	}()

	return taro.New(ctx, newReplay(events, engine))
}
