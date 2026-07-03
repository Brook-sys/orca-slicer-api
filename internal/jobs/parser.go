package jobs

import (
	"regexp"
	"strconv"
)

// O OrcaSlicer CLI emite linhas como: "[10%] slicing model..." ou "10% ..."
var progressRegex = regexp.MustCompile(`\[?(\d+)%\]?`)

// ExtractProgress tenta achar um % na linha de log, retorna true se achou.
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
