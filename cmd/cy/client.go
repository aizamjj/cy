package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cfoust/cy/pkg/cy"
	"github.com/cfoust/cy/pkg/geom"
	P "github.com/cfoust/cy/pkg/io/protocol"
	"github.com/cfoust/cy/pkg/io/ws"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

const (
	ENOENT       = "no such file or directory"
	ECONNREFUSED = "connection refused"
)

type ClientIO struct {
	conn cy.Connection
}

func (c *ClientIO) Write(p []byte) (n int, err error) {
	err = c.conn.Send(P.InputMessage{
		Data: p,
	})

	return len(p), err
}

var _ io.Writer = (*ClientIO)(nil)

// Do some sanity checks on a shell string.
func checkShell(shell string) bool {
	if len(shell) == 0 || shell[0] != '/' {
		return false
	}

	/**
	TODO(cfoust): 07/22/23
	if (areshell(shell))
		return (0);
	if (access(shell, X_OK) != 0)
		return (0);
	**/

	return true
}

func getShell() string {
	env := os.Getenv("SHELL")
	if checkShell(env) {
		return env
	}

	/**
	TODO(cfoust): 07/22/23
	pw = getpwuid(getuid());
	if (pw != NULL && checkshell(pw->pw_shell))
		return (pw->pw_shell);
	**/

	return "/bin/bash"
}

/**
if ((s = getenv("VISUAL")) != NULL || (s = getenv("EDITOR")) != NULL) {
	options_set_string(global_options, "editor", 0, "%s", s);
}
**/

func buildHandshake() (*P.HandshakeMessage, error) {
	columns, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	output := termenv.NewOutput(os.Stdout)

	return &P.HandshakeMessage{
		TERM:   os.Getenv("TERM"),
		SHELL:  getShell(),
		EDITOR: os.Getenv("EDITOR"),
		Size: geom.Size{
			R: rows,
			C: columns,
		},
		Profile: output.Profile,
	}, nil
}

func poll(conn cy.Connection) error {
	handshake, err := buildHandshake()
	if err != nil {
		return err
	}

	conn.Send(*handshake)

	writer := ClientIO{
		conn: conn,
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	go func() { _, _ = io.Copy(&writer, os.Stdin) }()

	// Handle window size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		currentRows := 0
		currentCols := 0

		for {
			select {
			case <-context.Background().Done():
				return
			case <-ch:
				columns, rows, err := term.GetSize(int(os.Stdin.Fd()))
				if err != nil {
					log.Error().Err(err).Msg("failed to get terminal dimensions")
					return
				}

				if columns == currentCols && rows == currentRows {
					continue
				}

				conn.Send(P.SizeMessage{
					Rows:    rows,
					Columns: columns,
				})

				currentCols = columns
				currentRows = rows
			}
		}
	}()
	ch <- syscall.SIGWINCH
	defer func() { signal.Stop(ch); close(ch) }()

	events := conn.Receive()
	for {
		select {
		case <-conn.Ctx().Done():
			return nil
		case packet := <-events:
			if packet.Error != nil {
				return packet.Error
			}

			if msg, ok := packet.Contents.(*P.OutputMessage); ok {
				os.Stdout.Write(msg.Data)
			}
		}
	}
}

func connect(socketPath string) (cy.Connection, error) {
	// mimics client_connect() in tmux's client.c
	var lock *os.File = nil

	defer func() {
		if lock != nil {
			lock.Close()
		}
	}()

	locked := false
	for {
		conn, err := ws.Connect(context.Background(), P.Protocol, socketPath)
		if err == nil {
			return conn, nil
		}

		message := err.Error()
		if !strings.Contains(message, ENOENT) && !strings.Contains(message, ECONNREFUSED) {
			return nil, err
		}

		if !locked {
			lock, err = getLock(socketPath)
			if err != nil {
				if err == ErrorLockFailed {
					continue
				}

				lock = nil

				return nil, err
			}

			/*
			 * FROM TMUX:
			 * Always retry at least once, even if we got the lock,
			 * because another client could have taken the lock,
			 * started the server and released the lock between our
			 * connect() and flock().
			 */
			locked = true
			continue
		}

		if err := os.Remove(socketPath); err != nil && !strings.Contains(err.Error(), ENOENT) {
			return nil, err
		}

		// Now we can start the server
		err = startServer(socketPath)
		if err != nil {
			return nil, err
		}

		time.Sleep(50 * time.Millisecond)
	}
}
