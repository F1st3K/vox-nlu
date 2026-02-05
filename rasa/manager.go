package rasa

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type RIntent struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

type REntity struct {
	Entity string      `json:"entity"`
	Value  interface{} `json:"value"`
	Start  int         `json:"start"`
	End    int         `json:"end"`
}

type NLUResult struct {
	Text     string    `json:"text"`
	Intent   RIntent   `json:"intent"`
	Entities []REntity `json:"entities"`
}

type TrainIntents struct {
	Name     string
	Examples []string
}

type ProcessText struct {
	Text string
}

type Manager struct {
	rasaBin string
	model   string
	config  string
	Workdir string

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.Writer
	scanner *bufio.Scanner
}

// NewManager создаёт новый Manager
func NewManager(rasaBin, workdir string) *Manager {
	return &Manager{
		rasaBin: rasaBin,
		Workdir: workdir,
		model:   fmt.Sprintf("%s/generated/model-nlu-only.tar.gz", workdir),
		config:  fmt.Sprintf("%s/config.yml", workdir),
	}
}

// Start запускает Rasa NLU subprocess
func (m *Manager) Start() error {
	err := GenerateDefaultConf(m.config)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := exec.Command(
		m.rasaBin,
		"shell", "nlu",
		"--model", m.model,
		"--quiet",
		"--log-level ERROR",
	)
	cmd.Env = append(os.Environ(),
		"PYTHONWARNINGS=ignore::DeprecationWarning,ignore::FutureWarning",
		"SQLALCHEMY_SILENCE_UBER_WARNING=1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	m.cmd = cmd
	m.stdin = stdin
	m.scanner = bufio.NewScanner(stdout)

	log.Println("Rasa NLU process started")
	return nil
}

// Stop корректно завершает процесс Rasa
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		_ = m.cmd.Process.Signal(os.Interrupt)
		m.cmd.Wait()
		m.cmd = nil
		m.stdin = nil
		m.scanner = nil
		log.Println("Rasa NLU process stopped")
	}
}

// Parse отправляет текст в Rasa и получает результат NLU
func (m *Manager) Parse(p ProcessText) (*NLUResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Fprintln(m.stdin, p.Text)

	var buf bytes.Buffer
	depth := 0
	started := false

	for m.scanner.Scan() {
		line := m.scanner.Text()

		// ждём начало JSON
		if strings.Contains(line, "{") {
			started = true
		}

		if !started {
			continue
		}

		buf.WriteString(line)
		buf.WriteByte('\n')

		depth += strings.Count(line, "{")
		depth -= strings.Count(line, "}")

		if started && depth == 0 {
			var res NLUResult
			if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
				return nil, err
			}
			fmt.Println(buf.String())
			return &res, nil
		}
	}

	return nil, io.EOF
}

// Train генерирует NLU данные и запускает тренировку модели
func (m *Manager) Train(intents []TrainIntents) error {
	log.Println("Generating training data...")

	genDir := fmt.Sprintf("%s/generated", m.Workdir)
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return err
	}

	isRegenerated, err := GenerateNLU(intents, genDir)
	if err != nil {
		log.Println("Failed to generate NLU:", err)
		return err
	}

	fi, err := os.Stat(m.model)
	if !isRegenerated && err == nil && !fi.IsDir() {
		log.Println("Skip retrain NLU:", "error:", err, "config regenerated:", isRegenerated)
		return nil
	}

	cmd := exec.Command(
		"rasa", "train", "nlu",
		"-c", m.config,
		"-d", fmt.Sprintf("%s/generated/domain.yml", m.Workdir),
		"-u", fmt.Sprintf("%s/generated/nlu.yml", m.Workdir),
		"--out", genDir,
		"--fixed-model-name", "model-nlu-only",
	)
	cmd.Dir = m.Workdir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Println("Starting Rasa train...")
	if err := cmd.Run(); err != nil {
		log.Println("Training failed:", err)
		return err
	}
	log.Println("Training finished successfully")

	// перезапускаем процесс NLU с новой моделью
	m.Stop()
	return m.Start()
}
