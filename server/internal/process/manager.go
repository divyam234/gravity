package process

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Process struct {
	Name    string
	Args    []string
	Cmd     *exec.Cmd
	Running bool
	mu      sync.Mutex
	done    chan struct{}
}

func New(name string, args []string) *Process {
	return &Process{
		Name: name,
		Args: args,
	}
}

func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Running {
		return fmt.Errorf("%s is already running", p.Name)
	}

	p.Cmd = exec.Command(p.Name, p.Args...)
	p.Cmd.Stdout = os.Stdout
	p.Cmd.Stderr = os.Stderr
	p.done = make(chan struct{})

	if err := p.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", p.Name, err)
	}

	// Only set Running after Start() succeeds
	p.Running = true

	go func() {
		_ = p.Cmd.Wait()
		p.mu.Lock()
		p.Running = false
		p.mu.Unlock()
		close(p.done)
		fmt.Printf("%s exited\n", p.Name)
	}()

	return nil
}

func (p *Process) Stop() error {
	p.mu.Lock()
	if !p.Running || p.Cmd == nil {
		p.mu.Unlock()
		return nil
	}
	done := p.done
	p.mu.Unlock()

	// Try graceful shutdown first
	if err := p.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = p.Cmd.Process.Kill()
	}

	// Wait for process to exit with timeout
	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(5 * time.Second):
		// Force kill after timeout
		_ = p.Cmd.Process.Kill()
		<-done
	}

	p.mu.Lock()
	p.Running = false
	p.mu.Unlock()

	return nil
}

func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Running
}
