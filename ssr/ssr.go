package ssr

import (
	"bufio"
	"os"
)

type Row struct {
	Current    bool
	Breakpoint bool
	Text       string
	Num        int
}

type Variables struct {
	Vartype string
	Values  []string
	Name    string
}

func ReadSourceFile(filepath string) ([]*Row, error) {

	n := 1

	text := make([]*Row, 0)

	file, err := os.Open(filepath)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		newRow := &Row{
			Current:    false,
			Text:       scanner.Text(),
			Num:        n,
			Breakpoint: false,
		}

		text = append(text, newRow)
		n++
	}

	return text, nil
}
