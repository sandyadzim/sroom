package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func readJSON(c *gin.Context, target any) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json")
		return false
	}

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(target)
	if err == nil {
		// The body must contain exactly one JSON object.
		err = decoder.Decode(new(json.RawMessage))
		if err == io.EOF {
			return true
		}
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return false
	}
	response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "send a valid JSON object")
	return false
}
