package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/cfoust/cy/pkg/cy"
)

func getContext() (socket string, id int, ok bool) {
	context, ok := os.LookupEnv(cy.CONTEXT_ENV)
	if !ok {
		return "", 0, false
	}

	match := cy.CONTEXT_REGEX.FindStringSubmatch(context)
	if match == nil {
		return "", 0, false
	}

	socket = match[cy.CONTEXT_REGEX.SubexpIndex("socket")]
	id, _ = strconv.Atoi(match[cy.CONTEXT_REGEX.SubexpIndex("id")])
	ok = true
	return
}

// execCommand is the entrypoint for the exec command.
func execCommand() error {
	if CLI.Exec.Command == "" && CLI.Exec.File == "" {
		return fmt.Errorf("no Janet code provided")
	}

	var err error
	var source, cwd string
	var code []byte

	cwd, err = os.Getwd()
	if err != nil {
		return err
	}

	if CLI.Exec.Command != "" {
		source = "<unknown>"
		code = []byte(CLI.Exec.Command)
	} else if CLI.Exec.File == "-" {
		source = "<stdin>"
		code, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %s", err)
		}
	} else {
		source = CLI.Exec.File
		code, err = ioutil.ReadFile(CLI.Exec.File)
		if err != nil {
			return fmt.Errorf("failed to read from %s: %s", CLI.Exec.File, err)
		}
	}

	socket, id, ok := getContext()
	if !ok {
		socket = CLI.Socket
	}

	socketPath, err := getSocketPath(socket)
	if err != nil {
		return err
	}

	var conn Connection
	conn, err = connect(socketPath, false)
	if err != nil {
		return err
	}

	response, err := RPC[RPCExecArgs, RPCExecResponse](
		conn, "exec", RPCExecArgs{
			Source: source,
			Code:   code,
			Node:   id,
			Dir:    cwd,
			JSON:   CLI.Exec.JSON,
		},
	)
	if err != nil || len(response.Data) == 0 {
		return err
	}

	_, err = os.Stdout.Write(response.Data)
	return err
}
