package tts

import (
	"os"

	"time"

	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"

)

func PlayAudio(file string) error {
    
    f, err := os.Open(file)
    if err != nil {
        return err
    }
    streamer, format, err := mp3.Decode(f)
    if err != nil {
        return err
    }
    // Do not defer streamer.Close() here, close after playback finishes.

    speaker.Init(beep.SampleRate(format.SampleRate), format.SampleRate.N(time.Second/10))

    done := make(chan bool)
    speaker.Play(beep.Seq(streamer, beep.Callback(func() {
        done <- true
    })))

    go func() {
        <-done
        streamer.Close()
    }()
    return nil
}

func PlayAudioSync(file string) error {
    f, err := os.Open(file)
    if err != nil {
        return err
    }

    streamer, format, err := mp3.Decode(f)
    if err != nil {
        return err
    }

    speaker.Init(beep.SampleRate(format.SampleRate), format.SampleRate.N(time.Second/10))

    done := make(chan bool)
    speaker.Play(beep.Seq(streamer, beep.Callback(func() {
        done <- true
    })))

    // Wait until playback is finished
    <-done
    streamer.Close()

    return nil
}

