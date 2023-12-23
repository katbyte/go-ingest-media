package ktio

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gookit/color"
)

func RunCommand(indent int, confirm bool, command string, args ...string) error {
	color.Printf("  <darkGray>%s %s</>", command, strings.Join(args, " "))

	fields := strings.Fields(command)
	cmd := exec.Command(fields[0], args...)

	if confirm {
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
