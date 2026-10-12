package cmd

import (
	"testing"
)

func TestDiffCmd_EmptyDelimiter(t *testing.T) {
	if err := diffCmd.Flags().Set("delimiter", ""); err != nil {
		t.Fatalf("failed to set delimiter flag: %v", err)
	}
	defer func() {
		_ = diffCmd.Flags().Set("delimiter", "\t")
	}()

	err := diffCmd.RunE(diffCmd, []string{})
	if err == nil {
		t.Fatal("expected error when delimiter is empty, got nil")
	}
	if err.Error() != "delimiter cannot be empty" {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), "delimiter cannot be empty")
	}
}

func TestDiffCmd_NonEmptyDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
	}{
		{name: "comma delimiter", delimiter: ","},
		{name: "tab delimiter", delimiter: "\t"},
		{name: "unicode delimiter", delimiter: "§"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := diffCmd.Flags().Set("delimiter", tc.delimiter); err != nil {
				t.Fatalf("failed to set delimiter flag: %v", err)
			}
			defer func() {
				_ = diffCmd.Flags().Set("delimiter", "\t")
			}()

			err := diffCmd.RunE(diffCmd, []string{})
			if err != nil && err.Error() == "delimiter cannot be empty" {
				t.Fatalf("unexpected empty delimiter error for delimiter %q", tc.delimiter)
			}
		})
	}
}
