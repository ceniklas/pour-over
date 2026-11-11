// pour-over asks for your password once, then runs `brew update` and
// `brew upgrade`, answering every sudo prompt along the way.
//
// Mechanism: Homebrew calls `sudo -A` when SUDO_ASKPASS is set. We point it at
// a tiny wrapper script that runs this same binary in askpass mode, which
// fetches the password from a private unix socket served by the main process.
// Homebrew strips most of the environment for some steps (e.g. cask
// uninstalls), so nothing here may depend on env vars reaching the helper.
package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// askpassArg marks an invocation as the sudo askpass helper; the socket path
// follows it.
const askpassArg = "__askpass__"

func main() {
	if len(os.Args) > 2 && os.Args[1] == askpassArg {
		if err := askpass(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "pour-over askpass:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	password, err := readPassword()
	if err != nil {
		return err
	}
	if err := verifyPassword(password); err != nil {
		return err
	}

	askpassPath, cleanup, err := serveAskpass(self, password)
	if err != nil {
		return err
	}
	defer cleanup()

	env := append(os.Environ(), "SUDO_ASKPASS="+askpassPath)

	stop := keepSudoAlive(env)
	defer stop()

	// Let brew handle Ctrl-C itself; we just wait for it to exit.
	signal.Ignore(syscall.SIGINT)

	steps := [][]string{
		{"update"},
		{"upgrade", "--no-ask"},
	}
	if len(os.Args) > 1 {
		steps = [][]string{os.Args[1:]}
	}
	for _, args := range steps {
		fmt.Printf("\n==> brew %s\n", strings.Join(args, " "))
		cmd := exec.Command("brew", args...)
		cmd.Env = env
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("brew %s: %w", args[0], err)
		}
	}
	return nil
}

// serveAskpass listens on a unix socket inside a private (0700) temp dir and
// hands the password to whoever connects, then writes the wrapper script that
// sudo should use as SUDO_ASKPASS. The password only ever lives in memory.
func serveAskpass(self, password string) (scriptPath string, stop func(), err error) {
	dir, err := os.MkdirTemp("", "pour-over-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	sock := filepath.Join(dir, "s")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("listening on %s: %w", sock, err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			fmt.Fprintln(conn, password)
			conn.Close()
		}
	}()

	scriptPath = filepath.Join(dir, "askpass")
	script := fmt.Sprintf("#!/bin/sh\nexec %q %s %q\n", self, askpassArg, sock)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		ln.Close()
		cleanup()
		return "", nil, err
	}
	return scriptPath, func() { ln.Close(); cleanup() }, nil
}

// askpass is the helper side: print whatever the main process serves.
func askpass(sock string) error {
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = io.Copy(os.Stdout, conn)
	return err
}

func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(b), nil
}

// verifyPassword fails fast on a typo, and requires admin rights up front.
func verifyPassword(password string) error {
	_ = exec.Command("sudo", "-k").Run() // drop any cached credentials so the check is real
	cmd := exec.Command("sudo", "-S", "-p", "", "-v")
	cmd.Stdin = strings.NewReader(password + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo rejected the password (do you have admin access yet?): %s",
			strings.TrimSpace(stderr.String()))
	}
	return nil
}

// keepSudoAlive refreshes sudo's timestamp so plain `sudo` calls made by
// installers (which don't use -A) also stay authenticated.
func keepSudoAlive(env []string) (stop func()) {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				cmd := exec.Command("sudo", "-A", "-v")
				cmd.Env = env
				_ = cmd.Run()
			}
		}
	}()
	return func() { close(done) }
}
