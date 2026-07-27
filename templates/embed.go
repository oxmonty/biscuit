// Package templates embeds the generated-repo template tree. The tree mirrors
// the emitted repository; every file carries a .tmpl suffix so nested go.mod
// and .go sources stay invisible to the Go toolchain.
package templates

import "embed"

//go:embed repo
var FS embed.FS
