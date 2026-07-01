package slicer

import (
	"bufio"
	"os"
	"strings"
)

func detectSupportInGCode(gcodePath string) (bool, error) {
	f, err := os.Open(gcodePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, ";type:support") ||
			strings.Contains(line, "; support") ||
			strings.Contains(line, ";type:support interface") {
			return true, nil
		}
	}
	return false, scanner.Err()
}
