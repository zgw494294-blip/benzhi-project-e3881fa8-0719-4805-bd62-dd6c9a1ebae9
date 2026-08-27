package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

type apiError struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return caption.Invalid("body", "请求体不能为空")
		}
		return caption.Invalid("body", "JSON 格式或字段不合法："+err.Error())
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return caption.Invalid("body", "只能提交一个 JSON 对象")
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, workflow.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, workflow.ErrConflict):
		status, code = http.StatusConflict, "already_exists"
	case errors.Is(err, caption.ErrVersionConflict):
		status, code = http.StatusConflict, "version_conflict"
	case errors.Is(err, workflow.ErrIdempotencyKey):
		status, code = http.StatusConflict, "idempotency_conflict"
	case errors.Is(err, caption.ErrInvalidState):
		status, code = http.StatusConflict, "invalid_state"
	case errors.Is(err, caption.ErrActorNotIndependent):
		status, code = http.StatusUnprocessableEntity, "reviewer_not_independent"
	case errors.Is(err, caption.ErrBlockingFindings):
		status, code = http.StatusUnprocessableEntity, "blocking_findings"
	case errors.Is(err, caption.ErrInvalidInput), errors.Is(err, caption.ErrDuplicateCue):
		status, code = http.StatusBadRequest, "invalid_input"
	}
	body := apiErrorBody{Code: code, Message: err.Error()}
	var validation *caption.ValidationError
	if errors.As(err, &validation) {
		body.Field = validation.Field
		body.Message = validation.Message
	}
	writeJSON(w, status, apiError{Error: body})
}

func packageID(r *http.Request) (string, error) {
	id := strings.TrimSpace(r.PathValue("packageId"))
	if id == "" {
		return "", caption.Invalid("packageId", "路径参数不能为空")
	}
	return id, nil
}
