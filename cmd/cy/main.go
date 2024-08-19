package main

import (
	"fmt"
	"os"

	"github.com/cfoust/cy/pkg/cy"
	"github.com/cfoust/cy/pkg/version"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog/log"
)

var CLI struct {
	Socket string `help:"Specify the name of the socket." name:"socket-name" optional:"" short:"L" default:"default"`

	Version bool `help:"Print version information and exit." short:"v"`

	Exec struct {
		Command string `help:"Provide Janet code as a string argument." name:"command" short:"c" optional:"" default:""`
		Format  string `name:"format" optional:"" enum:"raw,json,janet" short:"f" default:"raw" help:"Set the desired output format."`
		File    string `arg:"" optional:"" help:"Provide a file containing Janet code." type:"existingfile"`
	} `cmd:"" help:"Execute Janet code on the cy server."`

	Connect struct {
		CPU   string `help:"Save a CPU performance report to the given path." name:"perf-file" optional:"" default:""`
		Trace string `help:"Save a trace report to the given path." name:"trace-file" optional:"" default:""`
	} `cmd:"" default:"1" help:"Connect to the cy server, starting one if necessary."`
}

func writeError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("cy"),
		kong.Description("the time traveling terminal multiplexer"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	if CLI.Version {
		fmt.Printf(
			"cy %s (commit %s)\n",
			version.Version,
			version.GitCommit,
		)
		fmt.Printf(
			"built %s\n",
			version.BuildTime,
		)
		os.Exit(0)
	}

	if !cy.SOCKET_REGEX.MatchString(CLI.Socket) {
		log.Fatal().Msg("invalid socket name, the socket name must be alphanumeric")
	}

	switch ctx.Command() {
	case "exec":
		fallthrough
	case "exec <file>":
		err := execCommand()
		if err != nil {
			writeError(err)
		}
	case "connect":
		err := connectCommand()
		if err != nil {
			writeError(err)
		}
	}
}
