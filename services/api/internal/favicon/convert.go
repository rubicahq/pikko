// Package favicon converts uploaded images into a multi-resolution ICO.
package favicon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register image/webp decoder
)

// Standard favicon sizes written into the ICO.
var sizes = []int{16, 32, 48}

// Convert turns raster image bytes into a multi-size favicon.ico.
// SVG and already-valid ICO inputs are returned unchanged.
func Convert(data []byte, contentType string) (out []byte, outCT string, err error) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if ct == "image/svg+xml" || looksLikeSVG(data) {
		return data, "image/svg+xml", nil
	}
	if ct == "image/x-icon" || ct == "image/vnd.microsoft.icon" || looksLikeICO(data) {
		// Re-encode when we can decode an embedded bitmap; otherwise keep as-is.
		if img, ok := decodeICOFirst(data); ok {
			ico, err := encodeICO(img)
			if err != nil {
				return nil, "", err
			}
			return ico, "image/x-icon", nil
		}
		return data, "image/x-icon", nil
	}

	img, err := decodeImage(data, ct)
	if err != nil {
		return nil, "", fmt.Errorf("unable to decode image for favicon: %w", err)
	}
	ico, err := encodeICO(img)
	if err != nil {
		return nil, "", err
	}
	return ico, "image/x-icon", nil
}

func decodeImage(data []byte, ct string) (image.Image, error) {
	r := bytes.NewReader(data)
	switch ct {
	case "image/png":
		return png.Decode(r)
	case "image/jpeg", "image/jpg":
		return jpeg.Decode(r)
	case "image/gif":
		return gif.Decode(r)
	case "image/webp":
		img, _, err := image.Decode(r)
		return img, err
	default:
		img, _, err := image.Decode(r)
		if err != nil {
			return nil, fmt.Errorf("unsupported image type %q", ct)
		}
		return img, nil
	}
}

func encodeICO(src image.Image) ([]byte, error) {
	type entry struct {
		size int
		png  []byte
	}
	entries := make([]entry, 0, len(sizes))
	for _, s := range sizes {
		resized := resizeSquare(src, s)
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err != nil {
			return nil, err
		}
		entries = append(entries, entry{size: s, png: buf.Bytes()})
	}

	const (
		dirHeader = 6
		dirEntry  = 16
	)
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type = icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(entries)))

	offset := dirHeader + dirEntry*len(entries)
	for _, e := range entries {
		w := byte(e.size)
		if e.size >= 256 {
			w = 0
		}
		out.WriteByte(w)
		out.WriteByte(w) // height
		out.WriteByte(0) // color count
		out.WriteByte(0) // reserved
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))  // planes
		_ = binary.Write(&out, binary.LittleEndian, uint16(32)) // bit count
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(e.png)))
		_ = binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(e.png)
	}
	for _, e := range entries {
		out.Write(e.png)
	}
	return out.Bytes(), nil
}

func resizeSquare(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	// Cover-fit into square (center crop).
	side := sw
	if sh < sw {
		side = sh
	}
	x0 := b.Min.X + (sw-side)/2
	y0 := b.Min.Y + (sh-side)/2
	srcSquare := image.NewNRGBA(image.Rect(0, 0, side, side))
	xdraw.Draw(srcSquare, srcSquare.Bounds(), src, image.Pt(x0, y0), xdraw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), srcSquare, srcSquare.Bounds(), xdraw.Over, nil)
	return dst
}

func looksLikeSVG(data []byte) bool {
	l := bytes.ToLower(bytes.TrimSpace(data))
	return bytes.Contains(l, []byte("<svg")) || bytes.HasPrefix(l, []byte("<?xml"))
}

func looksLikeICO(data []byte) bool {
	if len(data) < 6 {
		return false
	}
	// reserved=0, type=1 (icon)
	return data[0] == 0 && data[1] == 0 && data[2] == 1 && data[3] == 0
}

// decodeICOFirst tries to extract the first PNG-compressed image from an ICO.
func decodeICOFirst(data []byte) (image.Image, bool) {
	if !looksLikeICO(data) || len(data) < 22 {
		return nil, false
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count < 1 {
		return nil, false
	}
	offset := int(binary.LittleEndian.Uint32(data[18:22]))
	size := int(binary.LittleEndian.Uint32(data[14:18]))
	if offset < 0 || size <= 0 || offset+size > len(data) {
		return nil, false
	}
	chunk := data[offset : offset+size]
	if bytes.HasPrefix(chunk, []byte{0x89, 'P', 'N', 'G'}) {
		img, err := png.Decode(bytes.NewReader(chunk))
		if err != nil {
			return nil, false
		}
		return img, true
	}
	return nil, false
}
