package ktio

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gookit/color"
)

func RunCommand(indent int, prompt bool, command string, args ...string) error {
	color.Printf("  <darkGray>%s %s</>", command, strings.Join(args, " "))

	cmd := exec.Command(command, args...) //nolint:gosec

	if prompt {
		color.Printf(" <lightYellow> CONFIRM y/n: </>")
		y, err := Confirm()
		fmt.Println()
		if err != nil {
			return err
		}
		if !y {
			return nil
		}
	} else {
		fmt.Println()
	}

	iw := IndentWriter{W: os.Stdout, Indent: strings.Repeat(" ", indent)}
	cmd.Stdout = iw
	cmd.Stderr = iw

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running command: %w", err)
	}

	return nil
}
