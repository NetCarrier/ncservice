package codegen

import "embed"

// Include the template files in Go package

//go:embed *.tpl
var internal embed.FS
