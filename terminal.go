package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh/terminal"
)

type webWriter struct{}

func (ww webWriter) Write(data []byte) (int, error) {
	Publish(data)
	return len(data), nil
}

func runPseudoTerminal() error {
	// Get the SHELL which is getting currently used
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	// Create arbitrary command.
	c := exec.Command(shell)
	// Start the command with a pty.
	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer ptmx.Close() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	defer close(ch)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.
	// Set stdin in raw mode.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(int(os.Stdin.Fd()), oldState) // Best effort.

	thenga2, _ := os.Create("/tmp/thenga2.txt")
	defer thenga2.Close()
	w := io.MultiWriter(ptmx)
	w2 := io.MultiWriter(os.Stdout, thenga2, new(webWriter))
	// Copy stdin to the pty and the pty to stdout.
	go func() {
		_, _ = io.Copy(w, os.Stdin)
	}()
	_, _ = io.Copy(w2, ptmx)
	return nil
}
