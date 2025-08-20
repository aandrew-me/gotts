package tts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
)

type Request struct {
	Text string `json:"text"`
	Voice string `json:"voice"`
	Pitch int `json:"pitch"`
	Rate int `json:"rate"`
}

func findVoiceIdByName(voiceName string) (string, error){
	for _, voice := range Voices {
		if voice.Name == voiceName {
			return voice.Id, nil
		}
	}

	return "", errors.New("Couldn't find voice id")
}

func GenerateAudio(text string, voiceName string, filepath string) error {
	client := &http.Client{}
	voiceId, err := findVoiceIdByName(voiceName)

	if err != nil {
		return err
	}

	request := Request {
		Text: text,
		Voice: voiceId,
		Pitch: 0,
		Rate: 0,
	}

	requestStr, _ := json.Marshal(request)

	req, err := http.NewRequest("POST", "https://speechma.com/com.api/tts-api.php", bytes.NewBuffer(requestStr))
	if err != nil {
		return err
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://speechma.com")
	req.Header.Set("referer", "https://speechma.com/")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	mp3File, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer mp3File.Close()

	_, err = mp3File.Write(bodyText)
	if err != nil {
		return err
	}

	return nil
}
