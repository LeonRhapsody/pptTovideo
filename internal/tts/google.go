package tts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
)

type GoogleProvider struct {
	Config *config.Config
}

func NewGoogleProvider(cfg *config.Config) *GoogleProvider {
	return &GoogleProvider{Config: cfg}
}

func (p *GoogleProvider) Synthesize(text string, outputPath string, voiceName string, opts Options) error {
	apiKey := p.Config.GoogleAPIKey
	if apiKey == "" {
		return fmt.Errorf("Google Cloud API Key not configured")
	}

	url := "https://texttospeech.googleapis.com/v1/text:synthesize?key=" + apiKey

	if voiceName == "" {
		voiceName = "cmn-CN-Wavenet-A" // Default Chinese
	}

	// Lang code from voice name prefix, e.g. cmn-CN
	langCode := "cmn-CN"
	if len(voiceName) > 6 {
		langCode = voiceName[:6]
	}

	reqBody := map[string]interface{}{
		"input": map[string]interface{}{
			"text": text,
		},
		"voice": map[string]interface{}{
			"languageCode": langCode,
			"name":         voiceName,
		},
		"audioConfig": map[string]interface{}{
			"audioEncoding": "MP3",
			"speakingRate":  1.0, // Google supports 0.25 to 4.0
			"pitch":         0,   // -20.0 to 20.0 semitones
			"volumeGainDb":  0,
		},
	}

	// Add param mapping logic here if needed

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Google API failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Google returns JSON with "audioContent": base64 string
	var result struct {
		AudioContent string `json:"audioContent"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, decoded, 0644)
}
