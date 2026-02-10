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
			fmt.Printf("Debug: rId %s not found in relMap\n", rId)
			continue
		}
		// Target is like "slides/slide1.xml"
		slideFilename := filepath.Base(target) // slide1.xml

		// Find notes for this slide
		slideRelPath := fmt.Sprintf("ppt/slides/_rels/%s.rels", slideFilename)

		noteText := ""
		noteFile := findNotesTarget(r, slideRelPath)
		if noteFile != "" {
			fullPath := filepath.Join("ppt/slides", noteFile)
			fullPath = filepath.ToSlash(filepath.Clean(fullPath))

			var errExtract error
			noteText, errExtract = extractTextFromXML(r, fullPath)
			if errExtract != nil {
				fmt.Printf("Warning: failed to extract notes from %s: %v\n", fullPath, errExtract)
			}
		}

		if noteText == "" {
			noteText = "No notes for this slide."
		}

		fmt.Printf("Debug: Slide %d (rId %s) notes length: %d\n", i+1, rId, len(noteText))

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

	// <p:sldId ... r:id="rId2"/>
	re := regexp.MustCompile(`sldId[^>]*r:id="([^"]+)"`)
	matches := re.FindAllStringSubmatch(content, -1)

	var ids []string
	for _, m := range matches {
		if len(m) > 1 {
			ids = append(ids, m[1])
		}
	}
	fmt.Printf("Debug: Found %d slide IDs in presentation.xml\n", len(ids))
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

	// Dynamic decoder to walk all nodes and find <a:t> occurrences
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var fullTextBuilder strings.Builder
	var inText bool
	var ignoreShape bool

	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			// If we enter a new shape, reset ignore flag
			if se.Name.Local == "sp" {
				ignoreShape = false
			}
			// Check for placeholder type
			if se.Name.Local == "ph" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "type" {
						val := strings.ToLower(attr.Value)
						// Ignore header, footer, date, and slide number placeholders
						if val == "hdr" || val == "ftr" || val == "dt" || val == "sldnum" {
							ignoreShape = true
						}
					}
				}
			}

			if se.Name.Local == "t" && !ignoreShape {
				inText = true
			}
		case xml.CharData:
			if inText {
				fullTextBuilder.WriteString(string(se))
			}
		case xml.EndElement:
			if se.Name.Local == "t" {
				inText = false
			}
			if se.Name.Local == "p" && !ignoreShape {
				fullTextBuilder.WriteString("\n")
			}
			if se.Name.Local == "sp" {
				ignoreShape = false
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
