package servicespec

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/go-yaml/yaml"
)

type testcase struct {
	title          string
	input          string
	expectedOutput string
	expectedError  string
}

func TestDeviceCgroupRules(t *testing.T) {
	input := `
service "regular" {
  image = "image"
  version = "latest"
  replicas = null
  how_to_update = "stop-old-first"
  restart_policy = "always"
  net { mode = "host" }
  ram_mb = 16
  devices = ["/dev/example:/dev/example"]
  device_cgroup_rules = ["c 189:* rwm", "b 7:* rm"]
}
global_service "global" {
  image = "image"
  version = "latest"
  how_to_update = "stop-old-first"
  restart_policy = "always"
  net { mode = "host" }
  ram_mb = 16
  device_cgroup_rules = ["c *:* r"]
}
service "empty" {
  image = "image"
  version = "latest"
  replicas = null
  how_to_update = "stop-old-first"
  restart_policy = "always"
  net { mode = "host" }
  ram_mb = 16
  device_cgroup_rules = []
}`
	output, err := specToCompose(strings.NewReader(input), Secrets{})
	if err != nil {
		t.Fatal(err)
	}
	// The Docker CLI Compose service type has no device cgroup rules field,
	// so decode generic service fields to inspect the emitted rules.
	var result struct {
		Services map[string]map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]string{
		"regular": {"c 189:* rwm", "b 7:* rm"},
		"global":  {"c *:* r"},
	} {
		rules, ok := result.Services[name]["device_cgroup_rules"].([]any)
		if !ok || len(rules) != len(want) {
			t.Fatalf("%s rules = %#v, want %v", name, result.Services[name]["device_cgroup_rules"], want)
		}
		for i, rule := range want {
			if rules[i] != rule {
				t.Errorf("%s rule %d = %#v, want %q", name, i, rules[i], rule)
			}
		}
	}
	if _, ok := result.Services["regular"]["devices"]; !ok {
		t.Error("regular service lost its devices")
	}
	if _, ok := result.Services["empty"]["device_cgroup_rules"]; ok {
		t.Error("empty rules should be omitted")
	}
}

func TestSpecToCompose(t *testing.T) {
	tests := []testcase{
		caseFromFile("simple"),
		caseFromFile("kitchenSink"),
		caseFromFile("howToUpdateMissing"),
		caseFromFile("restartPolicyMissing"),
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			actualOutput, err := specToCompose(bytes.NewBufferString(test.input), Secrets{})

			if test.expectedError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assert.EqualString(t, actualOutput, test.expectedOutput)
			} else {
				assert.Assert(t, err != nil)
				assert.EqualString(t, err.Error(), test.expectedError)
			}
		})
	}
}

func caseFromFile(title string) testcase {
	buf, err := os.ReadFile("testdata/" + title + ".txt")
	if err != nil {
		panic(err)
	}

	parts := strings.Split(string(buf), "\n-------------\n")
	if len(parts) != 2 {
		panic("invalid part length")
	}

	errorPrefix := "ERROR: "

	if strings.HasPrefix(parts[1], errorPrefix) {
		expectedError, err := strconv.Unquote(strings.TrimSuffix(parts[1][len(errorPrefix):], "\n"))
		if err != nil {
			panic(err)
		}

		return testcase{
			title:          title,
			input:          parts[0],
			expectedOutput: "",
			expectedError:  expectedError,
		}
	} else {
		return testcase{
			title:          title,
			input:          parts[0],
			expectedOutput: parts[1],
			expectedError:  "",
		}
	}
}
