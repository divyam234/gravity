package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	apperrors "gravity/internal/errors"
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

func sendAppError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	var msg string
	var code int
	var errCode string

	var appErr *apperrors.AppError
	var notFound *apperrors.NotFoundError

	if errors.As(err, &appErr) {
		msg = appErr.Message
		errCode = string(appErr.Code)
		// Map ErrorCode to HTTP Status
		switch appErr.Code {
		case apperrors.CodeNotFound:
			code = http.StatusNotFound
		case apperrors.CodeValidationFailed:
			code = http.StatusBadRequest
		case apperrors.CodeInvalidTransition, apperrors.CodeInvalidOperation:
			code = http.StatusConflict
		default:
			code = http.StatusInternalServerError
		}
	} else if errors.As(err, &notFound) {
		msg = notFound.Error()
		code = http.StatusNotFound
		errCode = string(apperrors.CodeNotFound)
	} else {
		msg = err.Error()
		code = http.StatusInternalServerError
		errCode = string(apperrors.CodeInternalError)
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     msg,
		Code:      code,
		ErrorCode: errCode,
	})
}

func sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

type Validatable interface {
	Validate() error
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

	// Check for Validatable interface
	if v, ok := dst.(Validatable); ok {
		if err := v.Validate(); err != nil {
			sendAppError(w, err)
			return false
		}
	}

	return true
}
