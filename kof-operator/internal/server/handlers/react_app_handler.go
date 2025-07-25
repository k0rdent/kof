package handlers

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	static "github.com/k0rdent/kof/kof-operator/webapp/collector"
)

const ReactAppMainFile = "index.html"

func ReactAppHandler(res *server.Response, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/api/") {
		NotFoundHandler(res, req)
		return
	}
	if err := serveStaticFile(res, req, static.ReactFS); err != nil {
		res.Logger.Error(err, "Failed to serve static file", "path", req.URL.Path)
	}
}

func serveStaticFile(res *server.Response, req *http.Request, staticFS fs.FS) error {
	filePath := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
	if filePath == "" {
		filePath = ReactAppMainFile
	}

	file, err := staticFS.Open(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			filePath = ReactAppMainFile
			file, err = staticFS.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open %s file: %v", ReactAppMainFile, err)
			}
		} else {
			return fmt.Errorf("failed to open %s file: %v", filePath, err)
		}
	}

	defer func() {
		err := file.Close()
		if err != nil {
			res.Logger.Error(err, "Cannot close file", "path", filePath)
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("requested path is a directory: %s", filePath)
	}

	res.SetContentType(getContentType(filePath))
	http.ServeContent(res.Writer, req, filePath, stat.ModTime(), file.(io.ReadSeeker))
	return nil
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
