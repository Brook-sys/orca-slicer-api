package jobs

import (
	"regexp"
	"strconv"
)

var progressRegex = regexp.MustCompile(`\[?(\d+)%\]?`)

func ExtractProgress(line string) (int, bool) {
	matches := progressRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		val, err := strconv.Atoi(matches[1])
		if err == nil {
			return val, true
		}
	}
	return 0, false
}
