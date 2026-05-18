package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type VideoService struct {
	repo *repository.VideoRepository

	// Cloudinary credentials for signed direct uploads.
	cloudinaryAPIKey    string
	cloudinaryAPISecret string
	cloudinaryCloud     string
	uploadPreset        string
	uploadFolder        string
}

type CloudinaryConfig struct {
	APIKey       string
	APISecret    string
	CloudName    string
	UploadPreset string
	Folder       string
}

func NewVideoService(repo *repository.VideoRepository) *VideoService {
	return &VideoService{repo: repo}
}

// WithCloudinary sets the Cloudinary credentials used by UploadSignature.
func (s *VideoService) WithCloudinary(c CloudinaryConfig) *VideoService {
	s.cloudinaryAPIKey = c.APIKey
	s.cloudinaryAPISecret = c.APISecret
	s.cloudinaryCloud = c.CloudName
	s.uploadPreset = c.UploadPreset
	s.uploadFolder = c.Folder
	return s
}

func (s *VideoService) List(ctx context.Context, target string) ([]domain.Video, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target != "" && target != "kiosk" && target != "display" && target != "both" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.List(ctx, target)
}

func (s *VideoService) Get(ctx context.Context, id string) (*domain.Video, error) {
	return s.repo.Get(ctx, id)
}

type VideoInput struct {
	Title           string
	URL             string
	TargetScreen    string
	DurationSeconds *int
	DisplayOrder    *int
	IsActive        *bool
	UploadedBy      *string
}

func validateTarget(t string) (string, error) {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "kiosk", "display", "both":
		return t, nil
	default:
		return "", domain.ErrInvalidInput
	}
}

func (s *VideoService) Create(ctx context.Context, in VideoInput) (*domain.Video, error) {
	title := strings.TrimSpace(in.Title)
	url := strings.TrimSpace(in.URL)
	if title == "" || url == "" {
		return nil, domain.ErrInvalidInput
	}
	target, err := validateTarget(in.TargetScreen)
	if err != nil {
		return nil, err
	}
	order := 0
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	return s.repo.Create(ctx, repository.VideoFields{
		Title: title, URL: url, TargetScreen: target,
		DurationSeconds: in.DurationSeconds, DisplayOrder: order,
		IsActive: active, UploadedBy: in.UploadedBy,
	})
}

type VideoPatchInput struct {
	Title           *string
	URL             *string
	TargetScreen    *string
	DurationSeconds *int
	ClearDuration   bool
	DisplayOrder    *int
	IsActive        *bool
}

func (s *VideoService) Update(ctx context.Context, id string, in VideoPatchInput) (*domain.Video, error) {
	patch := repository.VideoPatch{
		DurationSeconds: in.DurationSeconds,
		ClearDuration:   in.ClearDuration,
		DisplayOrder:    in.DisplayOrder,
		IsActive:        in.IsActive,
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.Title = &t
	}
	if in.URL != nil {
		u := strings.TrimSpace(*in.URL)
		if u == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.URL = &u
	}
	if in.TargetScreen != nil {
		t, err := validateTarget(*in.TargetScreen)
		if err != nil {
			return nil, err
		}
		patch.TargetScreen = &t
	}
	return s.repo.Update(ctx, id, patch)
}

func (s *VideoService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// UploadSignature produces parameters the browser uses to POST directly to
// the Cloudinary upload endpoint without exposing the API secret.
type UploadSignature struct {
	Signature    string `json:"signature"`
	Timestamp    int64  `json:"timestamp"`
	APIKey       string `json:"api_key"`
	CloudName    string `json:"cloud_name"`
	Folder       string `json:"folder,omitempty"`
	UploadPreset string `json:"upload_preset,omitempty"`
}

func (s *VideoService) UploadSignature() (*UploadSignature, error) {
	if s.cloudinaryAPISecret == "" || s.cloudinaryAPIKey == "" || s.cloudinaryCloud == "" {
		return nil, domain.ErrInvalidInput
	}
	ts := time.Now().Unix()

	// String-to-sign: alphabetically sorted key=value pairs joined by &,
	// then append the API secret.
	params := map[string]string{"timestamp": fmt.Sprintf("%d", ts)}
	if s.uploadFolder != "" {
		params["folder"] = s.uploadFolder
	}
	if s.uploadPreset != "" {
		params["upload_preset"] = s.uploadPreset
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	toSign := strings.Join(parts, "&") + s.cloudinaryAPISecret

	h := sha1.Sum([]byte(toSign))
	sig := hex.EncodeToString(h[:])

	return &UploadSignature{
		Signature:    sig,
		Timestamp:    ts,
		APIKey:       s.cloudinaryAPIKey,
		CloudName:    s.cloudinaryCloud,
		Folder:       s.uploadFolder,
		UploadPreset: s.uploadPreset,
	}, nil
}
