package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:   "gotts",
		Short: "A cli tts tool",
		Long:  `GoTTS is a tool to generate speech from text in your terminal`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Check help menu with -h or --help")
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}

func isPipedInput() bool {
	stat, _ := os.Stdin.Stat()
	// Check if stdin is not a terminal
	return (stat.Mode() & os.ModeCharDevice) == 0
}
