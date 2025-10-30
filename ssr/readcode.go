package ssr

import (
	"bufio"
	"html"
	"os"
)

type Row struct {
	Current    bool
	Breakpoint bool
	Text       string
	Num        int
}

func ReadSourceFile(filepath string) ([]Row, error) {

	n := 0

	text := make([]Row, 0)

	file, err := os.Open(filepath)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		newRow := Row{
			Current:    false,
			Text:       html.EscapeString(scanner.Text()),
			Num:        n,
			Breakpoint: false,
		}

		text = append(text, newRow)
		n++
	}

	text[0].Current = true

	return text, nil
}
