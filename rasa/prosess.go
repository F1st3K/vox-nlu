package rasa

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type NLUProcess struct {
	cmd     *exec.Cmd
	stdin   io.Writer
	scanner *bufio.Scanner
	mu      sync.Mutex
	model   string
	rasaBin string
}

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

func NewNLUProcess(rasaBin, model string) *NLUProcess {
	return &NLUProcess{
		rasaBin: rasaBin,
		model:   model,
	}
}

func (p *NLUProcess) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd := exec.Command(
		p.rasaBin, "shell", "nlu",
		"--model", p.model,
		"--quiet",
		"--json",
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

	p.cmd = cmd
	p.stdin = stdin
	p.scanner = bufio.NewScanner(stdout)

	return nil
}

func (p *NLUProcess) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
		p.cmd.Wait()
		p.cmd = nil
	}
}

func (p *NLUProcess) Parse(text string) (*NLUResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintln(p.stdin, text)

	for p.scanner.Scan() {
		line := p.scanner.Text()
		var res NLUResult
		if json.Unmarshal([]byte(line), &res) == nil && res.Intent.Name != "" {
			return &res, nil
		}
	}
	return nil, io.EOF
}

func NewManager(workdir string) *Manager {
	return &Manager{Workdir: workdir}
}

func (m *Manager) Train(intents []Intent) {
}
