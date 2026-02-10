package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	// Xunfei
	XunfeiAppID     string `json:"xunfei_app_id"`
	XunfeiAPIKey    string `json:"xunfei_api_key"`
	XunfeiAPISecret string `json:"xunfei_api_secret"`

	// Volcengine
	VolcAccessKey string `json:"volc_access_key"`
	VolcSecretKey string `json:"volc_secret_key"`
	VolcAppKey    string `json:"volc_app_key"`

	// Google
	GoogleAPIKey string `json:"google_api_key"`

	// OpenAI
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"` // Optional proxy

	// Fish Speech
	FishSpeechAPIKey string `json:"fish_speech_api_key"`
	FishSpeechAPIURL string `json:"fish_speech_api_url"` // e.g., https://api.fish.audio/v1/tts

	Port string `json:"port"`

	mu sync.RWMutex
}

const ConfigFile = "config.json"

func LoadConfig() *Config {
	cfg := &Config{
		Port: "8080",
	}

	// Try loading from file first
	if file, err := os.ReadFile(ConfigFile); err == nil {
		json.Unmarshal(file, cfg)
	}

	// Allow env vars to override (optional, keeping backward compat)
	if v := os.Getenv("XUNFEI_APPID"); v != "" {
		cfg.XunfeiAppID = v
	}
	if v := os.Getenv("XUNFEI_API_KEY"); v != "" {
		cfg.XunfeiAPIKey = v
	}
	if v := os.Getenv("XUNFEI_API_SECRET"); v != "" {
		cfg.XunfeiAPISecret = v
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}

	return cfg
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
