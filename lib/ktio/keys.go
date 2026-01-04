package ktio

import (
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/eiannone/keyboard"
)

func Confirm() (bool, error) {
	for {
		char, err := GetKey()
		if err != nil {
			return false, err
		}

		switch *char {
		case 'y', 'Y':
			return true, nil
		case 'n', 'N':
			return false, nil
		}

		fmt.Printf("%s", string(*char))
	}
}

func GetSelection(options ...rune) (rune, error) {
	optionMap := make(map[rune]bool)
	for _, option := range options {
		optionMap[option] = true
	}

	for {
		selected, err := GetKey()
		if err != nil {
			return 0, err
		}

		if optionMap[*selected] {
			fmt.Printf("%s", string(*selected))
			return *selected, nil
		}
		// Ignore invalid ktio and continue
	}
}

func GetKey() (*rune, error) {
	err := keyboard.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = keyboard.Close()
	}()

	// Setting up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	select {
	case <-sigChan:
		// Handling Ctrl-C
		return nil, errors.New("interrupt received")
	case key := <-getKeyPress():
		return key, nil
	}
}

// getKeyPress captures a key press and returns it through a channel
func getKeyPress() chan *rune {
	ch := make(chan *rune)
	go func() {
		char, _, err := keyboard.GetSingleKey()
		if err != nil {
			ch <- nil
		} else {
			ch <- &char
		}
		close(ch)
	}()
	return ch
}
