package handlers

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	static "github.com/k0rdent/kof/kof-operator/webapp/collector"
)

func ReactAppHandler(res *server.Response, req *http.Request) {
	if serveStaticFile(res, req, static.ReactFS) {
		return
	}
	NotFoundHandler(res, req)
}

func serveStaticFile(res *server.Response, req *http.Request, staticFS fs.FS) bool {
	filePath := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
	if filePath == "" {
		filePath = "index.html"
	}

	file, err := staticFS.Open(filePath)
	if err != nil {
		return false
	}
	defer func() {
		err := file.Close()
		if err != nil {
			res.Logger.Error(err, "Cannot close file", "path", filePath)
		}
	}()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		return false
	}

	contentType := getContentType(filePath)
	res.SetContentType(contentType)

	http.ServeContent(res.Writer, req, filePath, stat.ModTime(), file.(io.ReadSeeker))
	return true
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "text/plain"
	}
}
