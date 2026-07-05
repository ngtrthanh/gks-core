// Package docx extracts plain-text paragraphs from a .docx (OOXML) file using
// only the Go standard library (archive/zip + encoding/xml). It is deliberately
// minimal: it concatenates the text runs (<w:t>) of each paragraph (<w:p>) and
// returns one string per non-empty paragraph, in document order.
package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Paragraphs returns the non-empty text paragraphs of the .docx at path.
func Paragraphs(path string) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("docx: open %s: %w", path, err)
	}
	defer zr.Close()

	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return nil, fmt.Errorf("docx: word/document.xml not found in %s", path)
	}

	rc, err := doc.Open()
	if err != nil {
		return nil, fmt.Errorf("docx: open document.xml: %w", err)
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var (
		paras  []string
		cur    strings.Builder
		inText bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("docx: decode: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				cur.Reset()
			case "t":
				inText = true
			case "tab":
				cur.WriteByte('\t')
			case "br", "cr":
				cur.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				cur.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				if s := strings.TrimSpace(cur.String()); s != "" {
					paras = append(paras, s)
				}
			}
		}
	}
	return paras, nil
}
