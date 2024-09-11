package cmd

import (
	"context"

	"github.com/cfoust/cy/pkg/bind"
	C "github.com/cfoust/cy/pkg/cmd"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/mux/stream"
	"github.com/cfoust/cy/pkg/replay/detect"
	"github.com/cfoust/cy/pkg/replay/replayable"
	"github.com/cfoust/cy/pkg/sessions"
)

func New(
	ctx context.Context,
	options stream.CmdOptions,
	dataDir string,
	timeBinds, copyBinds *bind.BindScope,
) (*replayable.Replayable, error) {
	cmd, err := stream.NewCmd(ctx, options, geom.DEFAULT_SIZE)
	if err != nil {
		return nil, err
	}

	if len(dataDir) == 0 {
		replayable := replayable.New(
			ctx,
			cmd,
			cmd,
			timeBinds,
			copyBinds,
		)
		return replayable, nil
	}

	borgPath, err := sessions.GetFilename(dataDir, options.Directory)
	if err != nil {
		return nil, err
	}

	recorder, err := sessions.NewFileRecorder(ctx, borgPath)
	if err != nil {
		return nil, err
	}

	var r *replayable.Replayable
	handler := func(c detect.Command) {
		// Flush the .borg to disk so that other cy servers can read
		// it if necessary
		recorder.Flush()

		// TODO get path from OSC 7 before getting from cmd
		cwd, _ := cmd.Path()
		r.Publish(C.CommandEvent{
			Command: c,
			Borg:    borgPath,
			Cwd:     cwd,
		})
	}

	r = replayable.New(
		ctx,
		cmd,
		sessions.NewEventStream(cmd, recorder),
		timeBinds,
		copyBinds,
		detect.WithHandler(handler),
	)
	return r, nil
}
