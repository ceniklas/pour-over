package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The helper must work with an empty environment: Homebrew strips most env
// vars when it shells out to sudo during cask uninstalls.
func TestAskpassWithEmptyEnv(t *testing.T) {
	bin := t.TempDir() + "/pour-over"
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	script, cleanup, err := serveAskpass(bin, "s3cret pw")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	cmd := exec.Command(script, "Password:")
	cmd.Env = []string{} // nothing at all
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "s3cret pw" {
		t.Fatalf("got %q", got)
	}

	cleanup()
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatal("temp dir not removed")
	}
}
