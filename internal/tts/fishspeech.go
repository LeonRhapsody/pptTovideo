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

type FishSpeechProvider struct {
	Config *config.Config
}

func NewFishSpeechProvider(cfg *config.Config) *FishSpeechProvider {
	return &FishSpeechProvider{Config: cfg}
}

func (p *FishSpeechProvider) Synthesize(text string, outputPath string, voiceName string, opts Options) error {
	apiKey := p.Config.FishSpeechAPIKey
	// API Key is required for cloud but might not be for local.
	// We'll let the request fail if it's missing on cloud.

	baseURL := p.Config.FishSpeechAPIURL
	if baseURL == "" {
		baseURL = "https://api.fish.audio/v1/tts"
	}

	if voiceName == "" {
		// Default reference_id if none provided.
		// Note: User should usually provide a valid reference_id/voice model ID.
		voiceName = "7f02fd683e0d4050a133900c4644377b" // A common default if available
	}

	reqBody := map[string]interface{}{
		"text":         text,
		"reference_id": voiceName,
		"format":       "mp3",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Fish Speech API failed with status %d: %s", resp.StatusCode, string(body))
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}
