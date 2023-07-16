package cy

import (
	"context"
	"os"
	"time"

	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/janet"
	"github.com/cfoust/cy/pkg/mux/screen/server"
	"github.com/cfoust/cy/pkg/mux/screen/tree"
	"github.com/cfoust/cy/pkg/mux/stream"
	"github.com/cfoust/cy/pkg/util"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type Cy struct {
	util.Lifetime
	deadlock.RWMutex
	janet *janet.VM

	muxServer *server.Server

	// The tree of groups and panes.
	tree    *tree.Tree
	clients []*Client

	log zerolog.Logger
}

// Get the pane that new clients attach to. If there are other clients, we
// attach to the pane of the first other client. Otherwise we attach to the
// first pane we find, depth-first.
func (c *Cy) findInitialPane() tree.Node {
	c.RLock()
	defer c.RUnlock()

	if len(c.clients) > 0 {
		node := c.clients[0].Node()
		if node != nil {
			return node
		}
	}

	leaves := c.tree.Leaves()
	if len(leaves) == 0 {
		return nil
	}

	return leaves[0]
}

func Start(ctx context.Context, configFile string) (*Cy, error) {
	tree := tree.NewTree()

	cy := Cy{
		Lifetime:  util.NewLifetime(ctx),
		tree:      tree,
		muxServer: server.New(),
	}

	tree.Root().NewCmd(
		cy.Ctx(),
		stream.CmdOptions{
			Command: "/bin/bash",
		},
		geom.DEFAULT_SIZE,
	)

	logs := stream.NewReader()
	tree.Root().NewPane(
		cy.Ctx(),
		logs,
		geom.DEFAULT_SIZE,
	)

	consoleWriter := zerolog.ConsoleWriter{Out: logs.Writer(), TimeFormat: time.RFC3339}
	cy.log = log.Output(zerolog.MultiLevelWriter(consoleWriter, os.Stdout))

	vm, err := cy.initJanet(ctx, configFile)
	if err != nil {
		return nil, err
	}

	cy.janet = vm

	return &cy, nil
}
