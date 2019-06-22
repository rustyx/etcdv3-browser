package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnv(t *testing.T) {
	require.NotNil(t, env("non_existent", "", ""), "")
	require.NotEqual(t, env("PATH", "", ""), "", "PATH expected to exist")
	require.Equal(t, env("non_existent_42", "def_value", ""), "def_value", "expected to not exist")
}

func TestEnvInt(t *testing.T) {
	os.Setenv("existent_value", "42")
	require.Equal(t, envInt("existent_value", 3, ""), 42, "expected to receive parsed environment value")
	os.Unsetenv("existent_value")
	require.Equal(t, envInt("existent_value", 3, ""), 3, "expected to receive default value")
}

func TestEnvIntCrasher(t *testing.T) {
	if os.Getenv("crasher_latch") == "1" {
		os.Setenv("existent_value", "42blah") // bad int
		envInt("existent_value", 2, "")       // expecting an os.Exit() here
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestEnvIntCrasher")
	cmd.Env = append(os.Environ(), "crasher_latch=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
