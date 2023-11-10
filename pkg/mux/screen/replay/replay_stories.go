//go:build stories
// +build stories

package replay

import (
	"context"

	"github.com/cfoust/cy/pkg/bind"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/mux"
	"github.com/cfoust/cy/pkg/sessions"
	"github.com/cfoust/cy/pkg/stories"
	"github.com/cfoust/cy/pkg/taro"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xo/terminfo"
)

func createTestSession() []sessions.Event {
	return sessions.NewSimulator().
		Add(
			"\033[20h", // CRLF -- why is this everywhere?
			geom.DEFAULT_SIZE,
			"test string please ignore",
		).
		Term(terminfo.ClearScreen).
		Add("take two").
		Term(terminfo.ClearScreen).
		Add("test").
		Events()
}

func createStory(ctx context.Context, events []sessions.Event, msgs ...interface{}) mux.Screen {
	replay := New(ctx, events, bind.NewBindScope())

	var realMsg tea.Msg
	for _, msg := range msgs {
		realMsg = msg
		switch msg := msg.(type) {
		case ActionType:
			realMsg = ActionEvent{Type: msg}
		case string:
			keyMsgs := taro.KeysToMsg(msg)
			if len(keyMsgs) == 1 {
				realMsg = keyMsgs[0]
			}
		}
		replay.Send(realMsg)
	}

	return replay
}

var SearchTimeForward stories.Story = func(ctx context.Context) mux.Screen {
	replay := createStory(
		ctx,
		createTestSession(),
		ActionSearchForward,
	)

	return replay
}

func init() {
	config := stories.Config{
		Size: geom.DEFAULT_SIZE,
	}
	stories.Register("search-time-forward", SearchTimeForward, config)
}
