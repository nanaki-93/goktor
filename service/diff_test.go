package service

import (
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
