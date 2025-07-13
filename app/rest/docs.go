package rest

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed docs/index.html
var BuildFs embed.FS

// Get the subtree of the embedded files with `build` directory as a root.
func BuildHTTPFS() http.FileSystem {
	build, err := fs.Sub(BuildFs, "docs")
	if err != nil {
		log.Fatal(err)
	}
	return http.FS(build)
}
