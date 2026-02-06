package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type SystemProvider struct {
	DefaultVoice string
}

func NewSystemProvider() *SystemProvider {
	return &SystemProvider{
		DefaultVoice: "Tingting",
	}
}

func (s *SystemProvider) Synthesize(text string, outputPath string, voiceName string, opts Options) error {
	if voiceName == "" {
		voiceName = s.DefaultVoice
	}

	// Rate handling
	// MacOS `say` uses words per minute (wpm). Default is around 175.
	// We'll map the percentage input (e.g., "+20%", "-10%") to a wpm value.
	baseRate := 175.0
	targetRate := baseRate

	if opts.Rate != "" {
		// Clean string: "+20%" -> "20", "-10%" -> "-10"
		cleanRate := strings.TrimSuffix(opts.Rate, "%")
		// Handle leading +
		// Atoi handles negative sign, but usually doesn't like explicit +
		cleanRate = strings.TrimPrefix(cleanRate, "+")

		val, err := strconv.Atoi(cleanRate)
		if err == nil {
			factor := 1.0 + (float64(val) / 100.0)
			targetRate = baseRate * factor
		}
	}

	// MacOS `say` command can output AIFF or CoreAudio format.
	// For browser compatibility and size, we prefer MP3.
	// `say` on modern macOS can write to mp3 directly using `--data-format=LEF32@8000`? No, effectively it uses CoreAudio.
	// Easiest reliable way: say -> aiff -> ffmpeg -> mp3

	tmpAiff := outputPath + ".aiff"

	args := []string{"-v", voiceName, "-r", fmt.Sprintf("%.0f", targetRate), "-o", tmpAiff, text}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "say", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("system tts timed out")
		}
		return fmt.Errorf("system tts failed: %w, output: %s", err, string(output))
	}

	// Convert to MP3
	// ffmpeg -i input.aiff -y -acodec libmp3lame -qscale:a 2 output.mp3
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", "-i", tmpAiff, "-y", "-acodec", "libmp3lame", "-qscale:a", "2", outputPath)
	if out, err := ffmpegCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w, output: %s", err, string(out))
	}

	// Cleanup temp aiff
	os.Remove(tmpAiff)

	// Verify file
	info, err := os.Stat(outputPath)
	if err == nil && info.Size() > 0 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}
	return fmt.Errorf("system tts generated empty file")
}
