package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
	"vcr-cli/pkg/api"
	"vcr-cli/pkg/cmdutil"
	"vcr-cli/pkg/format"
	"vcr-cli/vcr/root"
)

var (
	// use build flags to set these values - ie: go build -ldflags "-X main.version=1.0.0"
	apiVersion = "v0.3"
	version    = "dev"
	buildDate  = "2021-09-01T00:00:00Z"
	commit     = "0000"
	releaseURL = "https://api.github.com/repos/Vonage/vonage-cloud-runtime-cli"
)

func main() {
	f := cmdutil.NewDefaultFactory(apiVersion, releaseURL)
	ctx := context.Background()
	updateMessageChan := make(chan string)
	rootCmd := root.NewCmdRoot(f, version, buildDate, commit, updateMessageChan)

	cmd, err := rootCmd.ExecuteContextC(ctx)
	if err != nil {
		printError(f.IOStreams(), err, cmd, updateMessageChan)
		return
	}
}

func printError(out *iostreams.IOStreams, err error, cmd *cobra.Command, updateMessageChan chan string) {
	c := out.ColorScheme()
	var flagError *cmdutil.FlagError
	var httpErr api.Error
	if errors.As(err, &flagError) || strings.HasPrefix(err.Error(), "unknown command ") {
		fmt.Fprintf(out.ErrOut, "%s\n", err)
		fmt.Fprintln(out.ErrOut, cmd.UsageString())
	} else if errors.As(err, &httpErr) {
		fmt.Fprintf(out.ErrOut, "%s %s\n", c.FailureIcon(), format.PrintAPIError(err, &httpErr))
		format.PrintUpdateMessage(out, version, updateMessageChan)
	} else {
		fmt.Fprintf(out.ErrOut, "%s %s\n", c.FailureIcon(), err)
		format.PrintUpdateMessage(out, version, updateMessageChan)

	}
}
