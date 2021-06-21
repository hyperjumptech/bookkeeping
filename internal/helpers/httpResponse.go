// Package helpers are code saving utility and helper functions
package helpers

import (
	"context"
	"encoding/json"
	"net/http"
)

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
