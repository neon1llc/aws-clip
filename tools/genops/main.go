// main.go - Generate the sensitive-output operation list from AWS service models

// Command genops reads botocore service models and writes the read-style operations
// that return credential-like data, so classification does not rest on memory.
//
// Usage: go run ./tools/genops -models DIR [-check]
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Regular expressions split operation names into the words AWS CLI v2 joins with hyphens
var (
	firstCap    = regexp.MustCompile(`(.)([A-Z][a-z]+)`)
	endCap      = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	specialCase = regexp.MustCompile(`[A-Z]{2,}s$`)
)

// Brand spellings botocore keeps whole because the general rule would split them
var cliNameWords = strings.NewReplacer(
	"iSCSI", "Iscsi",
	"HITs", "Hits",
	"PartiQL", "Partiql",
	"WhatsApp", "Whatsapp",
	"OTel", "Otel",
)

// Whole names botocore spells by hand
var cliNameOverrides = map[string]string{
	"CreateOAuth2Token":            "create-oauth2-token",
	"CreateOAuth2TokenWithIAM":     "create-oauth2-token-with-iam",
	"IntrospectOAuth2TokenWithIAM": "introspect-oauth2-token-with-iam",
	"RevokeOAuth2TokenWithIAM":     "revoke-oauth2-token-with-iam",
}

// Operation words the wrapper treats as read-only, matching internal/cli
var (
	readExact    = []string{"describe", "get", "head", "list", "lookup", "ls", "scan", "search", "select", "tail", "validate", "wait"}
	readPrefixes = []string{"batch-get-", "describe-", "get-", "head-", "list-", "lookup-", "search-", "validate-"}
)

// Member names that suggest credential material, and names that only identify or locate it
var (
	credentialName = regexp.MustCompile(`(?i)secret|password|passphrase|credential|private.?key|access.?key|session.?key|security.?key|api.?key|token|authorization|signing.?key|psk|preshared`)
	identifierName = regexp.MustCompile(`(?i)(arn|id|url|uri|name|type|location|expiration|time)$|^(next|previous|continuation|pagination|max)|marker$`)
)

type model struct {
	Operations map[string]struct {
		Output *struct {
			Shape string `json:"shape"`
		} `json:"output"`
	} `json:"operations"`
	Shapes map[string]shape `json:"shapes"`
}

type shape struct {
	Type      string            `json:"type"`
	Sensitive bool              `json:"sensitive"`
	Members   map[string]member `json:"members"`
	Member    *member           `json:"member"`
	Value     *member           `json:"value"`
}

type member struct {
	Shape string `json:"shape"`
}

type reviewed struct {
	NotReturned    map[string]string `json:"not_returned"`
	ManualPatterns map[string]string `json:"manual_patterns"`
}

// cliOperationName converts a model operation name to the AWS CLI command name
func cliOperationName(name string) string {
	if override, ok := cliNameOverrides[name]; ok {
		return override
	}
	name = cliNameWords.Replace(name)
	if match := specialCase.FindString(name); match != "" {
		name = name[:len(name)-len(match)] + "-" + strings.ToLower(match)
	}
	split := firstCap.ReplaceAllString(name, "${1}-${2}")
	return strings.ToLower(endCap.ReplaceAllString(split, "${1}-${2}"))
}

// isReadStyle matches the operation words internal/cli classifies as read-only
func isReadStyle(operation string) bool {
	for _, exact := range readExact {
		if operation == exact {
			return true
		}
	}
	for _, prefix := range readPrefixes {
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	return false
}

// returnsCredential reports whether an output shape holds a model-sensitive member with a credential-like name
func returnsCredential(shapes map[string]shape, name string, seen map[string]bool) bool {
	if seen[name] {
		return false
	}
	seen[name] = true
	current := shapes[name]
	switch current.Type {
	case "structure":
		for field, child := range current.Members {
			if shapes[child.Shape].Sensitive && credentialName.MatchString(field) && !identifierName.MatchString(field) {
				return true
			}
			if returnsCredential(shapes, child.Shape, seen) {
				return true
			}
		}
	case "list":
		return current.Member != nil && returnsCredential(shapes, current.Member.Shape, seen)
	case "map":
		return current.Value != nil && returnsCredential(shapes, current.Value.Shape, seen)
	}
	return false
}

// readModel loads the newest dated service model in a service directory, plain or gzipped
func readModel(directory string) (*model, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && regexp.MustCompile(`^\d{4}-\d\d-\d\d$`).MatchString(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	if len(versions) == 0 {
		return nil, nil
	}
	sort.Strings(versions)
	base := filepath.Join(directory, versions[len(versions)-1])
	var reader io.Reader
	if file, err := os.Open(filepath.Join(base, "service-2.json")); err == nil {
		defer file.Close()
		reader = file
	} else if file, err := os.Open(filepath.Join(base, "service-2.json.gz")); err == nil {
		defer file.Close()
		if reader, err = gzip.NewReader(file); err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}
	var result model
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s: %w", base, err)
	}
	return &result, nil
}

// candidates returns every service:operation pair that is read-style and returns credential-like data
func candidates(modelsDirectory string) (map[string]bool, int, error) {
	services, err := os.ReadDir(modelsDirectory)
	if err != nil {
		return nil, 0, err
	}
	found := make(map[string]bool)
	operations := 0
	for _, service := range services {
		if !service.IsDir() || strings.HasPrefix(service.Name(), "_") {
			continue
		}
		loaded, err := readModel(filepath.Join(modelsDirectory, service.Name()))
		if err != nil {
			return nil, 0, err
		}
		if loaded == nil {
			continue
		}
		for name, operation := range loaded.Operations {
			operations++
			command := cliOperationName(name)
			if operation.Output == nil || !isReadStyle(command) {
				continue
			}
			if returnsCredential(loaded.Shapes, operation.Output.Shape, map[string]bool{}) {
				found[service.Name()+":"+command] = true
			}
		}
	}
	return found, operations, nil
}

// render builds the Go source for the reviewed lists
func render(source string, operations []string, patterns []string) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString("// sensitive_operations_gen.go - Operations that return credential-like data\n\n")
	out.WriteString("// Code generated by tools/genops from botocore " + source + "; DO NOT EDIT.\n")
	out.WriteString("// Run make generate after reviewing tools/genops/reviewed.json.\n\n")
	out.WriteString("package cli\n\n")
	out.WriteString("// Botocore release whose service models produced the list below\n")
	out.WriteString("const generatedFromBotocore = \"" + source + "\"\n\n")
	out.WriteString("// Read-style operations whose models return a credential-like member, minus reviewed exceptions\n")
	out.WriteString("var generatedSensitiveOperations = map[string]bool{\n")
	for _, operation := range operations {
		fmt.Fprintf(&out, "\t%q: true,\n", operation)
	}
	out.WriteString("}\n\n")
	out.WriteString("// Patterns kept by hand for commands the models cannot show\n")
	out.WriteString("var manualSensitivePatterns = []string{\n")
	for _, pattern := range patterns {
		fmt.Fprintf(&out, "\t%q,\n", pattern)
	}
	out.WriteString("}\n")
	return format.Source(out.Bytes())
}

// sortedKeys returns a map's keys in order
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// committedVersion reads the botocore release recorded in the existing generated file
func committedVersion(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	match := regexp.MustCompile(`const generatedFromBotocore = "([^"]+)"`).FindSubmatch(content)
	if match == nil {
		return ""
	}
	return string(match[1])
}

// committedOperations reads the operations listed in the existing generated file
func committedOperations(path string) map[string]bool {
	content, _ := os.ReadFile(path)
	listed := make(map[string]bool)
	for _, match := range regexp.MustCompile(`(?m)^\t"([a-z0-9-]+:[a-z0-9-]+)":\s+true,$`).FindAllSubmatch(content, -1) {
		listed[string(match[1])] = true
	}
	return listed
}

// run generates or checks the file and returns the process exit status
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("genops", flag.ContinueOnError)
	flags.SetOutput(stderr)
	models := flags.String("models", "", "directory of botocore service models (the botocore/data folder)")
	version := flags.String("botocore-version", "", "botocore release the models come from (default: the version in the existing file)")
	out := flags.String("out", "internal/cli/sensitive_operations_gen.go", "generated Go file")
	review := flags.String("reviewed", "tools/genops/reviewed.json", "reviewed exceptions and manual patterns")
	check := flags.Bool("check", false, "fail when the generated file differs, listing new and removed operations")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *models == "" {
		fmt.Fprintln(stderr, "genops: -models is required")
		return 2
	}
	source := *version
	if source == "" {
		source = committedVersion(*out)
	}
	if source == "" {
		fmt.Fprintln(stderr, "genops: -botocore-version is required when no generated file exists")
		return 2
	}

	rawReview, err := os.ReadFile(*review)
	var decisions reviewed
	if err == nil {
		err = json.Unmarshal(rawReview, &decisions)
	}
	if err != nil {
		fmt.Fprintf(stderr, "genops: %v\n", err)
		return 1
	}
	found, total, err := candidates(*models)
	if err != nil {
		fmt.Fprintf(stderr, "genops: %v\n", err)
		return 1
	}

	// A review entry for an operation that is not a candidate is stale or mistyped
	failed := false
	for operation := range decisions.NotReturned {
		if !found[operation] {
			fmt.Fprintf(stderr, "genops: not_returned lists %s, which is not a candidate in these models\n", operation)
			failed = true
		}
		delete(found, operation)
	}
	content, err := render(source, sortedKeys(found), sortedKeys(decisions.ManualPatterns))
	if err != nil {
		fmt.Fprintf(stderr, "genops: %v\n", err)
		return 1
	}

	if *check {
		existing, _ := os.ReadFile(*out)
		if bytes.Equal(existing, content) && !failed {
			fmt.Fprintf(stdout, "genops: %s is current (%d sensitive operations from %d model operations)\n", *out, len(found), total)
			return 0
		}
		listed := committedOperations(*out)
		for _, operation := range sortedKeys(found) {
			if !listed[operation] {
				fmt.Fprintf(stderr, "genops: new sensitive-output operation %s: review it, then run make generate\n", operation)
			}
		}
		for _, operation := range sortedKeys(listed) {
			if !found[operation] {
				fmt.Fprintf(stderr, "genops: %s is listed but no longer a candidate: run make generate\n", operation)
			}
		}
		fmt.Fprintf(stderr, "genops: %s is out of date\n", *out)
		return 1
	}
	if failed {
		return 1
	}
	if err := os.WriteFile(*out, content, 0o644); err != nil {
		fmt.Fprintf(stderr, "genops: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "genops: wrote %d sensitive operations from %d model operations to %s\n", len(found), total, *out)
	return 0
}

// main exits with the status run reports
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
