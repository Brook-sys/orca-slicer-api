package slicer

import (
	"archive/zip"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var timePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)total estimated time:\s*((?:(\d+)d\s*)?(?:(\d+)h\s*)?(?:(\d+)m\s*)?(?:(\d+)s)?)`),
	regexp.MustCompile(`(?i); estimated printing time \(normal mode\) =\s*((?:(\d+)d\s*)?(?:(\d+)h\s*)?(?:(\d+)m\s*)?(?:(\d+)s)?)`),
}

var filamentMMPattern = regexp.MustCompile(`; filament used \[mm\] =\s*(\d+(?:\.\d+)?)`)
var filamentGPattern = regexp.MustCompile(`; filament used \[g\] =\s*(\d+(?:\.\d+)?)`)

func ExtractMetadata(path string) Metadata {
	if strings.HasSuffix(strings.ToLower(path), ".3mf") {
		return extract3MFMetadata(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}
	}
	return ParseMetadata(string(data))
}

func ParseMetadata(content string) Metadata {
	metadata := Metadata{}
	for _, pattern := range timePatterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) == 6 {
			metadata.PrintTimeSeconds = seconds(matches[2], matches[3], matches[4], matches[5])
			break
		}
	}
	if matches := filamentMMPattern.FindStringSubmatch(content); len(matches) == 2 {
		metadata.FilamentUsedMM = parseFloat(matches[1])
	}
	if matches := filamentGPattern.FindStringSubmatch(content); len(matches) == 2 {
		metadata.FilamentUsedG = parseFloat(matches[1])
	}
	return metadata
}

func AggregateMetadata(files []string) Metadata {
	metadata := Metadata{}
	for _, file := range files {
		item := ExtractMetadata(file)
		metadata.PrintTimeSeconds += item.PrintTimeSeconds
		metadata.FilamentUsedG += item.FilamentUsedG
		metadata.FilamentUsedMM += item.FilamentUsedMM
	}
	return metadata
}

func extract3MFMetadata(path string) Metadata {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Metadata{}
	}
	defer reader.Close()

	metadata := Metadata{}
	for _, file := range reader.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".gcode") {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			continue
		}
		item := ParseMetadata(string(data))
		metadata.PrintTimeSeconds += item.PrintTimeSeconds
		metadata.FilamentUsedG += item.FilamentUsedG
		metadata.FilamentUsedMM += item.FilamentUsedMM
	}
	return metadata
}

func seconds(days string, hours string, minutes string, secs string) float64 {
	return float64(parseInt(days)*86400 + parseInt(hours)*3600 + parseInt(minutes)*60 + parseInt(secs))
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}
