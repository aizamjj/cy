package replay

import (
	"context"
	"time"

	"github.com/cfoust/cy/pkg/bind"
	"github.com/cfoust/cy/pkg/emu"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/sessions"
	"github.com/cfoust/cy/pkg/sessions/search"
	"github.com/cfoust/cy/pkg/taro"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Replay struct {
	render *taro.Renderer
	binds  *bind.Engine[bind.Action]

	// the size of the terminal
	terminal emu.Terminal

	viewport geom.Size

	isPlaying    bool
	playbackRate int
	currentTime  time.Time

	// The location of Replay in time
	// R: the index of the displayed event in `events`
	// C: the byte within it
	location search.Address
	events   []sessions.Event

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

// Get the glyphs for a row in term space.
func (r *Replay) getLine(row int) emu.Line {
	screen := r.terminal.Screen()
	history := r.terminal.History()

	// Handle out-of-bounds lines
	clamped := geom.Clamp(row, -len(history), r.getTerminalSize().R-1)
	if clamped != row {
		return nil
	}

	var line emu.Line
	if row < 0 {
		line = history[len(history)+row]
	} else {
		line = screen[row]
	}

	return line
}

func (r *Replay) exitSelectionMode() {
	r.isSelectionMode = false
	termCursor := r.getTerminalCursor()
	r.centerPoint(termCursor)
	r.cursor = r.termToViewport(termCursor)
	r.desiredCol = r.cursor.C
}

func (r *Replay) Init() tea.Cmd {
	return textinput.Blink
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
		render:       taro.NewRenderer(),
		events:       events,
		terminal:     emu.New(),
		searchInput:  ti,
		playbackRate: 1,
		binds:        binds,
	}
	m.gotoIndex(-1, -1)
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
