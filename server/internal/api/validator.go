package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gravity/internal/logger"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var validate = validator.New()

func sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message, Code: code})
}

func sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func decodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) bool {
	// We read the body to log it on error
	bodyBytes, _ := io.ReadAll(r.Body)
	// Restore body for decoder
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		sendError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return false
	}

	if err := validate.Struct(dst); err != nil {
		// ...
		errMsg := "validation failed: "
		for _, err := range err.(validator.ValidationErrors) {
			errMsg += fmt.Sprintf("[%s: %s] ", err.Field(), err.Tag())
		}

		logger.L.Warn("API validation failed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("error", errMsg),
			zap.String("body", string(bodyBytes)),
		)

		sendError(w, errMsg, http.StatusBadRequest)
		return false
	}

	return true
}
