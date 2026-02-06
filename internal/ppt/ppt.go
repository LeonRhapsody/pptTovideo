package ppt

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

type Slide struct {
	Index     int
	Note      string
	ImagePath string
}

// ParsePPT extracts notes from the PPTX file.
// It returns a list of Slides with populated Notes.
// Note: ImagePath is not populated here, it's done after image conversion.
func ParsePPT(pptxPath string) ([]Slide, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx: %w", err)
	}
	defer r.Close()

	// Map slide filename (e.g., "slide1.xml") to its content/relationships
	// We need to determine the order.
	// Robust way: Parse ppt/presentation.xml to get slide ID order (rId list).
	// Then map rId to filename via ppt/_rels/presentation.xml.rels.

	// 1. Parse relationship map (rId -> target)
	relMap, err := parseRelationships(r, "ppt/_rels/presentation.xml.rels")
	if err != nil {
		return nil, fmt.Errorf("failed to parse presentation rels: %w", err)
	}

	// 2. Parse presentation.xml to get slide order (list of rIds)
	slideRIds, err := parsePresentationOrder(r, "ppt/presentation.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to parse presentation slide order: %w", err)
	}

	var slides []Slide

	for i, rId := range slideRIds {
		target, ok := relMap[rId]
		if !ok {
			continue
		}
		// Target is like "slides/slide1.xml"
		slideFilename := filepath.Base(target) // slide1.xml

		// Find notes for this slide
		// Look in ppt/slides/_rels/slideX.xml.rels
		slideRelPath := fmt.Sprintf("ppt/slides/_rels/%s.rels", slideFilename)

		noteText := ""
		// slideRelMap was unused in previous code, removing checking directly.

		if err == nil {
			// Find relationship of type notesSlide
			// But since we just have a map of ID->Target, we need to scan the actual XML or just look for the target that contains "notesSlide" in our simple map?
			// The simple map I implemented below keys by rId. I should probably return more info or scan values.
			// Let's modify parseRelationships to return a list or helper.
			// Actually, let's just re-parse specifically for notes logic here to be precise.

			noteFile := findNotesTarget(r, slideRelPath)
			if noteFile != "" {
				// noteFile is usually relative to the slide part (ppt/slides/), e.g., "../notesSlides/notesSlide1.xml"
				// We assume the slide is in "ppt/slides/" based on standard structure.
				// Resolve path: ppt/slides/ + ../notesSlides/notesSlide1.xml -> ppt/notesSlides/notesSlide1.xml

				// Use path/filepath to clean it, but ensure we use forward slashes for zip
				fullPath := filepath.Join("ppt/slides", noteFile)
				// filepath.Clean handles the ".." resolution
				fullPath = filepath.ToSlash(filepath.Clean(fullPath))

				var errExtract error
				noteText, errExtract = extractTextFromXML(r, fullPath)
				if errExtract != nil {
					// Just log/print and continue, don't fail the whole parse
					fmt.Printf("Warning: failed to extract notes from %s: %v\n", fullPath, errExtract)
				}
			}
		}

		if noteText == "" {
			noteText = "No notes for this slide."
		}

		slides = append(slides, Slide{
			Index: i + 1,
			Note:  noteText,
		})
	}

	return slides, nil
}

// Helpers

func parseRelationships(r *zip.ReadCloser, path string) (map[string]string, error) {
	f := findFile(r, path)
	if f == nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	type Relationship struct {
		Id     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
		Type   string `xml:"Type,attr"`
	}
	type Relationships struct {
		List []Relationship `xml:"Relationship"`
	}

	var rels Relationships
	byteValue, _ := ioutil.ReadAll(rc)
	if err := xml.Unmarshal(byteValue, &rels); err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, rel := range rels.List {
		m[rel.Id] = rel.Target
	}
	return m, nil
}

func parsePresentationOrder(r *zip.ReadCloser, path string) ([]string, error) {
	f := findFile(r, path)
	if f == nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	// Simplified check for <p:sldIdLst> containing <p:sldId r:id="..."/>
	// We'll use regex for speed and simplicity avoiding complex structs with namespaces
	data, _ := ioutil.ReadAll(rc)
	content := string(data)

	// Regex to find r:id="..." inside sldId
	// <p:sldId ... r:id="rId2"/>
	re := regexp.MustCompile(`p:sldId[^>]*r:id="([^"]+)"`)
	matches := re.FindAllStringSubmatch(content, -1)

	var ids []string
	for _, m := range matches {
		if len(m) > 1 {
			ids = append(ids, m[1])
		}
	}
	return ids, nil
}

func findNotesTarget(r *zip.ReadCloser, path string) string {
	f := findFile(r, path)
	if f == nil {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()

	type Relationship struct {
		Target string `xml:"Target,attr"`
		Type   string `xml:"Type,attr"`
	}
	type Relationships struct {
		List []Relationship `xml:"Relationship"`
	}
	var rels Relationships
	data, _ := ioutil.ReadAll(rc)
	xml.Unmarshal(data, &rels)

	for _, rel := range rels.List {
		if strings.Contains(rel.Type, "notesSlide") {
			return rel.Target
		}
	}
	return ""
}

func extractTextFromXML(r *zip.ReadCloser, path string) (string, error) {
	f := findFile(r, path)
	if f == nil {
		return "", fmt.Errorf("file not found: %s", path)
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// XML Struct definitions for Notes Slide
	type Ph struct {
		Type string `xml:"type,attr"`
	}
	type NvPr struct {
		Ph Ph `xml:"ph"`
	}
	type NvSpPr struct {
		NvPr NvPr `xml:"nvPr"`
	}
	type T struct {
		Content string `xml:",chardata"`
	}
	type R struct {
		T T `xml:"t"`
	}
	type P struct {
		R []R `xml:"r"`
	}
	type TxBody struct {
		P []P `xml:"p"`
	}
	type Sp struct {
		NvSpPr NvSpPr `xml:"nvSpPr"`
		TxBody TxBody `xml:"txBody"`
	}
	type SpTree struct {
		Sp []Sp `xml:"sp"`
	}
	type CSld struct {
		SpTree SpTree `xml:"spTree"`
	}
	type Notes struct {
		CSld CSld `xml:"cSld"`
	}

	var notes Notes
	if err := xml.Unmarshal(data, &notes); err != nil {
		return "", err
	}

	var fullTextBuilder strings.Builder

	// Iterate over shapes to find the one with type="body"
	for _, sp := range notes.CSld.SpTree.Sp {
		if sp.NvSpPr.NvPr.Ph.Type == "body" {
			// This is the notes body
			for _, p := range sp.TxBody.P {
				for _, run := range p.R {
					fullTextBuilder.WriteString(run.T.Content)
				}
				// Add newline for each paragraph, or space?
				// Notes usually paragraphs. A newline is safer.
				fullTextBuilder.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(fullTextBuilder.String()), nil
}

func findFile(r *zip.ReadCloser, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}
