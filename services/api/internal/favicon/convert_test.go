package favicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestConvertPNGToICO(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 61, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, ct, err := Convert(buf.Bytes(), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/x-icon" {
		t.Fatalf("content-type: got %q", ct)
	}
	if !looksLikeICO(out) {
		t.Fatal("expected ICO header")
	}
	// 3 directory entries for 16/32/48
	count := int(out[4]) | int(out[5])<<8
	if count != 3 {
		t.Fatalf("entry count: got %d want 3", count)
	}
}

func TestConvertSVGPassthrough(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"><rect width="32" height="32" fill="#FF3D00"/></svg>`)
	out, ct, err := Convert(svg, "image/svg+xml")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/svg+xml" {
		t.Fatalf("content-type: got %q", ct)
	}
	if !bytes.Equal(out, svg) {
		t.Fatal("SVG should pass through unchanged")
	}
}

func TestConvertRejectsGarbage(t *testing.T) {
	_, _, err := Convert([]byte("not-an-image"), "image/png")
	if err == nil {
		t.Fatal("expected decode error")
	}
}
