package ppt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ConvertSlidesToImages converts a PPTX file to a sequence of images.
// It returns a slice of absolute paths to the generated images, sorted by page number.
func ConvertSlidesToImages(pptxPath string, outDir string) ([]string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	sofficeCmd := "soffice"
	if _, err := exec.LookPath("soffice"); err != nil {
		// Fallback for Mac
		possiblePath := "/Applications/LibreOffice.app/Contents/MacOS/soffice"
		if _, err := os.Stat(possiblePath); err == nil {
			sofficeCmd = possiblePath
		} else {
			return nil, fmt.Errorf("LibreOffice not found. Please install it (brew install --cask libreoffice) or ensure 'soffice' is in your PATH")
		}
	}

	// 1. Convert to PDF first. This is more reliable than HTML or direct image export
	// which often only exports the first slide or has low quality.
	cmd := exec.Command(sofficeCmd, "--headless", "--convert-to", "pdf", "--outdir", outDir, pptxPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("libreoffice conversion to pdf failed: %s, output: %s", err, string(output))
	}

	// Identify the generated PDF file
	pdfName := strings.TrimSuffix(filepath.Base(pptxPath), filepath.Ext(pptxPath)) + ".pdf"
	pdfPath := filepath.Join(outDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("expected pdf file not found: %s", pdfPath)
	}

	// 2. Convert PDF to images using pdftoppm (part of poppler)
	// pdftoppm -jpeg -r 150 input.pdf output_prefix
	prefix := "slide"
	cmd = exec.Command("pdftoppm", "-jpeg", "-r", "150", pdfPath, filepath.Join(outDir, prefix))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdftoppm conversion failed: %s, output: %s. (Ensure poppler is installed: brew install poppler)", err, string(output))
	}

	// 3. Collect generated images
	var images []string
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, prefix) && (strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
			images = append(images, filepath.Join(outDir, name))
		}
	}

	// If no images found, fail
	if len(images) == 0 {
		return nil, fmt.Errorf("no images found after pdftoppm conversion")
	}

	// 4. Sort images naturally
	sort.Slice(images, func(i, j int) bool {
		return extractNumber(images[i]) < extractNumber(images[j])
	})

	return images, nil
}

func extractNumber(s string) int {
	// pdftoppm format: prefix-1.jpg, prefix-01.jpg, or prefix-10.jpg
	// We just need to find the number at the end of the filename body.
	base := filepath.Base(s)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// search backwards for digits
	end := len(name)
	for end > 0 && name[end-1] >= '0' && name[end-1] <= '9' {
		end--
	}

	if end < len(name) {
		val, _ := strconv.Atoi(name[end:])
		return val
	}
	return 0
}
