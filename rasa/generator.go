package rasa

import (
	"fmt"
	"os"
)

func GenerateNLU(intents []Intent, path string) error {
	f, err := os.Create(fmt.Sprintf("%s/nlu.yml", path))
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "version: \"3.1\"")
	fmt.Fprintln(f, "nlu:")

	for _, i := range intents {
		fmt.Fprintf(f, "- intent: %s\n  examples: |\n", i.Name)
		for _, ex := range i.Examples {
			fmt.Fprintf(f, "    - %s\n", ex)
		}
	}
	return nil
}
