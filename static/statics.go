package static

import (
	"embed"
	"net/http"
	"os"
	"strings"

	"github.com/hyperjumptech/bookkeeping/static/mime"
	"github.com/sirupsen/logrus"
)

//go:embed api dashboard
var fs embed.FS

// FileData is a file data structure
type FileData struct {
	Bytes       []byte
	ContentType string
}

// IsDir checks if path is a dir
func IsDir(path string) bool {
	for _, s := range GetPathTree("static") {
		if s == "[DIR]"+path {
			return true
		}
	}
	return false
}

// GetPathTree builds ta tree from the path
func GetPathTree(path string) []string {
	logrus.Infof("Into %s", path)
	var entries []os.DirEntry
	var err error
	if strings.HasPrefix(path, "./") {
		entries, err = fs.ReadDir(path[2:])
	} else {
		entries, err = fs.ReadDir(path)
	}
	ret := make([]string, 0)
	if err != nil {
		return ret
	}
	logrus.Infof("Path %s %d etries", path, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ret = append(ret, "[DIR]"+path+"/"+e.Name())
			ret = append(ret, GetPathTree(path+"/"+e.Name())...)
		} else {
			ret = append(ret, path+"/"+e.Name())
		}
	}
	return ret
}

// GetFile returns the file from a given path
func GetFile(path string) (*FileData, error) {
	bytes, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mimeType, err := mime.ForFileName(path)
	if err != nil {
		return &FileData{
			Bytes:       bytes,
			ContentType: http.DetectContentType(bytes),
		}, nil
	}
	return &FileData{
		Bytes:       bytes,
		ContentType: mimeType,
	}, nil
}
