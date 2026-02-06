package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type RenderOptions struct {
	EnableSubtitles bool
	FontSize        int
}

// ComposeVideo creates a video from corresponding images and audios with subtitles.
func ComposeVideo(images []string, audios []string, texts []string, output string, opts RenderOptions) error {
	if len(images) != len(audios) {
		return fmt.Errorf("number of images (%d) and audios (%d) do not match", len(images), len(audios))
	}

	tempDir := filepath.Dir(output)
	var videoParts []string
	var tempImages []string

	for i, img := range images {
		audio := audios[i]
		text := ""
		if i < len(texts) {
			text = texts[i]
		}

		currentImg := img
		if opts.EnableSubtitles && text != "" {
			burnedImgPath := filepath.Join(tempDir, fmt.Sprintf("burned_%d.jpg", i))
			err := DrawSubtitle(img, burnedImgPath, text, opts.FontSize)
			if err != nil {
				fmt.Printf("Warning: Failed to draw subtitle for slide %d: %v\n", i, err)
			} else {
				currentImg = burnedImgPath
				tempImages = append(tempImages, burnedImgPath)
			}
		}

		partPath := filepath.Join(tempDir, fmt.Sprintf("part_%d.mp4", i))

		// FIX: Get exact audio duration to ensure perfect sync
		dur, err := getDuration(audio)
		if err != nil {
			fmt.Printf("Warning: Failed to get duration for %s: %v\n", audio, err)
			dur = 5.0 // Fallback
		}

		// Standard simple filter: pad ensures even dimensions
		vf := "pad=ceil(iw/2)*2:ceil(ih/2)*2"

		// Use -t for explicit duration matching
		input1 := ffmpeg.Input(currentImg, ffmpeg.KwArgs{"loop": 1, "t": dur})
		input2 := ffmpeg.Input(audio)

		err = ffmpeg.Output([]*ffmpeg.Stream{input1, input2}, partPath, ffmpeg.KwArgs{
			"c:v":     "libx264",
			"tune":    "stillimage",
			"c:a":     "aac",
			"b:a":     "192k",
			"pix_fmt": "yuv420p",
			"vf":      vf,
		}).
			OverWriteOutput().
			Run()

		if err != nil {
			return fmt.Errorf("failed to create part %d: %v", i, err)
		}
		videoParts = append(videoParts, partPath)
	}

	// 2. Concatenate all parts
	concatListPath := filepath.Join(tempDir, "concat_list.txt")
	file, err := os.Create(concatListPath)
	if err != nil {
		return err
	}

	for _, part := range videoParts {
		file.WriteString(fmt.Sprintf("file '%s'\n", filepath.Base(part)))
	}
	file.Close()

	err = ffmpeg.Input(concatListPath, ffmpeg.KwArgs{"f": "concat", "safe": 0}).
		Output(output, ffmpeg.KwArgs{"c": "copy"}).
		OverWriteOutput().
		Run()

	if err != nil {
		return fmt.Errorf("failed to concat videos: %w", err)
	}

	// Cleanup
	for _, part := range videoParts {
		os.Remove(part)
	}
	for _, tmp := range tempImages {
		os.Remove(tmp)
	}
	os.Remove(concatListPath)

	return nil
}

func getDuration(path string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	val := strings.TrimSpace(string(out))
	return strconv.ParseFloat(val, 64)
}
