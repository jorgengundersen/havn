package dolt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jorgengundersen/havn/internal/config"
	"github.com/jorgengundersen/havn/internal/dolt"
)

func TestGenerateConfig_DefaultPort(t *testing.T) {
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Port: 3308,
		},
	}

	got := dolt.GenerateConfig(cfg)

	want := `log_level: info

listener:
  host: 0.0.0.0
  port: 3308
  read_timeout_millis: 300000
  write_timeout_millis: 300000

data_dir: /var/lib/dolt

behavior:
  autocommit: true
`

	assert.Equal(t, want, string(got))
}

func TestGenerateConfig_CustomPort(t *testing.T) {
	cfg := config.Config{
		Dolt: config.DoltConfig{
			Port: 3309,
		},
	}

	got := dolt.GenerateConfig(cfg)

	assert.Contains(t, string(got), "port: 3309")
}
