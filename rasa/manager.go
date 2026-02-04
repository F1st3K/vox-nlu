package rasa

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

type Manager struct {
	nlu     *NLUProcess
	rasaBin string
	model   string
	mu      sync.Mutex
	Workdir string
}

func (m *Manager) Start() error {
	err := GenerateDefaultConf(fmt.Sprintf("%s/config.yml", m.Workdir))

	if err != nil {
		log.Println("Failed to generate default conf:", err)
		return nil
	}

	m.nlu = NewNLUProcess(m.rasaBin, m.model)
	return m.nlu.Start()
}

func (m *Manager) Stop() {
	if m.nlu != nil {
		m.nlu.Stop()
	}
}

func (m *Manager) Retrain(intents []Intent) error {
	log.Println("Generating training data...")

	genDir := fmt.Sprintf("%s/generated", m.Workdir)
	err := os.MkdirAll(genDir, 0755)

	isRegerated, err := GenerateNLU(intents, genDir)
	if err != nil {
		log.Println("Failed to generate NLU:", err)
		return nil
	}

	fi, err := os.Stat(fmt.Sprintf("%s/generated/model-nlu-only.tar.gz", m.Workdir))
	if !isRegerated && err == nil && !fi.IsDir() {
		log.Println("Skip retrain NLU:", "config is regenerated:", isRegerated, "error:", err)
		return nil
	}

	cmd := exec.Command("rasa", "train", "nlu",
		"-c", fmt.Sprintf("%s/config.yml", m.Workdir),
		"-d", fmt.Sprintf("%s/generated/domain.yml", m.Workdir),
		"-u", fmt.Sprintf("%s/generated/nlu.yml", m.Workdir),
		"--out", fmt.Sprintf("%s/generated", m.Workdir),
		"--fixed-model-name", "model-nlu-only",
	)
	cmd.Dir = m.Workdir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Println("Starting Rasa train...")
	if err := cmd.Run(); err != nil {
		log.Println("Training failed:", err)
	} else {
		log.Println("Training finished successfully")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Stop()

	return m.Start()
}

func (m *Manager) Parse(text string) (*NLUResult, error) {
	return m.nlu.Parse(text)
}
