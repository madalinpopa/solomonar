package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	keyCtrlC  = 3
	keyEscape = 27
)

var errInputClosed = errors.New("read answer: input closed")

// prompter asks the user questions on one shared input stream (I section 3.7).
type prompter struct {
	in      *bufio.Reader
	out     io.Writer
	rawMode func() (restore func(), err error)
}

type choice struct {
	name        string
	description string
}

func newPrompter(env Env) *prompter {
	return &prompter{in: bufio.NewReader(env.Stdin), out: env.Stdout, rawMode: env.RawMode}
}

// text asks for one line; an empty answer gives def.
func (p *prompter) text(label, def string) (string, error) {
	if def == "" {
		fmt.Fprintf(p.out, "%s: ", label)
	} else {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	}
	answer, err := p.readLine()
	if err != nil {
		return "", err
	}
	if answer == "" {
		return def, nil
	}
	return answer, nil
}

// confirm asks a yes-or-no question until the answer is one; an empty answer gives def.
func (p *prompter) confirm(question string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	for {
		fmt.Fprintf(p.out, "%s %s ", question, hint)
		answer, err := p.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintln(p.out, "  answer y or n")
	}
}

func (p *prompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if errors.Is(err, io.EOF) {
		return "", errInputClosed
	}
	if err != nil {
		return "", fmt.Errorf("read answer: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// choose lets the user mark items with keys and returns the marked names in item order (CLI-D5).
func (p *prompter) choose(items []choice, selected []string) ([]string, error) {
	restore, err := p.rawMode()
	if err != nil {
		return nil, err
	}
	defer restore()
	marked := make([]bool, len(items))
	for i, item := range items {
		marked[i] = slices.Contains(selected, item.name)
	}
	list := checklist{items: items, marked: marked}
	fmt.Fprint(p.out, list.render(true))
	for {
		key, err := p.readKey()
		if err != nil {
			return nil, err
		}
		if key == '\r' || key == '\n' {
			fmt.Fprint(p.out, list.clear()+list.render(false))
			return list.names(), nil
		}
		if list.press(key) {
			fmt.Fprint(p.out, list.clear()+list.render(true))
		}
	}
}

// readKey reads one key, folding the arrow escape sequences into 'k' and 'j'; 0 is a key to ignore.
func (p *prompter) readKey() (byte, error) {
	key, err := p.readByte()
	if err != nil || key != keyEscape {
		return key, err
	}
	if next, err := p.readByte(); err != nil || next != '[' {
		return 0, err
	}
	arrow, err := p.readByte()
	switch {
	case err != nil:
		return 0, err
	case arrow == 'A':
		return 'k', nil
	case arrow == 'B':
		return 'j', nil
	default:
		return 0, nil
	}
}

func (p *prompter) readByte() (byte, error) {
	b, err := p.in.ReadByte()
	switch {
	case errors.Is(err, io.EOF):
		return 0, errInputClosed
	case err != nil:
		return 0, fmt.Errorf("read answer: %w", err)
	case b == keyCtrlC:
		return 0, context.Canceled
	}
	return b, nil
}

type checklist struct {
	items  []choice
	marked []bool
	cursor int
}

// press applies one key and reports whether the list changed.
func (c *checklist) press(key byte) bool {
	switch key {
	case 'k':
		c.cursor = max(c.cursor-1, 0)
	case 'j':
		c.cursor = min(c.cursor+1, len(c.items)-1)
	case ' ':
		c.marked[c.cursor] = !c.marked[c.cursor]
	case 'a', 'n':
		for i := range c.marked {
			c.marked[i] = key == 'a'
		}
	default:
		return false
	}
	return true
}

// render lists the items, with the cursor and help line while the user is choosing.
// Lines end in "\r\n" because raw mode turns off the terminal's output processing.
func (c *checklist) render(choosing bool) string {
	width := 0
	for _, item := range c.items {
		width = max(width, len(item.name))
	}
	var b strings.Builder
	for i, item := range c.items {
		pointer, box := "  ", "[ ]"
		if choosing && i == c.cursor {
			pointer = "> "
		}
		if c.marked[i] {
			box = "[x]"
		}
		line := fmt.Sprintf("  %s%s %-*s  %s", pointer, box, width, item.name, item.description)
		b.WriteString(strings.TrimRight(line, " ") + "\r\n")
	}
	if choosing {
		b.WriteString("  ↑/↓ move · Space toggle · Enter accept\r\n")
	}
	return b.String()
}

// clear moves the cursor up over the rendered list and erases it.
func (c *checklist) clear() string {
	return fmt.Sprintf("\x1b[%dA\r\x1b[J", len(c.items)+1)
}

func (c *checklist) names() []string {
	names := []string{}
	for i, item := range c.items {
		if c.marked[i] {
			names = append(names, item.name)
		}
	}
	return names
}
