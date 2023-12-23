package ktio

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type IndentWriter struct {
	W      io.Writer
	Indent string
}

// add "     " to the start of each line
func (iw IndentWriter) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		line := scanner.Text()
		indentedLine := fmt.Sprintf("%s%s\n", iw.Indent, line)

		if _, err = iw.W.Write([]byte(indentedLine)); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
