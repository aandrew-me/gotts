package tts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/go-mp3"
	"github.com/youpy/go-wav"

	"github.com/gen2brain/malgo"
)

func PlayAudioMalgo(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()

	var reader io.Reader
	var channels, sampleRate uint32

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		w := wav.NewReader(file)
		f, err := w.Format()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		reader = w
		channels = uint32(f.NumChannels)
		sampleRate = f.SampleRate

	case ".mp3":
		m, err := mp3.NewDecoder(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		reader = m
		channels = 2
		sampleRate = uint32(m.SampleRate())
	default:
		fmt.Println("Not a valid file.")
		os.Exit(1)
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = channels
	deviceConfig.SampleRate = sampleRate
	deviceConfig.Alsa.NoMMap = 1

	// Channel to notify main goroutine that playback finished (EOF)
	done := make(chan struct{}, 1)

	onSamples := func(pOutputSample, _ []byte, framecount uint32) {
		// Try to fill the whole output buffer from the decoder.
		n, err := io.ReadFull(reader, pOutputSample)
		if err != nil {
			// If we got a partial read or EOF, zero out the rest (silence).
			if n < len(pOutputSample) {
				for i := n; i < len(pOutputSample); i++ {
					pOutputSample[i] = 0
				}
			}
			// Notify main that we're done — non-blocking so the callback never blocks.
			select {
			case done <- struct{}{}:
			default:
			}
			return
		}
		_ = err
	}

	deviceCallbacks := malgo.DeviceCallbacks{Data: onSamples}
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer device.Uninit()

	if err = device.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for done (EOF) OR for user Enter if stdin is interactive.
	// Only try Scanln if stdin is a TTY; otherwise just wait for done.
	if isInteractiveStdin() {
		fmt.Println("Playing audio, press Enter to quit...")
		// Use a goroutine to wait for Enter so we can still react to EOF
		enter := make(chan struct{})
		go func() {
			fmt.Scanln()
			close(enter)
		}()

		select {
		case <-enter:
			// User requested stop
		case <-done:
			// File finished playing
		}
	} else {
		// Non-interactive: block until audio finishes (EOF)
		<-done
	}

	_ = device.Stop()
}

func isInteractiveStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// If stdin is a pipe/redirect, ModeCharDevice won't be set.
	return (fi.Mode() & os.ModeCharDevice) != 0
}
