package static

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed dist
var reactApp embed.FS

var ReactFS fs.FS

func init() {
	reactFS, err := fs.Sub(reactApp, "dist")
	if err != nil {
		panic(fmt.Sprintf("Initialization error: Failed to access embedded React files in 'dist' directory: %v", err))
	}
	ReactFS = reactFS
}
