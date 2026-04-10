// White-box test for streamBuildOutput and tarDir. The stream-parsing logic has
// many edge cases (success, error mid-stream, malformed JSON, nil writer) that
// are best verified directly rather than through the full ImageBuild call path
// which requires a Docker daemon.
package docker

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamBuildOutput_WritesStreamContent(t *testing.T) {
	input := `{"stream":"Step 1/3 : FROM alpine\n"}
{"stream":"Step 2/3 : RUN echo hello\n"}
{"stream":"Step 3/3 : CMD [\"sh\"]\n"}
`
	var output bytes.Buffer
	err := streamBuildOutput(strings.NewReader(input), &output)

	require.NoError(t, err)
	assert.Contains(t, output.String(), "Step 1/3")
	assert.Contains(t, output.String(), "Step 2/3")
	assert.Contains(t, output.String(), "Step 3/3")
}

func TestStreamBuildOutput_ReturnsImageBuildErrorOnFailure(t *testing.T) {
	input := `{"stream":"Step 1/2 : FROM alpine\n"}
{"error":"The command '/bin/sh -c bad' returned a non-zero code: 1"}
`
	var output bytes.Buffer
	err := streamBuildOutput(strings.NewReader(input), &output)

	var buildErr *ImageBuildError
	require.ErrorAs(t, err, &buildErr)
	assert.Contains(t, buildErr.Detail, "non-zero code")
	assert.Contains(t, output.String(), "Step 1/2")
}

func TestStreamBuildOutput_NilWriter(t *testing.T) {
	input := `{"stream":"Step 1/1 : FROM alpine\n"}
`
	err := streamBuildOutput(strings.NewReader(input), nil)

	assert.NoError(t, err)
}

func TestStreamBuildOutput_EmptyStream(t *testing.T) {
	var output bytes.Buffer
	err := streamBuildOutput(strings.NewReader(""), &output)

	assert.NoError(t, err)
	assert.Empty(t, output.String())
}

func TestTarDir_IncludesFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0o644))

	var buf bytes.Buffer
	err := tarDir(dir, &buf)
	require.NoError(t, err)

	names := tarEntryNames(t, &buf)
	assert.Contains(t, names, "Dockerfile")
	assert.Contains(t, names, "src")
	assert.Contains(t, names, "src/main.go")
}

func TestTarDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	err := tarDir(dir, &buf)
	require.NoError(t, err)

	names := tarEntryNames(t, &buf)
	assert.Empty(t, names)
}

func TestTarDir_PreservesFileContent(t *testing.T) {
	dir := t.TempDir()
	content := "FROM ubuntu:22.04\nRUN apt-get update\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0o644))

	var buf bytes.Buffer
	err := tarDir(dir, &buf)
	require.NoError(t, err)

	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", hdr.Name)

	data, err := io.ReadAll(tr)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

// tarEntryNames extracts all entry names from a tar archive.
func tarEntryNames(t *testing.T, r io.Reader) []string {
	t.Helper()
	tr := tar.NewReader(r)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
	return names
}
