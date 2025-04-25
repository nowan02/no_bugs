package target

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// First return: start of the textarea, second return: end
func (t *Tracee) FindTextareaLinux() (uint64, uint64) {
	maps, err := os.Open(fmt.Sprintf("/proc/%d/maps", t.Proc.Pid))

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(maps)

	scanner.Split(bufio.ScanLines)

	textareastart, textareaend := "", ""
	prev_end := ""

	for scanner.Scan() {
		currentline := strings.Split(scanner.Text(), " ")

		if strings.Contains(currentline[len(currentline)-1], t.ElfPath) {
			if textareastart == "" {
				textareastart = strings.Split(currentline[0], "-")[0]
			}

			prev_end = strings.Split(currentline[0], "-")[1]
		} else {
			textareaend = prev_end
			break
		}
	}

	maps.Close()

	begin_addr, err := strconv.ParseUint(textareastart, 16, 64)

	if err != nil {
		panic(err)
	}

	end_addr, err := strconv.ParseUint(textareaend, 16, 64)

	if err != nil {
		panic(err)
	}

	return begin_addr, end_addr
}
