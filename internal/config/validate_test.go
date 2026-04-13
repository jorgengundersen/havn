package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgengundersen/havn/internal/config"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := config.Default()

	err := config.Validate(cfg)

	assert.NoError(t, err)
}

func TestValidate_EnvironmentReservedKeyRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = map[string]string{
		"SSH_AUTH_SOCK": "/tmp/agent.sock",
	}

	err := config.Validate(cfg)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "environment.SSH_AUTH_SOCK", valErr.Field)
}

func TestValidate_EnvironmentUnsetPassthroughRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = map[string]string{
		"API_TOKEN": "${UNSET_API_TOKEN}",
	}

	err := config.Validate(cfg)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "environment.API_TOKEN", valErr.Field)
}

func TestValidate_CPUsZero(t *testing.T) {
	cfg := config.Default()
	cfg.Resources.CPUs = 0

	err := config.Validate(cfg)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "resources.cpus", valErr.Field)
	assert.Equal(t, "must be greater than 0", valErr.Reason)
}

func TestValidate_InvalidMemoryFormat(t *testing.T) {
	tests := []struct {
		name   string
		memory string
	}{
		{name: "no unit suffix", memory: "8"},
		{name: "invalid unit", memory: "8x"},
		{name: "empty string", memory: ""},
		{name: "letters only", memory: "abc"},
		{name: "negative value", memory: "-4g"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Resources.Memory = tt.memory

			err := config.Validate(cfg)

			var valErr *config.ValidationError
			require.ErrorAs(t, err, &valErr)
			assert.Equal(t, "resources.memory", valErr.Field)
		})
	}
}

func TestValidate_InvalidMemorySwapFormat(t *testing.T) {
	cfg := config.Default()
	cfg.Resources.MemorySwap = "bad"

	err := config.Validate(cfg)

	var valErr *config.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "resources.memory_swap", valErr.Field)
}

func TestValidate_EmptyMemorySwap_Valid(t *testing.T) {
	cfg := config.Default()
	cfg.Resources.MemorySwap = ""

	err := config.Validate(cfg)

	assert.NoError(t, err)
}

func TestValidate_InvalidDoltPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{name: "zero", port: 0},
		{name: "negative", port: -1},
		{name: "too high", port: 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Dolt.Port = tt.port

			err := config.Validate(cfg)

			var valErr *config.ValidationError
			require.ErrorAs(t, err, &valErr)
			assert.Equal(t, "dolt.port", valErr.Field)
		})
	}
}

func TestValidate_InvalidPorts(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "missing container port", port: "8080"},
		{name: "missing host port", port: ":8080"},
		{name: "not a number host", port: "abc:8080"},
		{name: "not a number container", port: "8080:abc"},
		{name: "empty string", port: ""},
		{name: "triple colon", port: "80:80:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Ports = []string{tt.port}

			err := config.Validate(cfg)

			var valErr *config.ValidationError
			require.ErrorAs(t, err, &valErr)
			assert.Equal(t, "ports", valErr.Field)
		})
	}
}

func TestValidate_ValidPorts(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "simple mapping", port: "8080:8080"},
		{name: "different ports", port: "3000:80"},
		{name: "with tcp proto", port: "8080:8080/tcp"},
		{name: "with udp proto", port: "8080:8080/udp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Ports = []string{tt.port}

			err := config.Validate(cfg)

			assert.NoError(t, err)
		})
	}
}

func TestValidate_InvalidMountsConfig(t *testing.T) {
	tests := []struct {
		name  string
		mount string
	}{
		{name: "missing mode", mount: ".gitconfig"},
		{name: "invalid mode", mount: ".gitconfig:rw-invalid"},
		{name: "empty string", mount: ""},
		{name: "wrong mode value", mount: ".gitconfig:exec"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Mounts.Config = []string{tt.mount}

			err := config.Validate(cfg)

			var valErr *config.ValidationError
			require.ErrorAs(t, err, &valErr)
			assert.Equal(t, "mounts.config", valErr.Field)
		})
	}
}

func TestValidate_ValidMountsConfig(t *testing.T) {
	tests := []struct {
		name  string
		mount string
	}{
		{name: "readonly", mount: ".gitconfig:ro"},
		{name: "readwrite", mount: ".config/nvim/:rw"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Mounts.Config = []string{tt.mount}

			err := config.Validate(cfg)

			assert.NoError(t, err)
		})
	}
}

func TestValidate_ValidMemoryFormats(t *testing.T) {
	tests := []struct {
		name   string
		memory string
	}{
		{name: "gigabytes", memory: "8g"},
		{name: "megabytes", memory: "512m"},
		{name: "kilobytes", memory: "1024k"},
		{name: "bytes", memory: "1073741824b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Resources.Memory = tt.memory

			err := config.Validate(cfg)

			assert.NoError(t, err)
		})
	}
}
