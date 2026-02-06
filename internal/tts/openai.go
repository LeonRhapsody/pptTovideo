package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
)

type OpenAIProvider struct {
	Config *config.Config
}

func NewOpenAIProvider(cfg *config.Config) *OpenAIProvider {
	return &OpenAIProvider{Config: cfg}
}

func (p *OpenAIProvider) Synthesize(text string, outputPath string, voiceName string, opts Options) error {
	apiKey := p.Config.OpenAIAPIKey
	if apiKey == "" {
		return fmt.Errorf("OpenAI API Key not configured")
	}

	baseURL := p.Config.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/audio/speech", baseURL)

	if voiceName == "" {
		voiceName = "alloy"
	}

	// Rate handling: OpenAI supports speed 0.25 to 4.0. Default 1.0.
	speed := 1.0
	// ... simple parsing logic similar to System TTS but for float ...
	// TODO: Shared parsing logic would be nice

	reqBody := map[string]interface{}{
		"model": "tts-1",
		"input": text,
		"voice": voiceName,
		"speed": speed,
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API failed with status %d: %s", resp.StatusCode, string(body))
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}
