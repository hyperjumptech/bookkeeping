// Package helpers are code saving utility and helper functions
package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ParsePathParams parse request path param according to path template and extract its values.
func ParsePathParams(template, path string) (map[string]string, error) {
	var pth string
	if strings.Contains(path, "?") {
		pth = path[:strings.Index(path, "?")]
	} else {
		pth = path
	}
	templatePaths := strings.Split(template, "/")
	pathPaths := strings.Split(pth, "/")
	if len(templatePaths) != len(pathPaths) {
		return nil, fmt.Errorf("pathElement length not equals to templateElement length")
	}
	ret := make(map[string]string)
	for idx, templateElement := range templatePaths {
		pathElement := pathPaths[idx]
		if len(templateElement) > 0 && len(pathElement) > 0 {
			if templateElement[:1] == "{" && templateElement[len(templateElement)-1:] == "}" {
				tKey := templateElement[1 : len(templateElement)-1]
				ret[tKey] = pathElement
			} else if templateElement != pathElement {
				return nil, fmt.Errorf("template %s not compatible with path %s", template, path)
			}
		}
	}
	return ret, nil
}

// ResponseJSON define the structure of all response
type ResponseJSON struct {
	Message   string      `json:"message"`
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorCode int         `json:"error_code,omitempty"`
}

// HTTPResponseBuilder builds the response headers and payloads
func HTTPResponseBuilder(ctx context.Context, w http.ResponseWriter, r *http.Request, httpStatus int, message string, data interface{}, errorCode int) {

	resp := ResponseJSON{
		Data:    data,
		Message: message,
	}

	if httpStatus >= 200 && httpStatus < 300 {
		resp.Status = "SUCCESS"
	} else {
		resp.Status = "FAIL"
		resp.ErrorCode = errorCode
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}
