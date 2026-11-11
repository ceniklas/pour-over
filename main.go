// pour-over asks for your password once, then runs `brew update` and
// `brew upgrade`, answering every sudo prompt along the way.
//
// Mechanism: Homebrew calls `sudo -A` when SUDO_ASKPASS is set. We point it at
// this same binary, which in askpass mode just prints the password.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	askpassModeEnv = "POUR_OVER_ASKPASS"
	passwordEnv    = "POUR_OVER_PW"
)

func main() {
	if os.Getenv(askpassModeEnv) == "1" {
		fmt.Println(os.Getenv(passwordEnv))
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

	env := append(os.Environ(),
		"SUDO_ASKPASS="+self,
		askpassModeEnv+"=1",
		passwordEnv+"="+password,
	)

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
