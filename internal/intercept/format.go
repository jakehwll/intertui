package intercept

import (
	"regexp"
	"strings"
)

var colorCode = regexp.MustCompile(`¬.`)

// ansiReset ends active SGR styling.
const ansiReset = "\033[0m"

// Intercept ¬-prefixed codes mapped to ANSI SGR (see intercept.py).
var ansiReplacer = strings.NewReplacer(
	"¬w", "\033[97m", // bright white
	"¬W", "\033[90m", // bright black
	"¬R", "\033[31m", // red
	"¬r", "\033[91m", // bright red
	"¬G", "\033[32m", // green
	"¬g", "\033[92m", // bright green
	"¬B", "\033[34m", // blue
	"¬b", "\033[94m", // bright blue
	"¬y", "\033[33m", // yellow
	"¬o", "\033[93m", // bright yellow
	"¬P", "\033[36m", // cyan
	"¬p", "\033[96m", // bright cyan
	"¬v", "\033[35m", // magenta
	"¬V", "\033[95m", // bright magenta
	"¬*", ansiReset,
	"¬?", "\033[37m", // white
)

// Clean strips Intercept game color codes (¬ prefix sequences).
func Clean(line string) string {
	return colorCode.ReplaceAllString(line, "")
}

// ANSI converts Intercept ¬ color codes to ANSI escape sequences.
func ANSI(line string) string {
	line = ansiReplacer.Replace(line)
	line = colorCode.ReplaceAllString(line, "")
	if strings.Contains(line, "\033[") && !strings.HasSuffix(line, ansiReset) {
		line += ansiReset
	}
	return line
}
