package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type EdgeProvider struct {
	// Default voice if none provided
	DefaultVoice string
}

func NewEdgeProvider() *EdgeProvider {
	return &EdgeProvider{
		DefaultVoice: "zh-CN-XiaoxiaoNeural",
	}
}

func (e *EdgeProvider) Synthesize(text string, outputPath string, voiceName string) error {
	if voiceName == "" {
		voiceName = e.DefaultVoice
	}

	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second) // Simple backoff
		}

		// Use a context with timeout to prevent hangs
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Use python3.9 -m edge_tts since we installed it there
		cmd := exec.CommandContext(ctx, "python3.9", "-m", "edge_tts",
			"--text", text,
			"--write-media", outputPath,
			"--voice", voiceName,
		)

		output, err := cmd.CombinedOutput()
		cancel() // Cancel context after command finishes

		if err == nil {
			// Verify file exists and is not empty
			info, err := os.Stat(outputPath)
			if err == nil && info.Size() > 0 {
				return nil // Success
			}
			if err != nil {
				lastErr = fmt.Errorf("failed to stat output file: %w", err)
			} else {
				lastErr = fmt.Errorf("edge-tts generated empty file, output: %s", string(output))
			}
		} else {
			if ctx.Err() == context.DeadlineExceeded {
				lastErr = fmt.Errorf("edge-tts synthesis timed out after 30s")
			} else {
				lastErr = fmt.Errorf("edge-tts cli failed: %w, output: %s", err, string(output))
			}
		}
	}

	return fmt.Errorf("synthesis failed after %d attempts: %v", maxRetries, lastErr)
}
