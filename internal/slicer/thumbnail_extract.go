package slicer

import (
	"bufio"
	"os"
	"strings"
)

func extractThumbnailFromGCode(gcodePath string) string {
	f, err := os.Open(gcodePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var b64 strings.Builder
	inThumb := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "; THUMBNAIL_BLOCK_START") {
			continue
		}
		if strings.Contains(line, "; thumbnail begin") {
			inThumb = true
			continue
		}
		if strings.Contains(line, "; thumbnail end") {
			break
		}
		if inThumb && strings.HasPrefix(line, "; ") {
			b64.WriteString(strings.TrimPrefix(line, "; "))
		}
	}
	return b64.String()
}
