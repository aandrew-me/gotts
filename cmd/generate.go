package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/aandrew-me/gotts/tts"
	"github.com/spf13/cobra"
)

var (
	voiceName string
	text      string
	outfile   string
	isPlay    bool
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.PersistentFlags().StringVar(&voiceName, "voice", "Andrew Multilingual", "Name of the voice")
	generateCmd.PersistentFlags().StringVar(&text, "text", "Hello, I am go tts, a commandline text to speech tool", "Text to generate audio from")
	generateCmd.PersistentFlags().StringVar(&outfile, "out", "generated.mp3", "Generated audio file path")
	generateCmd.PersistentFlags().BoolVar(&isPlay, "play", false, "Play the generated audio file")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate audio from text",
	Run: func(cmd *cobra.Command, args []string) {
		voice, _ := cmd.Flags().GetString("voice")
		text, _ := cmd.Flags().GetString("text")
		filepath, _ := cmd.Flags().GetString("out")

		// tmpPath := path.Join(os.TempDir(), "gotts.mp3")

		if isPipedInput() {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Println("Error reading stdin")
				return
			}

			err = tts.GenerateAudio(string(data), "Andrew Multilingual", filepath)

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)

				return
			} else {
				fmt.Println("Saved audio to", filepath)
			}

			if isPlay {
				tts.PlayAudioMalgo(filepath)
			}

		} else {
			err := tts.GenerateAudio(text, voice, filepath)

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
			} else {
				fmt.Println("Saved audio file to", filepath)
			}

			if isPlay {
				tts.PlayAudioMalgo(filepath)
			}
		}

	},
}
