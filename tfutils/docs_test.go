package tfutils

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImport(t *testing.T) {
	var tests = []struct {
		input    []string
		expected string
	}{
		{
			input:    readlines(t, "testdata/testImport.md"),
			expected: "import {\n\t\t\t\tto = btp_subaccount.<resource_name>\n\t\t\t\tid = \"<subaccount_id>\"\n\t\t\t  }",
		},
	}

	for _, tt := range tests {
		parser := tfMarkdownParser{}
		parser.parseImport(tt.input)
		actual := parser.ret.Import
		assert.Equal(t, tt.expected, actual)
	}
}

func TestParseArgumentFromMarkdownLine(t *testing.T) {
	//nolint:lll
	tests := []struct {
		input         string
		expectedName  string
		expectedDesc  string
		expectedFound bool
	}{
		{"* `name` (String) A descriptive name of the subaccount.", "name", "A descriptive name of the subaccount.", true},
		{"* `region` (String) The region of the subaccount.", "region", "The region of the subaccount.", true},
		{"  * `subaccount_id` - The id of the subaccount", "subaccount_id", "The id of the subaccount", true},
		{"  * id - The id of the subaccount", "", "", false},
	}

	for _, test := range tests {
		name, desc, isFound := parseArgumentFromMarkdownLine(test.input)
		assert.Equal(t, test.expectedName, name)
		assert.Equal(t, test.expectedDesc, desc)
		assert.Equal(t, test.expectedFound, isFound)
	}
}

func TestParseAttrReferenceSection(t *testing.T) {
	ret := entityDocs{
		Arguments:  make(map[string]*argumentDocs),
		Attributes: make(map[string]string),
	}
	parseAttrReferenceSection([]string{
		"The following attributes are exported:",
		"",
		"* `id` - The ID of the subaccount",
		"* `name`- The name of the subaccount",
		"* `region` - The region of the subaccount",
	}, &ret)
	assert.Len(t, ret.Attributes, 3)
}

func TestParseTFMarkdown(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		kind     DocKind
		rawName  string
		fileName string
	}

	test := func(name string, configure ...func(tc *testCase)) testCase {
		tc := testCase{
			name:     name,
			kind:     "resouces",
			rawName:  "btp_subaccount",
			fileName: "subaccount.md",
		}
		for _, c := range configure {
			c(&tc)
		}
		return tc
	}

	tests := []testCase{
		test("subaccount"),
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NotZero(t, tt.name)
			input := testFilePath(tt.name, "input_subaccount.md")
			expected := filepath.Join(tt.name, "expected_subaccount.json")
			p := &tfMarkdownParser{
				kind:             tt.kind,
				markdownFileName: tt.fileName,
				rawname:          tt.rawName,
			}

			inputBytes, err := os.ReadFile(input)
			require.NoError(t, err)

			actual, err := p.parse(inputBytes)
			require.NoError(t, err)

			actualBytes, err := json.MarshalIndent(actual, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			compareTestFile(t, expected, string(actualBytes), assert.JSONEq)
		})
	}
}

func compareTestFile(
	t *testing.T, path, actual string,
	comp func(t assert.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool,
) {
	comp(t, readTestFile(t, path), actual)
}

func readTestFile(t *testing.T, name string) string {
	bytes, err := os.ReadFile(testFilePath(name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Replace(string(bytes), "\r\n", "\n", -1)
}

func readlines(t *testing.T, file string) []string {
	t.Helper()
	f, err := os.Open(file)
	require.NoError(t, err)
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func testFilePath(path ...string) string {
	return filepath.Join(append([]string{"testdata"}, path...)...)
}
