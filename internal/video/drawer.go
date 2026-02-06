package video

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// DrawSubtitle draws text onto the image at srcPath and saves it to dstPath.
func DrawSubtitle(srcPath, dstPath, text string, fontSize int) error {
	// 1. Load Image
	imgFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return err
	}

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	// 2. Load Font
	fontPath := GetBestFontPath()
	if fontPath == "" {
		return fmt.Errorf("no suitable font found for subtitles")
	}

	b, err := ioutil.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("failed to read font %s: %v", fontPath, err)
	}

	loadedFont, err := truetype.Parse(b)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}

	// 3. Setup Context
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(loadedFont)
	c.SetFontSize(float64(fontSize)) // Dynamic size
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(image.NewUniform(color.White))

	// 4. Measure & Wrap Text
	// Create a face to measure width
	opts := truetype.Options{Size: float64(fontSize), DPI: 72}
	face := truetype.NewFace(loadedFont, &opts)

	// Max width: 90% of image width
	maxWidth := int(float64(rgba.Bounds().Dx()) * 0.9)
	paddingX := (rgba.Bounds().Dx() - maxWidth) / 2

	var lines []string
	var currentLine string

	runes := []rune(text)
	for _, r := range runes {
		testLine := currentLine + string(r)
		width := measureStringWidth(face, testLine)
		if width > maxWidth && len(currentLine) > 0 {
			lines = append(lines, currentLine)
			currentLine = string(r)
		} else {
			currentLine = testLine
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	// 5. Draw Lines (Bottom Up)
	lineHeight := int(float64(fontSize) * 1.5)
	totalHeight := len(lines) * lineHeight
	startY := rgba.Bounds().Dy() - totalHeight - 50 // Bottom margin

	for i, line := range lines {
		// Center text
		lineWidth := measureStringWidth(face, line)
		x := (rgba.Bounds().Dx() - lineWidth) / 2
		// If calculation fails, fallback to padding
		if x < paddingX {
			x = paddingX
		}

		y := startY + (i+1)*lineHeight
		pt := freetype.Pt(x, y)

		// Shadow
		c.SetSrc(image.NewUniform(color.Black))
		offset := 2
		for dy := -offset; dy <= offset; dy++ {
			for dx := -offset; dx <= offset; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				c.DrawString(line, freetype.Pt(pt.X.Ceil()+dx, pt.Y.Ceil()+dy))
			}
		}

		// Text
		c.SetSrc(image.NewUniform(color.White))
		c.DrawString(line, pt)
	}

	// 5. Save
	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, rgba, nil)
}

func measureStringWidth(face font.Face, text string) int {
	width := 0
	for _, x := range text {
		awidth, _ := face.GlyphAdvance(x)
		width += awidth.Ceil()
	}
	return width
}
