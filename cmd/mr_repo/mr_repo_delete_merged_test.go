package mr_repo

import (
	"testing"
)

func TestDeleteMergedCmd_EmptyEndDate(t *testing.T) {
	err := deleteMergedCmd.RunE(deleteMergedCmd, []string{
		"",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "an end date arg is required"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestDeleteMergedCmd_ArgsValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no arguments",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "exact single argument",
			args:      []string{"2026-01-01"},
			wantError: false,
		},
		{
			name:      "too many arguments",
			args:      []string{"2026-01-01", "extra"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteMergedCmd.Args(deleteMergedCmd, tt.args)
			if (err != nil) != tt.wantError {
				t.Fatalf("Args() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestDeleteMergedCmd_Flags(t *testing.T) {
	flag := deleteMergedCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("expected 'dry-run' flag to be defined")
	}
	if flag.Shorthand != "d" {
		t.Fatalf("expected shorthand 'd', got %q", flag.Shorthand)
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected default value 'false', got %q", flag.DefValue)
	}
}
