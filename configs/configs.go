package configs

import "embed"

//go:embed config.yaml
var EmbeddedConfigs embed.FS
