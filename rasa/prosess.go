package rasa

import (
	"log"
	"os/exec"
)

type Manager struct {
	Workdir string
}

func NewManager(workdir string) *Manager {
	return &Manager{Workdir: workdir}
}

func (m *Manager) Train(intents []Intent) {
	log.Println("Generating training data...")
	err := GenerateNLU(intents, m.Workdir)
	if err != nil {
		log.Println("Failed to generate NLU:", err)
		return
	}

	cmd := exec.Command("rasa", "train")
	cmd.Dir = m.Workdir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Println("Starting Rasa train...")
	if err := cmd.Run(); err != nil {
		log.Println("Training failed:", err)
	} else {
		log.Println("Training finished successfully")
	}
}
