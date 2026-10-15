package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute_EarlyErrorWithNilLogger(t *testing.T) {
	if os.Getenv("GOKTOR_TEST_EARLY_ERROR") == "1" {
		GlobalLogger = nil
		RootCmd.SetArgs([]string{"--unknown-flag-for-test"})
		Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestExecute_EarlyErrorWithNilLogger$")
	cmd.Env = append(os.Environ(), "GOKTOR_TEST_EARLY_ERROR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected process to exit with error, got: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	out := stderr.String()
	if strings.Contains(out, "panic:") {
		t.Errorf("expected no panic in stderr, got: %s", out)
	}
	if !strings.Contains(out, "Error: unknown flag: --unknown-flag-for-test") {
		t.Errorf("expected stderr to contain unknown flag error, got: %s", out)
	}
}

func TestExecute_ErrorAfterPreRunInitializesLogger(t *testing.T) {
	if os.Getenv("GOKTOR_TEST_PRERUN_ERROR") == "1" {
		GlobalLogger = nil
		failCmd := &cobra.Command{
			Use: "fail-test-cmd",
			RunE: func(cmd *cobra.Command, args []string) error {
				return errors.New("subcommand failure")
			},
		}
		RootCmd.AddCommand(failCmd)
		RootCmd.SetArgs([]string{"fail-test-cmd"})
		Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestExecute_ErrorAfterPreRunInitializesLogger$")
	cmd.Env = append(os.Environ(), "GOKTOR_TEST_PRERUN_ERROR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected process to exit with error, got: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	out := stderr.String()
	if strings.Contains(out, "panic:") {
		t.Errorf("expected no panic in stderr, got: %s", out)
	}
	if !strings.Contains(out, "Error: subcommand failure") {
		t.Errorf("expected stderr to contain subcommand error, got: %s", out)
	}
}
