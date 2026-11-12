package ui

import (
	"strings"
	"unicode"

	"github.com/go-text/typesetting/segmenter"
)

// WrapText uses Unicode line opportunities rather than spaces alone, so Chinese
// labels wrap too. Oversized words break at grapheme boundaries without clipping.
func WrapText(value string, width float64, measure func(string) float64) string {
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		var segments segmenter.Segmenter
		segments.Init([]rune(paragraph))
		it := segments.LineIterator()
		line := ""
		trim := func(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }
		flush := func() { lines = append(lines, trim(line)); line = "" }
		for it.Next() {
			part := string(it.Line().Text)
			if measure(trim(line+part)) <= width {
				line += part
				continue
			}
			if line != "" {
				flush()
			}
			part = strings.TrimLeftFunc(part, unicode.IsSpace)
			if measure(trim(part)) <= width {
				line = part
				continue
			}
			var word segmenter.Segmenter
			word.Init([]rune(part))
			graphemes := word.GraphemeIterator()
			for graphemes.Next() {
				cluster := string(graphemes.Grapheme().Text)
				if line != "" && measure(line+cluster) > width {
					flush()
				}
				line += cluster
			}
		}
		flush()
	}
	return strings.Join(lines, "\n")
}
