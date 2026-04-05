package prompt

import (
	"github.com/charmbracelet/huh"
)

// Confirm prompts the user with a yes/no question using huh.
// Returns true if confirmed, false otherwise. Defaults to false.
func Confirm(title string) (bool, error) {
	var confirmed bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}
