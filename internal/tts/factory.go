package tts

import (
	"errors"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
)

type EngineType string

const (
	EngineXunfei EngineType = "xunfei"
	EngineEdge   EngineType = "edge"
)

// NewTTSProvider returns a TTSProvider based on the engine type.
func NewTTSProvider(engine EngineType, cfg *config.Config) (TTSProvider, error) {
	switch engine {
	case EngineXunfei:
		return NewXunfeiProvider(cfg), nil
	case EngineEdge:
		return NewEdgeProvider(), nil
	default:
		return nil, errors.New("unsupported TTS engine")
	}
}
