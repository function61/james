package servicespec

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/function61/gokit/assert"
)

type testcase struct {
	title          string
	input          string
	expectedOutput string
	expectedError  string
}

func TestSpecToCompose(t *testing.T) {
	tests := []testcase{
		caseFromFile("simple"),
		caseFromFile("kitchenSink"),
		caseFromFile("howToUpdateMissing"),
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
