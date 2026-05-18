package service

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

func TestVideoService_UploadSignature_RequiresCloudinaryCreds(t *testing.T) {
	svc := NewVideoService(nil)
	_, err := svc.UploadSignature()
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput when creds missing, got %v", err)
	}
}

func TestVideoService_UploadSignature_ProducesValidSHA1(t *testing.T) {
	svc := NewVideoService(nil).WithCloudinary(CloudinaryConfig{
		APIKey:       "api-key-123",
		APISecret:    "supersecret",
		CloudName:    "my-cloud",
		UploadPreset: "videos-preset",
		Folder:       "sistem-antrian/videos",
	})

	got, err := svc.UploadSignature()
	if err != nil {
		t.Fatalf("UploadSignature: %v", err)
	}

	if got.APIKey != "api-key-123" {
		t.Errorf("api_key = %q", got.APIKey)
	}
	if got.CloudName != "my-cloud" {
		t.Errorf("cloud_name = %q", got.CloudName)
	}
	if got.UploadPreset != "videos-preset" {
		t.Errorf("upload_preset = %q", got.UploadPreset)
	}
	if got.Folder != "sistem-antrian/videos" {
		t.Errorf("folder = %q", got.Folder)
	}
	if got.Timestamp == 0 {
		t.Error("timestamp must be set")
	}

	// Reproduce the signature: alphabetically-sorted key=value pairs joined
	// by '&', then append the secret, then SHA-1 hex.
	params := map[string]string{
		"timestamp":     fmt.Sprintf("%d", got.Timestamp),
		"folder":        "sistem-antrian/videos",
		"upload_preset": "videos-preset",
	}
	keys := []string{}
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	want := sha1.Sum([]byte(strings.Join(parts, "&") + "supersecret"))
	if got.Signature != hex.EncodeToString(want[:]) {
		t.Errorf("signature mismatch: got %s", got.Signature)
	}
}

func TestVideoService_UploadSignature_OmitsOptionalParams(t *testing.T) {
	svc := NewVideoService(nil).WithCloudinary(CloudinaryConfig{
		APIKey: "k", APISecret: "s", CloudName: "c",
		// no preset, no folder
	})
	got, err := svc.UploadSignature()
	if err != nil {
		t.Fatalf("UploadSignature: %v", err)
	}
	if got.UploadPreset != "" || got.Folder != "" {
		t.Errorf("expected blank optional params, got preset=%q folder=%q", got.UploadPreset, got.Folder)
	}

	// Only "timestamp" should contribute to the signed string.
	want := sha1.Sum([]byte(fmt.Sprintf("timestamp=%d", got.Timestamp) + "s"))
	if got.Signature != hex.EncodeToString(want[:]) {
		t.Errorf("signature with only timestamp mismatch: got %s", got.Signature)
	}
}
