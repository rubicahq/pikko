package ai

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"strings"
)

const (
	// visionLongEdgeCap keeps full-page vision requests within typical provider limits.
	visionLongEdgeCap = 2048
	// visionTileLongEdge triggers overlapping tiles for dense poster pages.
	visionTileMinPixels = 1800 * 2400
	visionJPEGQuality   = 85
	visionTileOverlap   = 0.08 // 8% overlap between tiles
)

type visionTile struct {
	PageImage
	// Origin of this tile in full-page percent space (0-100).
	OriginXPct float64
	OriginYPct float64
	WidthPct   float64
	HeightPct  float64
}

// prepareVisionTiles decodes a page image into one or more JPEG tiles for the
// vision model. Large/dense pages are split so product grids stay readable.
func prepareVisionTiles(page PageImage) ([]visionTile, error) {
	raw, err := decodePageBytes(page)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode page image: %w", err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty page image")
	}

	// Single tile when the page is small enough for one shot.
	if w*h < visionTileMinPixels {
		scaled := scaleImage(img, visionLongEdgeCap)
		b64, sw, sh, err := encodeJPEGBase64(scaled)
		if err != nil {
			return nil, err
		}
		return []visionTile{{
			PageImage: PageImage{
				PageNumber: page.PageNumber,
				MIMEType:   "image/jpeg",
				Base64Data: b64,
				Width:      sw,
				Height:     sh,
			},
			OriginXPct: 0, OriginYPct: 0, WidthPct: 100, HeightPct: 100,
		}}, nil
	}

	// Dense/large pages (typical A1 jastip posters): 2×3 overlapping tiles.
	cols, rows := 2, 3
	tiles := make([]visionTile, 0, cols*rows)
	overlapX := visionTileOverlap
	overlapY := visionTileOverlap
	tileW := (1.0 + overlapX*(float64(cols)-1)) / float64(cols)
	tileH := (1.0 + overlapY*(float64(rows)-1)) / float64(rows)
	stepX := (1.0 - tileW) / float64(cols-1)
	stepY := (1.0 - tileH) / float64(rows-1)
	if cols == 1 {
		stepX = 0
		tileW = 1
	}
	if rows == 1 {
		stepY = 0
		tileH = 1
	}

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x0 := float64(col) * stepX
			y0 := float64(row) * stepY
			x1 := x0 + tileW
			y1 := y0 + tileH
			if x1 > 1 {
				x1 = 1
				x0 = x1 - tileW
				if x0 < 0 {
					x0 = 0
				}
			}
			if y1 > 1 {
				y1 = 1
				y0 = y1 - tileH
				if y0 < 0 {
					y0 = 0
				}
			}
			px0 := bounds.Min.X + int(x0*float64(w))
			py0 := bounds.Min.Y + int(y0*float64(h))
			px1 := bounds.Min.X + int(x1*float64(w))
			py1 := bounds.Min.Y + int(y1*float64(h))
			if px1 <= px0 {
				px1 = px0 + 1
			}
			if py1 <= py0 {
				py1 = py0 + 1
			}
			cropped := cropImage(img, image.Rect(px0, py0, px1, py1))
			scaled := scaleImage(cropped, visionLongEdgeCap)
			b64, sw, sh, err := encodeJPEGBase64(scaled)
			if err != nil {
				return nil, err
			}
			tiles = append(tiles, visionTile{
				PageImage: PageImage{
					PageNumber: page.PageNumber,
					MIMEType:   "image/jpeg",
					Base64Data: b64,
					Width:      sw,
					Height:     sh,
				},
				OriginXPct: x0 * 100,
				OriginYPct: y0 * 100,
				WidthPct:   (x1 - x0) * 100,
				HeightPct:  (y1 - y0) * 100,
			})
		}
	}
	return tiles, nil
}

func decodePageBytes(page PageImage) ([]byte, error) {
	if page.Base64Data != "" {
		raw, err := base64.StdEncoding.DecodeString(page.Base64Data)
		if err != nil {
			return nil, fmt.Errorf("decode page base64: %w", err)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("page image has no data")
}

func encodeJPEGBase64(img image.Image) (string, int, int, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: visionJPEGQuality}); err != nil {
		return "", 0, 0, err
	}
	b := img.Bounds()
	return base64.StdEncoding.EncodeToString(buf.Bytes()), b.Dx(), b.Dy(), nil
}

func scaleImage(img image.Image, longEdge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if longEdge <= 0 || (w <= longEdge && h <= longEdge) {
		return img
	}
	var nw, nh int
	if w >= h {
		nw = longEdge
		nh = int(float64(h) * float64(longEdge) / float64(w))
	} else {
		nh = longEdge
		nw = int(float64(w) * float64(longEdge) / float64(h))
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	// Nearest-neighbor is fine for catalog OCR-ish vision prep and avoids extra deps.
	for y := 0; y < nh; y++ {
		sy := b.Min.Y + y*h/nh
		for x := 0; x < nw; x++ {
			sx := b.Min.X + x*w/nw
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	return dst
}

func cropImage(img image.Image, r image.Rectangle) image.Image {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return img
	}
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if s, ok := img.(subImager); ok {
		return s.SubImage(r)
	}
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.Set(x-r.Min.X, y-r.Min.Y, img.At(x, y))
		}
	}
	return dst
}

func mapTileBBoxToPage(it *ExtractedItem, tile visionTile) {
	if it.BBoxXPct != nil && it.BBoxWidthPct != nil {
		x := tile.OriginXPct + (*it.BBoxXPct)*tile.WidthPct/100
		w := (*it.BBoxWidthPct) * tile.WidthPct / 100
		it.BBoxXPct = &x
		it.BBoxWidthPct = &w
	}
	if it.BBoxYPct != nil && it.BBoxHeightPct != nil {
		y := tile.OriginYPct + (*it.BBoxYPct)*tile.HeightPct/100
		h := (*it.BBoxHeightPct) * tile.HeightPct / 100
		it.BBoxYPct = &y
		it.BBoxHeightPct = &h
	}
}

func mergeExtractedItems(dst []ExtractedItem, incoming []ExtractedItem) []ExtractedItem {
	for _, it := range incoming {
		if dup := findDuplicate(dst, it); dup >= 0 {
			// Prefer the higher-confidence / more complete copy.
			if itemScore(it) > itemScore(dst[dup]) {
				dst[dup] = it
			}
			continue
		}
		dst = append(dst, it)
	}
	return dst
}

func findDuplicate(items []ExtractedItem, cand ExtractedItem) int {
	cname := normalizeName(cand.Name)
	for i, it := range items {
		iou := bboxIoU(it, cand)
		// Strong spatial overlap → same cell across overlapping tiles.
		if iou >= 0.45 {
			return i
		}
		// Same name only counts as a duplicate when boxes also overlap modestly,
		// or when neither side has a box (tile without geometry). Different sizes
		// of the same product name must stay distinct.
		if cname != "" && cname == normalizeName(it.Name) {
			if iou >= 0.2 {
				return i
			}
			if it.BBoxXPct == nil || cand.BBoxXPct == nil {
				if samePrice(it, cand) {
					return i
				}
			}
		}
	}
	return -1
}

func samePrice(a, b ExtractedItem) bool {
	pa := effectivePrice(a)
	pb := effectivePrice(b)
	if pa == nil || pb == nil {
		return pa == nil && pb == nil
	}
	return *pa == *pb
}

func effectivePrice(it ExtractedItem) *int64 {
	if it.PriceDiscountIDR != nil {
		return it.PriceDiscountIDR
	}
	if it.PriceNormalIDR != nil {
		return it.PriceNormalIDR
	}
	return it.PriceIDR
}

func normalizeName(s *string) string {
	if s == nil {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(*s)), " "))
}

func itemScore(it ExtractedItem) float64 {
	score := 0.0
	if it.Name != nil && strings.TrimSpace(*it.Name) != "" {
		score += 10
	}
	if it.PriceNormalIDR != nil || it.PriceDiscountIDR != nil || it.PriceIDR != nil {
		score += 5
	}
	if it.ConfidencePct != nil {
		score += *it.ConfidencePct / 100
	}
	if it.BBoxXPct != nil {
		score += 1
	}
	return score
}

func bboxIoU(a, b ExtractedItem) float64 {
	if a.BBoxXPct == nil || a.BBoxYPct == nil || a.BBoxWidthPct == nil || a.BBoxHeightPct == nil {
		return 0
	}
	if b.BBoxXPct == nil || b.BBoxYPct == nil || b.BBoxWidthPct == nil || b.BBoxHeightPct == nil {
		return 0
	}
	ax0, ay0 := *a.BBoxXPct, *a.BBoxYPct
	ax1, ay1 := ax0+*a.BBoxWidthPct, ay0+*a.BBoxHeightPct
	bx0, by0 := *b.BBoxXPct, *b.BBoxYPct
	bx1, by1 := bx0+*b.BBoxWidthPct, by0+*b.BBoxHeightPct
	ix0 := maxFloat(ax0, bx0)
	iy0 := maxFloat(ay0, by0)
	ix1 := minFloat(ax1, bx1)
	iy1 := minFloat(ay1, by1)
	iw := ix1 - ix0
	ih := iy1 - iy0
	if iw <= 0 || ih <= 0 {
		return 0
	}
	inter := iw * ih
	areaA := (*a.BBoxWidthPct) * (*a.BBoxHeightPct)
	areaB := (*b.BBoxWidthPct) * (*b.BBoxHeightPct)
	union := areaA + areaB - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
