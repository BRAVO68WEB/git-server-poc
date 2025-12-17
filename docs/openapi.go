package docs

import "embed"

//go:embed openapi.json
var OpenAPISpec embed.FS
