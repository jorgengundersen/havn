package dolt

import (
	"fmt"

	"github.com/jorgengundersen/havn/internal/config"
)

// GenerateConfig produces the Dolt server config.yaml content
// matching specs/shared-dolt-server.md §Server configuration.
func GenerateConfig(cfg config.Config) []byte {
	yaml := fmt.Sprintf(`log_level: info

listener:
  host: 0.0.0.0
  port: %d
  read_timeout_millis: 300000
  write_timeout_millis: 300000

data_dir: /var/lib/dolt

behavior:
  autocommit: true
`, cfg.Dolt.Port)

	return []byte(yaml)
}
