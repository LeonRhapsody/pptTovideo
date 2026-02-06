package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	workDir := "uploads/1770355555555555000"
	absWorkDir, _ := filepath.Abs(workDir)
	textPath := filepath.Join(absWorkDir, "slide_text.txt")

	// Create dummy Text
	textContent := "Hello World Check Font"
	err := os.WriteFile(textPath, []byte(textContent), 0644)
	if err != nil {
		fmt.Printf("Failed to write Text: %v\n", err)
		return
	}
	defer os.Remove(textPath)

	img := filepath.Join(absWorkDir, "images/slide-01.jpg")
	audio := filepath.Join(absWorkDir, "audio_render/audio_0.mp3")
	out := filepath.Join(absWorkDir, "debug_out_drawtext.mp4")

	// Filter string for drawtext
	// drawtext=fontfile='...':textfile='...':fontcolor=white:fontsize=24:x=(w-text_w)/2:y=h-50

	// If font is not found by name, we might need path.
	// But let's try font='Microsoft YaHei' first.
	// On Mac, font names work if fontconfig is there.

	// Issue: ffmpeg `libfreetype` is needed for drawtext. Most have it.

	// Escaping: 'textfile' path needs escaping similar to before.
	// vf := fmt.Sprintf("pad=ceil(iw/2)*2:ceil(ih/2)*2,drawtext=textfile='%s':font='Microsoft YaHei':fontcolor=white:fontsize=40:x=(w-text_w)/2:y=h-th-50", textPath)

	// Using generic font first to verify 'drawtext' works, then will switch to YaHei.
	vf := fmt.Sprintf("pad=ceil(iw/2)*2:ceil(ih/2)*2,drawtext=textfile='%s':font='Arial':fontcolor=white:fontsize=40:x=(w-text_w)/2:y=h-th-50", textPath)

	fmt.Printf("Filter string: %s\n", vf)

	cmd := exec.Command("ffmpeg",
		"-loop", "1",
		"-i", img,
		"-i", audio,
		"-c:v", "libx264",
		"-tune", "stillimage",
		"-c:a", "aac",
		"-b:a", "192k",
		"-pix_fmt", "yuv420p",
		"-vf", vf,
		"-shortest",
		"-y", out,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("FFmpeg failed: %v\nOutput:\n%s\n", err, string(output))
	} else {
		fmt.Println("FFmpeg Success!")
	}
}
