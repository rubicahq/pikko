package storage

import "testing"

func TestPurposeLimitsBrandingTwoMB(t *testing.T) {
	const twoMB = int64(2 << 20)
	for _, purpose := range []string{"favicon", "store_icon", "logo"} {
		lim, ok := PurposeLimits[purpose]
		if !ok {
			t.Fatalf("missing purpose %q", purpose)
		}
		if lim.MaxBytes != twoMB {
			t.Fatalf("%s MaxBytes: got %d want %d", purpose, lim.MaxBytes, twoMB)
		}
	}
}

func TestValidateUploadFaviconAcceptsJPEGUnderTwoMB(t *testing.T) {
	// Minimal JPEG SOI + JFIF-ish head for DetectContentType (needs ~512).
	head := make([]byte, 32)
	head[0], head[1] = 0xff, 0xd8
	head[2], head[3] = 0xff, 0xe0
	copy(head[6:], []byte("JFIF"))
	ct, err := ValidateUpload("favicon", 1<<20, "image/jpeg", head)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("got %q", ct)
	}
}

func TestValidateUploadRejectsOverTwoMB(t *testing.T) {
	head := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	_, err := ValidateUpload("store_icon", (2<<20)+1, "image/png", head)
	if err == nil || err.Error() != "file size exceeds limit" {
		t.Fatalf("expected size limit error, got %v", err)
	}
}
