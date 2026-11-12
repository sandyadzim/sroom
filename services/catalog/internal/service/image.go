package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidImage     = errors.New("image_object_key must refer to an uploaded image")
	ErrImageUnavailable = errors.New("image storage unavailable")
)

type ImageChecker interface {
	ImageExists(ctx context.Context, key string) (bool, error)
}

func checkImage(ctx context.Context, images ImageChecker, key *string) error {
	if key == nil {
		return nil
	}
	filename, validPrefix := strings.CutPrefix(*key, "catalog/images/")
	id, extension, hasExtension := strings.Cut(filename, ".")
	if !validPrefix || !hasExtension || len(id) != 26 ||
		strings.Trim(id, "abcdefghijklmnopqrstuvwxyz234567") != "" {
		return ErrInvalidImage
	}
	if extension != "jpeg" && extension != "png" && extension != "webp" {
		return ErrInvalidImage
	}
	if images == nil {
		return ErrImageUnavailable
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	exists, err := images.ImageExists(ctx, *key)
	if err != nil {
		return ErrImageUnavailable
	}
	if !exists {
		return ErrInvalidImage
	}
	return nil
}
