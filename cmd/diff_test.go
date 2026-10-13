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

func TestDiffCmd_MultiCharDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
	}{
		{name: "repeated ascii", delimiter: ",,"},
		{name: "multi-char string", delimiter: "csv"},
		{name: "multi-rune unicode", delimiter: "§§"},
		{name: "multi-rune emoji", delimiter: "🎉🎊"},
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
			if err == nil {
				t.Fatalf("expected error when delimiter %q is multi-character, got nil", tc.delimiter)
			}
			if err.Error() != "delimiter must be a single character" {
				t.Fatalf("unexpected error message for %q: got %q, want %q", tc.delimiter, err.Error(), "delimiter must be a single character")
			}
		})
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
			if err != nil && (err.Error() == "delimiter cannot be empty" || err.Error() == "delimiter must be a single character") {
				t.Fatalf("unexpected delimiter validation error for delimiter %q: %v", tc.delimiter, err)
			}
		})
	}
}
