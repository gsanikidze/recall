//go:build !ui

package uiassets

import "embed"

// FS is empty when the UI has not been built. Use `make build` to embed assets.
var FS embed.FS
