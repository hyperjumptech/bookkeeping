package static

import (
	"embed"
	"fmt"
	"github.com/IDN-Media/awards/static/mime"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

var (
	errFileNotFound = fmt.Errorf("file not found")
)

//go:embed api dashboard
var fs embed.FS

type FileData struct {
	Bytes       []byte
	ContentType string
}

func IsDir(path string) bool {
	for _, s := range GetPathTree("static") {
		if s == "[DIR]"+path {
			return true
		}
	}
	return false
}

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

func GetFile(path string) (*FileData, error) {
	bytes, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mimeType, err := mime.MimeForFileName(path)
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
