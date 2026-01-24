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

func decodeAndValidate(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	// We read the body to log it on error
	bodyBytes, _ := io.ReadAll(r.Body)
	// Restore body for decoder
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
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

		http.Error(w, errMsg, http.StatusBadRequest)
		return false
	}

	return true
}
