package main

import (
	"debug/elf"
	"fmt"
	"io"
	"log"
)

func parseElf(FilePath string) {
	elfFile, err := elf.Open(FilePath)
	if err != nil {
		log.Fatal(err)
	}

	dwarfData, err := elfFile.DWARF()
	if err != nil {
		log.Fatal(err)
	}

	entryReader := dwarfData.Reader()

	for {
		entry, err := entryReader.Next()

		if err == io.EOF {
			break
		}

		fmt.Printf("%s\n", entry.Tag)

		fmt.Println("Fields:")
		for _, field := range entry.Field {
			fmt.Printf("\t%s %s %s\n", field.Class, field.Attr, field.Val)
		}
	}

	elfFile.Close()
}
