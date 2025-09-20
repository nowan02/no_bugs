package ssr

import (
	"bufio"
	"os"
)

type Row struct {
	Current bool
	Text    string
}

func ReadSourceFile(filepath string) ([]Row, error) {
	text := make([]Row, 0)

	file, err := os.Open(filepath)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		newRow := Row{
			Current: false,
			Text:    scanner.Text(),
		}

		text = append(text, newRow)
	}

	text[0].Current = true

	return text, nil
}
