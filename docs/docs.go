package main

import (
	"log"

	"github.com/spf13/cobra/doc"
	"vcr-cli/pkg/cmdutil"
	"vcr-cli/vcr/root"
)

func main() {
	f := cmdutil.NewDefaultFactory("v0.3", "https://api.github.com/repos/Vonage/vonage-cloud-runtime-cli")
	updateMessageChan := make(chan string)
	defer close(updateMessageChan)
	rootCmd := root.NewCmdRoot(f, "dev", "2021-09-01T00:00:00Z", "0000", updateMessageChan)
	err := doc.GenMarkdownTree(rootCmd, "./docs")
	if err != nil {
		log.Fatal(err)
	}
}
