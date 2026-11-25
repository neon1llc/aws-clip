// main_test.go - Test the model reader and the CLI name conversion

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Names whose conversion trips a naive split, checked without botocore installed
func TestCLIOperationNameSpecialCases(t *testing.T) {
	for name, want := range map[string]string{
		"GetSecretValue":              "get-secret-value",
		"DescribeDBInstances":         "describe-db-instances",
		"ListWebACLs":                 "list-web-acls",
		"DescribeStorediSCSIVolumes":  "describe-stored-iscsi-volumes",
		"CreateOAuth2Token":           "create-oauth2-token",
		"ListWhatsAppTemplateLibrary": "list-whatsapp-template-library",
		"GetOTelEnrichment":           "get-otel-enrichment",
	} {
		if got := cliOperationName(name); got != want {
			t.Errorf("cliOperationName(%q) = %q, want %q", name, got, want)
		}
	}
}

// Every operation in the installed botocore must convert as botocore does; skipped without a names file
func TestCLIOperationNamesMatchBotocore(t *testing.T) {
	path := os.Getenv("GENOPS_NAMES")
	if path == "" {
		t.Skip("GENOPS_NAMES is not set")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	if err := json.Unmarshal(content, &names); err != nil {
		t.Fatal(err)
	}
	for name, want := range names {
		if got := cliOperationName(name); got != want {
			t.Errorf("cliOperationName(%q) = %q, want %q", name, got, want)
		}
	}
	t.Logf("compared %d operation names", len(names))
}

// A tiny model checks which outputs count as returning credentials
func TestCandidatesFromModel(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "demo", "2020-01-01")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	model := `{"operations":{
	  "GetSecretThing":{"output":{"shape":"Secret"}},
	  "GetNote":{"output":{"shape":"Note"}},
	  "ListNames":{"output":{"shape":"Names"}},
	  "CreateKey":{"output":{"shape":"Secret"}},
	  "GetPageToken":{"output":{"shape":"Page"}},
	  "GetNested":{"output":{"shape":"Outer"}}},
	 "shapes":{
	  "Secret":{"type":"structure","members":{"SecretString":{"shape":"Hidden"}}},
	  "Note":{"type":"structure","members":{"Name":{"shape":"Hidden"}}},
	  "Names":{"type":"structure","members":{"Items":{"shape":"NameList"}}},
	  "NameList":{"type":"list","member":{"shape":"Note"}},
	  "Page":{"type":"structure","members":{"NextToken":{"shape":"Hidden"}}},
	  "Outer":{"type":"structure","members":{"Inner":{"shape":"Secret"}}},
	  "Hidden":{"type":"string","sensitive":true}}}`
	if err := os.WriteFile(filepath.Join(directory, "service-2.json"), []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	reads, found, _, err := candidates(filepath.Dir(filepath.Dir(directory)))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for operation := range found {
		got = append(got, operation)
	}
	want := map[string]bool{"demo:get-secret-thing": true, "demo:get-nested": true}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", strings.Join(got, ","), want)
	}
	for _, operation := range got {
		if !want[operation] {
			t.Errorf("unexpected candidate %s", operation)
		}
	}
	for _, operation := range []string{"demo:get-secret-thing", "demo:get-note", "demo:list-names", "demo:get-page-token", "demo:get-nested"} {
		if !reads[operation] {
			t.Errorf("missing exact read operation %s", operation)
		}
	}
}
