package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// First return: start of the textarea, second return: end
func FindTextareaLinux(pid int) (uintptr, uintptr) {
	maps, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))

	ErrCheck(err)

	scanner := bufio.NewScanner(maps)

	scanner.Split(bufio.ScanLines)

	textareastart, textareaend := "", ""

	for scanner.Scan() {
		currentline := strings.Split(scanner.Text(), " ")

		if currentline[1] == "r-xp" {
			textareastart = strings.Split(currentline[0], "-")[0]
			textareaend = strings.Split(currentline[0], "-")[1]
			break
		}
	}

	maps.Close()

	begin_addr, err := strconv.ParseUint(textareastart, 16, 64)

	ErrCheck(err)

	end_addr, err := strconv.ParseUint(textareaend, 16, 64)

	ErrCheck(err)

	return uintptr(begin_addr), uintptr(end_addr)
}
