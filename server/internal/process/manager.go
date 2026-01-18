package process

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type Process struct {
	Name    string
	Args    []string
	Cmd     *exec.Cmd
	Running bool
	mu      sync.Mutex
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
	// p.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Allow killing process group

	if err := p.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", p.Name, err)
	}

	p.Running = true
	go func() {
		_ = p.Cmd.Wait()
		p.mu.Lock()
		p.Running = false
		p.mu.Unlock()
		fmt.Printf("%s exited\n", p.Name)
	}()

	return nil
}

func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Running || p.Cmd == nil {
		return nil
	}

	// Try to kill gracefully first
	if err := p.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Force kill if needed
		_ = p.Cmd.Process.Kill()
	}

	p.Running = false
	return nil
}
