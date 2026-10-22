package project

import (
	"bytes"
	"cmp"
	"slices"
	"strings"
)

// Render keeps the "## With" sections whose extensions are installed and adds provenance
// (I section 4.3).
func Render(content []byte, self string, installed []string, decision string) (rendered []byte, with []string) {
	lines := slices.Collect(bytes.Lines(content))
	var out [][]byte
	for i := 0; i < len(lines); {
		names, ok := sectionNames(lines, i)
		if !ok {
			out = append(out, lines[i])
			i++
			continue
		}
		end := i + 1
		for end < len(lines) && !bytes.HasPrefix(lines[end], []byte("## ")) {
			end++
		}
		switch {
		case !containsAll(installed, names) && end == len(lines):
			out = trimBlankTail(out)
		case containsAll(installed, names):
			out = append(append(out, lines[i]), lines[i+2:end]...)
			with = append(with, names...)
		}
		i = end
	}
	with = slices.DeleteFunc(slices.Compact(slices.Sorted(slices.Values(with))), func(n string) bool { return n == self })
	if len(with) == 0 {
		with = nil
	}
	return bytes.Join(withProvenance(out, provenance(decision, with)), nil), with
}

// sectionNames parses the names of a "## With" heading at line i followed by its marker line.
func sectionNames(lines [][]byte, i int) ([]string, bool) {
	if !bytes.HasPrefix(lines[i], []byte("## With ")) || i+1 == len(lines) {
		return nil, false
	}
	marker := strings.TrimSpace(string(lines[i+1]))
	list, ok := strings.CutPrefix(marker, "<!-- solomonar:with ")
	list, closed := strings.CutSuffix(list, " -->")
	if !ok || !closed {
		return nil, false
	}
	var names []string
	for name := range strings.SplitSeq(list, ",") {
		names = append(names, strings.TrimSpace(name))
	}
	return names, true
}

func containsAll(installed, names []string) bool {
	return !slices.ContainsFunc(names, func(n string) bool { return !slices.Contains(installed, n) })
}

func trimBlankTail(lines [][]byte) [][]byte {
	for len(lines) > 0 && len(bytes.TrimSpace(lines[len(lines)-1])) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func provenance(decision string, with []string) string {
	line := "Installed by " + cmp.Or(decision, "solomonar")
	if len(with) > 0 {
		line += "; rendered with: " + strings.Join(with, ", ")
	}
	return line + ".\n"
}

// withProvenance puts line under the first "# " title, between single blank lines.
func withProvenance(lines [][]byte, line string) [][]byte {
	title := slices.IndexFunc(lines, func(l []byte) bool { return bytes.HasPrefix(l, []byte("# ")) })
	if title < 0 {
		return lines
	}
	rest := title + 1
	for rest < len(lines) && len(bytes.TrimSpace(lines[rest])) == 0 {
		rest++
	}
	heading := bytes.TrimSuffix(lines[title], []byte("\n"))
	block := [][]byte{lines[title][:len(heading)], []byte("\n\n"), []byte(line)}
	if rest < len(lines) {
		block = append(block, []byte("\n"))
	}
	return slices.Concat(lines[:title], block, lines[rest:])
}
