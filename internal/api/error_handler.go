package api

import (
	"encoding/json"
	"net/http"

	model "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func DefaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	status := http.StatusBadRequest
	code := "bad_request"
	message := err.Error()

	switch err.(type) {
	case *RequiredParamError:
		code = "required_param"
	case *RequiredHeaderError:
		code = "required_header"
	case *UnmarshalingParamError:
		code = "invalid_json"
	case *InvalidParamFormatError:
		code = "invalid_format"
	case *TooManyValuesForParamError:
		code = "too_many_values"
	case *UnescapedCookieParamError:
		code = "invalid_cookie"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.Error{Code: &status, Error: &code, Message: &message})
}
