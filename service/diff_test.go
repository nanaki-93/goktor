package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nanaki-93/goktor/model"
)

func TestCompareUsesGroupedDuplicateKeys(t *testing.T) {
	left := map[string]model.DiffContent{
		"duplicated":   {Content: "left", Type: "text"},
		"duplicated-2": {Content: "right", Type: "text"},
	}
	right := map[string]model.DiffContent{
		"duplicated":   {Content: "other", Type: "text"},
		"duplicated-2": {Content: "right", Type: "text"},
	}

	result, err := compare(left, right)
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}

	if result.KO["duplicated"] != "DIFF" {
		t.Errorf("got %q, want DIFF", result.KO["duplicated"])
	}

	if result.OK["duplicated-2"] != "OK" {
		t.Errorf("got %q, want OK", result.OK["duplicated-2"])
	}
}

func TestCompareUsesCanonicalJSON(t *testing.T) {
	left := map[string]model.DiffContent{
		"json": {Content: `{"a":1,"b":2}`, Type: model.JsonType},
	}
	right := map[string]model.DiffContent{
		"json": {Content: `{"b":2,"a":1}`, Type: model.JsonType},
	}

	result, err := compare(left, right)
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}

	if result.OK["json"] != "OK" {
		t.Errorf("got %q, want OK", result.OK["json"])
	}
}

func TestCompareMissingKeysOnEitherSide(t *testing.T) {
	left := map[string]model.DiffContent{
		"only-left":  {Content: "val", Type: "text"},
		"empty-left": {Content: "", Type: ""},
		"both":       {Content: "same", Type: "text"},
	}
	right := map[string]model.DiffContent{
		"only-right":  {Content: "val", Type: "text"},
		"empty-right": {Content: "", Type: ""},
		"both":        {Content: "same", Type: "text"},
	}

	result, err := compare(left, right)
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}

	if len(result.OK) != 1 || result.OK["both"] != "OK" {
		t.Errorf("expected only 'both' in OK, got %v", result.OK)
	}
	if _, ok := result.OK["empty-left"]; ok {
		t.Errorf("expected missing key 'empty-left' not to be in OK")
	}
	if val, ok := result.KO["only-left"]; !ok || strings.Contains(val, "types are not equal") {
		t.Errorf("expected 'only-left' in KO without type error, got %q (present=%v)", val, ok)
	}
	if val, ok := result.KO["empty-left"]; !ok || val == "" {
		t.Errorf("expected 'empty-left' in KO, got %q (present=%v)", val, ok)
	}
	if val, ok := result.KO["only-right"]; !ok || val == "" {
		t.Errorf("expected 'only-right' in KO, got %q (present=%v)", val, ok)
	}
	if val, ok := result.KO["empty-right"]; !ok || val == "" {
		t.Errorf("expected 'empty-right' in KO, got %q (present=%v)", val, ok)
	}
}

func TestCompareOneSideEmpty(t *testing.T) {
	leftOnly := map[string]model.DiffContent{
		"k1": {Content: "v1", Type: "text"},
	}
	result, err := compare(leftOnly, map[string]model.DiffContent{})
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}
	if len(result.OK) != 0 || len(result.KO) != 1 {
		t.Errorf("expected 0 OK and 1 KO for empty right map, got OK=%v KO=%v", result.OK, result.KO)
	}

	rightOnly := map[string]model.DiffContent{
		"k2": {Content: "v2", Type: "text"},
	}
	result, err = compare(map[string]model.DiffContent{}, rightOnly)
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}
	if len(result.OK) != 0 || len(result.KO) != 1 {
		t.Errorf("expected 0 OK and 1 KO for empty left map, got OK=%v KO=%v", result.OK, result.KO)
	}
}

func TestPopulateMapStreamingAndValidation(t *testing.T) {
	tmpDir := t.TempDir()
	svc := &DiffFileService{logger: &DefaultLogger{}}

	validPath := filepath.Join(tmpDir, "valid.csv")
	validCSV := "id\tcontent\ttype\nk1\t'val1'\ttext\nk2\tval2\tjson\n"
	if err := os.WriteFile(validPath, []byte(validCSV), 0644); err != nil {
		t.Fatalf("failed to write test csv: %v", err)
	}

	m, err := svc.populateMap(validPath, model.DiffConfig{Delimiter: '\t', WithHeader: true})
	if err != nil {
		t.Fatalf("populateMap failed: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m["k1"].Content != "val1" || m["k1"].Type != "text" {
		t.Errorf("unexpected k1 entry: %+v", m["k1"])
	}
	if m["k2"].Content != "val2" || m["k2"].Type != "json" {
		t.Errorf("unexpected k2 entry: %+v", m["k2"])
	}

	shortPath := filepath.Join(tmpDir, "short.csv")
	if err := os.WriteFile(shortPath, []byte("id\tcontent\nonly_two\tcols\n"), 0644); err != nil {
		t.Fatalf("failed to write short csv: %v", err)
	}
	if _, err := svc.populateMap(shortPath, model.DiffConfig{Delimiter: '\t', WithHeader: true}); err == nil || !strings.Contains(err.Error(), "invalid row 2") {
		t.Errorf("expected invalid row 2 error, got %v", err)
	}

	dupPath := filepath.Join(tmpDir, "dup.csv")
	if err := os.WriteFile(dupPath, []byte("dup\tv1\ttext\ndup\tv2\ttext\n"), 0644); err != nil {
		t.Fatalf("failed to write dup csv: %v", err)
	}
	if _, err := svc.populateMap(dupPath, model.DiffConfig{Delimiter: '\t', WithHeader: false}); err == nil || !strings.Contains(err.Error(), "duplicate key dup") {
		t.Errorf("expected duplicate key error, got %v", err)
	}
}
