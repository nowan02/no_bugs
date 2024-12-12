package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func FindTextareaLinux(pid int) uintptr {
	maps, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))

	ErrCheck(err)

	scanner := bufio.NewScanner(maps)

	scanner.Split(bufio.ScanLines)

	textareastart := ""

	for scanner.Scan() {
		currentline := strings.Split(scanner.Text(), " ")

		if currentline[1] == "r-xp" {
			textareastart = strings.Split(currentline[0], "-")[0]
			break
		}
	}

	maps.Close()

	addr, err := strconv.ParseUint(textareastart, 16, 64)

	ErrCheck(err)

	return uintptr(addr)
}
