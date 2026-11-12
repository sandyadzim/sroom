package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp"
)

const maxImageBytes = 5 * 1024 * 1024

type ImageStore interface {
	Upload(ctx context.Context, key, contentType string, data []byte) error
}

func UploadImage(store ImageStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "image storage unavailable")
			return
		}

		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			response.Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be multipart/form-data")
			return
		}
		reader, err := c.Request.MultipartReader()
		if err != nil {
			writeUploadReadError(c, err)
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			writeUploadReadError(c, err)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "send one image in the file field")
			return
		}

		data, err := io.ReadAll(io.LimitReader(part, maxImageBytes+1))
		if err != nil {
			writeUploadReadError(c, err)
			return
		}
		if len(data) > maxImageBytes {
			response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "image must be at most 5 MiB")
			return
		}
		if _, err := reader.NextPart(); err != io.EOF {
			writeUploadReadError(c, err)
			return
		}
		if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
			writeUploadReadError(c, err)
			return
		}

		_, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
			response.Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "file must be a JPEG, PNG, or WebP image")
			return
		}

		key := "catalog/images/" + strings.ToLower(rand.Text()) + "." + format
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		if err := store.Upload(ctx, key, "image/"+format, data); err != nil {
			slog.Error("image upload failed", "service", "catalog", "operation", "image.upload", "request_id", c.Writer.Header().Get("X-Request-ID"))
			response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "image storage unavailable")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": gin.H{"object_key": key}})
	}
}

func writeUploadReadError(c *gin.Context, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return
	}

	response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "send one image in the file field")
}
