package rasa

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func GenerateNLU(intents []TrainIntents, path string) (bool, error) {
	nluFile, err := os.CreateTemp(path, "nlu-*.yml")
	if err != nil {
		return false, err
	}
	defer nluFile.Close()

	domainFile, err := os.CreateTemp(path, "domain-*.yml")
	if err != nil {
		return false, err
	}
	defer domainFile.Close()

	fmt.Fprintln(nluFile, "version: \"3.1\"")
	fmt.Fprintln(nluFile, "nlu:")
	fmt.Fprintln(domainFile, "version: \"3.1\"")
	fmt.Fprintln(domainFile, "intents:")

	re := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	entities := make(map[string]struct{})
	for _, i := range intents {
		fmt.Fprintf(nluFile, "- intent: %s\n  examples: |\n", i.Name)
		fmt.Fprintf(domainFile, "  - %s\n", i.Name)
		for _, ex := range i.Examples {
			fmt.Fprintf(nluFile, "    - %s\n", ex)
			for _, m := range re.FindAllStringSubmatch(ex, -1) {
				if len(m) > 1 {
					entities[m[1]] = struct{}{}
				}
			}
		}
	}

	fmt.Fprintln(domainFile, "\nentities:")
	for e := range entities {
		fmt.Fprintf(domainFile, "  - %s\n", e)
	}
	fmt.Fprintln(domainFile, "\nresponses: {}\n\nslots: {}\n\nactions: []")

	nluPath := fmt.Sprintf("%s/nlu.yml", path)
	domainPath := fmt.Sprintf("%s/domain.yml", path)

	dataTmp, err := os.ReadFile(nluFile.Name())
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(nluPath)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte{}
		} else {
			os.Remove(nluFile.Name())
			os.Remove(domainFile.Name())
			return false, err
		}
	}

	if bytes.Equal(dataTmp, data) {
		os.Remove(nluFile.Name())
		os.Remove(domainFile.Name())
		return false, nil
	}

	os.Rename(nluFile.Name(), nluPath)
	os.Rename(domainFile.Name(), domainPath)

	return true, nil
}

func GenerateDefaultConf(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	content := `language: ru

pipeline:
  - name: WhitespaceTokenizer

  - name: CountVectorsFeaturizer

  - name: DIETClassifier
    epochs: 100
    constrain_similarities: True  # полезно для cross-entropy loss, убирает warning
    intent_classification: True
    entity_recognition: True

  - name: EntitySynonymMapper

policies: []
`

	dir := filepath.Dir(path)
	if err := os.MkdirAll(fmt.Sprintf("%v", dir), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}
