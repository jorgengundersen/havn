package container_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/container"
)

func TestMergeNixRegistryAliases_PreservesExistingAndAddsMissing(t *testing.T) {
	existing := []byte(`{"version":2,"flakes":[{"from":{"type":"indirect","id":"flake:devenv"},"to":{"type":"github","owner":"old","repo":"env"}}]}`)
	incoming := []byte(`{"version":2,"flakes":[{"from":{"type":"indirect","id":"flake:devenv"},"to":{"type":"github","owner":"new","repo":"env"}},{"from":{"type":"indirect","id":"flake:nixpkgs"},"to":{"type":"github","owner":"NixOS","repo":"nixpkgs"}}]}`)

	merged, changed, err := container.MergeNixRegistryAliases(existing, incoming, "/state/nix/registry.json", "/legacy/nix/registry.json")

	require.NoError(t, err)
	assert.True(t, changed)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(merged, &doc))
	flakes, ok := doc["flakes"].([]any)
	require.True(t, ok)
	require.Len(t, flakes, 2)

	first, ok := flakes[0].(map[string]any)
	require.True(t, ok)
	to, ok := first["to"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "old", to["owner"])

	second, ok := flakes[1].(map[string]any)
	require.True(t, ok)
	from, ok := second["from"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "flake:nixpkgs", from["id"])
}

func TestMergeNixRegistryAliases_NoMissingAliases_ReturnsOriginalJSON(t *testing.T) {
	existing := []byte(`{"version":2,"flakes":[{"from":{"type":"indirect","id":"flake:devenv"},"to":{"type":"github","owner":"current","repo":"env"}}]}`)
	incoming := []byte(`{"version":2,"flakes":[{"from":{"type":"indirect","id":"flake:devenv"},"to":{"type":"github","owner":"different","repo":"env"}}]}`)

	merged, changed, err := container.MergeNixRegistryAliases(existing, incoming, "/state/nix/registry.json", "/legacy/nix/registry.json")

	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, string(existing), string(merged))
}

func TestMergeNixRegistryAliases_MalformedExisting_ReturnsActionableError(t *testing.T) {
	_, _, err := container.MergeNixRegistryAliases(
		[]byte(`{"version":2,"flakes":[`),
		[]byte(`{"version":2,"flakes":[]}`),
		"/state/nix/registry.json",
		"/legacy/nix/registry.json",
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "/state/nix/registry.json")
	assert.ErrorContains(t, err, "fix JSON syntax or remove the file")
}

func TestMergeNixRegistryAliases_MalformedIncoming_ReturnsActionableError(t *testing.T) {
	_, _, err := container.MergeNixRegistryAliases(
		[]byte(`{"version":2,"flakes":[]}`),
		[]byte(`{"version":2,"flakes":[`),
		"/state/nix/registry.json",
		"/legacy/nix/registry.json",
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "/legacy/nix/registry.json")
	assert.ErrorContains(t, err, "fix JSON syntax or remove the file")
}
