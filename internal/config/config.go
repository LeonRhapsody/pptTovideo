package config

import "os"

type Config struct {
	XunfeiAppID     string
	XunfeiAPIKey    string
	XunfeiAPISecret string
	Port            string
}

func LoadConfig() *Config {
	return &Config{
		XunfeiAppID:     os.Getenv("XUNFEI_APPID"),
		XunfeiAPIKey:    os.Getenv("XUNFEI_API_KEY"),
		XunfeiAPISecret: os.Getenv("XUNFEI_API_SECRET"),
		Port:            getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
