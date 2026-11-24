package storage

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

type UploadLimits struct {
	MaxBytes     int64
	AllowedMIME  []string
}

var PurposeLimits = map[string]UploadLimits{
	"logo":       {MaxBytes: 2 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp", "image/svg+xml"}},
	"favicon":    {MaxBytes: 512 << 10, AllowedMIME: []string{"image/png", "image/x-icon", "image/vnd.microsoft.icon", "image/svg+xml", "image/webp"}},
	"store_icon": {MaxBytes: 1 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp"}},
	"product":    {MaxBytes: 5 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp"}},
	"banner":     {MaxBytes: 5 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp"}},
	"collection": {MaxBytes: 5 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp"}},
	"other":      {MaxBytes: 2 << 20, AllowedMIME: []string{"image/png", "image/jpeg", "image/webp"}},
}

func ValidateUpload(purpose string, size int64, contentType string, head []byte) (string, error) {
	limits, ok := PurposeLimits[purpose]
	if !ok {
		return "", fmt.Errorf("unsupported upload purpose")
	}
	if size <= 0 || size > limits.MaxBytes {
		return "", fmt.Errorf("file size exceeds limit")
	}
	detected := http.DetectContentType(head)
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" || ct == "application/octet-stream" {
		ct = detected
	}
	// Normalize jpeg
	if ct == "image/jpg" {
		ct = "image/jpeg"
	}
	if strings.HasPrefix(detected, "image/") && !mimeAllowed(detected, limits.AllowedMIME) && detected != "text/xml; charset=utf-8" {
		// Prefer sniffed type when clearly an image mismatch
		if !mimeAllowed(ct, limits.AllowedMIME) {
			return "", fmt.Errorf("unsupported file type")
		}
	}
	if !mimeAllowed(ct, limits.AllowedMIME) {
		// SVG may sniff as text/xml
		if purpose == "logo" || purpose == "favicon" {
			if looksLikeSVG(head) && mimeAllowed("image/svg+xml", limits.AllowedMIME) {
				return "image/svg+xml", nil
			}
		}
		return "", fmt.Errorf("unsupported file type")
	}
	if (purpose == "logo" || purpose == "favicon") && ct == "image/svg+xml" && !looksLikeSVG(head) && !bytes.Contains(head, []byte("<svg")) {
		// Allow if content-type claims svg and size small; block obvious HTML
		if bytes.Contains(bytes.ToLower(head), []byte("<script")) {
			return "", fmt.Errorf("unsupported file type")
		}
	}
	return ct, nil
}

func mimeAllowed(ct string, allowed []string) bool {
	ct = strings.ToLower(strings.Split(ct, ";")[0])
	for _, a := range allowed {
		if ct == a {
			return true
		}
	}
	return false
}

func looksLikeSVG(head []byte) bool {
	l := bytes.ToLower(head)
	return bytes.Contains(l, []byte("<svg")) || bytes.Contains(l, []byte("<?xml"))
}
