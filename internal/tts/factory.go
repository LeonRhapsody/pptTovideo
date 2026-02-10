package tts

import (
	"errors"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
)

type EngineType string

const (
	EngineXunfei EngineType = "xunfei"
	EngineEdge   EngineType = "edge"
	EngineSystem EngineType = "system"
	EngineVolc   EngineType = "volcengine"
	EngineGoogle EngineType = "google"
	EngineOpenAI EngineType = "openai"
	EngineFish   EngineType = "fishspeech"
)

// NewTTSProvider returns a TTSProvider based on the engine type.
func NewTTSProvider(engine EngineType, cfg *config.Config) (TTSProvider, error) {
	switch engine {
	case EngineXunfei:
		return NewXunfeiProvider(cfg), nil
	case EngineEdge:
		return NewEdgeProvider(), nil
	case EngineSystem:
		return NewSystemProvider(), nil
	case EngineVolc:
		return NewVolcengineProvider(cfg), nil
	case EngineGoogle:
		return NewGoogleProvider(cfg), nil
	case EngineOpenAI:
		return NewOpenAIProvider(cfg), nil
	case EngineFish:
		return NewFishSpeechProvider(cfg), nil
	default:
		return nil, errors.New("unsupported TTS engine")
	}
}
