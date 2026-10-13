// Package launch is the one way an Againrom program starts another: the
// starter starts the game and the map editor, and the map editor starts the
// game for a test play. A launch is data: the program, its arguments and the
// directory it runs in.
package launch

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

// Program is one launch: the program file, its arguments without the program
// name, and the working directory.
type Program struct {
	Path string
	Args []string
	Dir  string
}

// For is the launch of the program at path with args, run in the program's
// own folder.
func For(path string, args []string) Program {
	return Program{Path: path, Args: append([]string(nil), args...), Dir: filepath.Dir(path)}
}

// Beside is the program name in the folder of the program self: name with
// ".exe" on Windows.
func Beside(self, name, goos string) string {
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(self), name)
}

// Command builds the process of p under ctx; a console program gets no console
// window.
func (p Program) Command(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, p.Path, p.Args...)
	cmd.Dir = p.Dir
	HideConsole(cmd)
	return cmd
}

// Start launches p and returns without waiting for it to end.
func (p Program) Start() error {
	cmd := p.Command(context.Background())
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// Run starts p, waits at most limit for it and returns its combined output.
func (p Program) Run(limit time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := p.Command(ctx)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}
