package auth

import (
	"math"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	logoLines    = 4
	padAfterLogo = 2
	menuItemRows = 2
	menuItemGap  = 1

	formHeaderLines  = 3 // title, subtitle, spacer
	formFieldStride  = 3 // label, input, gap
	formCheckboxLine = formHeaderLines + formFieldStride*3
)

func formContentWidth() int {
	w := 0
	for _, line := range strings.Split(logo, "\n") {
		if n := ansi.StringWidth(line); n > w {
			w = n
		}
	}
	return w
}

// formColumnWidth is the shared width for form inputs, labels, and hints.
func formColumnWidth() int {
	w := formContentWidth()
	for _, s := range []string{
		"Create an Intercept account on the server.",
		"Save credentials and connect.",
		"tab next · space toggle",
		"enter submit · esc back",
		"[x] Save credentials",
	} {
		if n := ansi.StringWidth(s); n > w {
			w = n
		}
	}
	return w
}

func menuItemLineRange(index int) (start, end int) {
	start = logoLines + padAfterLogo + index*(menuItemRows+menuItemGap)
	return start, start + menuItemRows - 1
}

func bodyContentSize(body string) (width, height int) {
	lines := strings.Split(body, "\n")
	height = len(lines)
	for _, line := range lines {
		if w := ansi.StringWidth(ansi.Strip(line)); w > width {
			width = w
		}
	}
	return width, height
}

func renderCentered(body string, termW, termH int) string {
	if termW <= 0 {
		return body
	}
	_, contentH := bodyContentSize(body)
	ox, oy := bodyOffset(body, termW, termH)

	lines := strings.Split(body, "\n")
	out := make([]string, 0, termH)
	for range oy {
		out = append(out, "")
	}
	for _, line := range lines {
		out = append(out, strings.Repeat(" ", ox)+line)
	}
	for range max(0, termH-oy-contentH) {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

func bodyOffset(body string, termW, termH int) (x, y int) {
	contentW, contentH := bodyContentSize(body)
	if gapX := termW - contentW; gapX > 0 {
		split := int(math.Round(float64(gapX) * 0.5))
		x = gapX - split
	}
	if gapY := termH - contentH; gapY > 0 {
		split := int(math.Round(float64(gapY) * 0.5))
		y = gapY - split
	}
	return x, y
}

func (m Model) menuChoiceAt(x, y int) (menuChoice, bool) {
	body := m.viewMenu()
	ox, oy := bodyOffset(body, m.width, m.height)
	relX := x - ox
	relY := y - oy
	if relX < 0 || relY < 0 {
		return 0, false
	}

	contentW, contentH := bodyContentSize(body)
	if relX >= contentW || relY >= contentH {
		return 0, false
	}

	for i := range 3 {
		start, end := menuItemLineRange(i)
		if relY >= start && relY <= end {
			return menuChoice(i), true
		}
	}
	return 0, false
}

func formFieldAt(body string, termW, termH, x, y int) (field int, ok bool) {
	ox, oy := bodyOffset(body, termW, termH)
	relX := x - ox
	relY := y - oy
	if relX < 0 || relY < 0 {
		return 0, false
	}

	contentW, contentH := bodyContentSize(body)
	if relX >= contentW || relY >= contentH {
		return 0, false
	}

	for i := range 3 {
		start := formHeaderLines + i*formFieldStride
		if relY >= start && relY <= start+1 {
			return i, true
		}
	}
	if relY == formCheckboxLine {
		return 3, true
	}
	return 0, false
}
