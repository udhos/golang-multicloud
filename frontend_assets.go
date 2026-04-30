// Package multicloud is a helper package that embeds the static frontend assets for the shopping cart backend.
package multicloud

import "embed"

// FrontendFiles contains the static frontend assets served by the backend.
//
//go:embed frontend
var FrontendFiles embed.FS
