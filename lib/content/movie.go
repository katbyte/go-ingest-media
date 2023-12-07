package content

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Movie struct {
	Content

	// video details
	Resolution string
	Codec      string
	Audio      string
}

func (l Library) MovieFor(folder string) (*Movie, error) {
	m := Movie{}

	c, err := l.ContentFor(folder)
	if err != nil {
		return nil, err
	}
	m.Content = *c

	return &m, nil
}

type IndentWriter struct {
	w      io.Writer
	Indent string
}

// add "     " to the start of each line
func (iw IndentWriter) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		line := scanner.Text()
		indentedLine := fmt.Sprintf("%s%s", iw.Indent, line)

		if _, err = iw.w.Write([]byte(indentedLine)); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (m Movie) Move(intent int) error {
	cmd := exec.Command("echo", "mv", "-v", m.SrcPath(), m.DstPath())

	iw := IndentWriter{w: os.Stdout, Indent: strings.Repeat(" ", intent)}

	cmd.Stdout = iw
	cmd.Stderr = iw

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running command: %w", err)
	}

	return nil
}
