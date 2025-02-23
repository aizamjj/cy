package cy

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/janet"

	"github.com/stretchr/testify/require"
)

import _ "embed"

type testFailure struct {
	File  string
	Name  string
	Error error
}

//go:embed api_test.janet
var API_TEST_FILE []byte

func runTestFile(t *testing.T, file string) (failures []testFailure) {
	server, create, err := NewTestServer()
	require.NoError(t, err)

	server.Callback("run-test", "", func(
		name string,
		// whether a client should be included
		context bool,
		callback *janet.Function,
	) {
		var callContext interface{}
		if context {
			client, err := create(geom.DEFAULT_SIZE)
			require.NoError(t, err)
			callContext = client
			defer client.Cancel()
		}

		// Clears out and resets the node tree on every run
		defer server.tree.Reset()

		err := callback.CallContext(server.Ctx(), callContext)
		if err == nil {
			return
		}

		failures = append(failures, testFailure{
			Name:  name,
			File:  file,
			Error: err,
		})
	})

	_, err = server.ExecuteCall(server.Ctx(), nil, janet.Call{
		Code:       API_TEST_FILE,
		SourcePath: "api_test.janet",
		Options:    janet.DEFAULT_CALL_OPTIONS,
	})
	require.NoError(t, err)

	err = server.ExecuteFile(server.Ctx(), file)
	require.NoError(t, err)
	return
}

func TestAPI(t *testing.T) {
	_, apiTestFile, _, _ := runtime.Caller(0)
	tests, err := filepath.Glob(filepath.Join(
		filepath.Dir(apiTestFile),
		"api",
		"*_test.janet",
	))
	require.NoError(t, err)

	var failures []testFailure
	for _, testFile := range tests {
		failures = append(failures, runTestFile(t, testFile)...)
	}

	if len(failures) == 0 {
		return
	}

	for _, failure := range failures {
		t.Logf("%s: '%s' failed: %+v", failure.File, failure.Name, failure.Error)
	}

	t.Errorf("%d API test(s) failed", len(failures))
}
