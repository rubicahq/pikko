package ai

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"bytes"
	"testing"
)

func TestPrepareVisionTilesSmallPageSingle(t *testing.T) {
	img := solidRGBA(400, 500, color.RGBA{R: 200, G: 200, B: 210, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	page := PageImage{
		PageNumber: 1,
		MIMEType:   "image/png",
		Base64Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
		Width:      400,
		Height:     500,
	}
	tiles, err := prepareVisionTiles(page)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != 1 {
		t.Fatalf("expected 1 tile, got %d", len(tiles))
	}
	if tiles[0].MIMEType != "image/jpeg" {
		t.Fatalf("expected jpeg, got %s", tiles[0].MIMEType)
	}
	if tiles[0].WidthPct != 100 || tiles[0].HeightPct != 100 {
		t.Fatalf("unexpected tile coverage %#v", tiles[0])
	}
}

func TestPrepareVisionTilesLargePageSplits(t *testing.T) {
	// Above visionTileMinPixels (1800*2400)
	img := solidRGBA(2000, 2800, color.RGBA{R: 180, G: 190, B: 200, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	page := PageImage{
		PageNumber: 1,
		MIMEType:   "image/png",
		Base64Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
		Width:      2000,
		Height:     2800,
	}
	tiles, err := prepareVisionTiles(page)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != 6 {
		t.Fatalf("expected 2x3=6 tiles, got %d", len(tiles))
	}
	for i, tile := range tiles {
		if tile.Base64Data == "" {
			t.Fatalf("tile %d empty", i)
		}
		if tile.WidthPct <= 0 || tile.HeightPct <= 0 {
			t.Fatalf("tile %d bad coverage", i)
		}
	}
}

func TestMergeExtractedItemsDedupesByNameAndBBox(t *testing.T) {
	n1 := "Bebio Minyak Telon"
	n2 := "bebio minyak telon"
	x, y, w, h := 10.0, 20.0, 20.0, 15.0
	x2, y2 := 11.0, 21.0
	a := ExtractedItem{Name: &n1, BBoxXPct: &x, BBoxYPct: &y, BBoxWidthPct: &w, BBoxHeightPct: &h}
	b := ExtractedItem{Name: &n2, BBoxXPct: &x2, BBoxYPct: &y2, BBoxWidthPct: &w, BBoxHeightPct: &h}
	out := mergeExtractedItems(nil, []ExtractedItem{a})
	out = mergeExtractedItems(out, []ExtractedItem{b})
	if len(out) != 1 {
		t.Fatalf("expected dedupe to 1, got %d", len(out))
	}
}

func TestMapTileBBoxToPage(t *testing.T) {
	x, y, w, h := 50.0, 50.0, 20.0, 20.0
	it := ExtractedItem{BBoxXPct: &x, BBoxYPct: &y, BBoxWidthPct: &w, BBoxHeightPct: &h}
	tile := visionTile{OriginXPct: 10, OriginYPct: 20, WidthPct: 40, HeightPct: 30}
	mapTileBBoxToPage(&it, tile)
	// 10 + 50%*40 = 30; 20 + 50%*30 = 35; widths 8 and 6
	if *it.BBoxXPct != 30 || *it.BBoxYPct != 35 || *it.BBoxWidthPct != 8 || *it.BBoxHeightPct != 6 {
		t.Fatalf("mapped bbox unexpected: x=%v y=%v w=%v h=%v", *it.BBoxXPct, *it.BBoxYPct, *it.BBoxWidthPct, *it.BBoxHeightPct)
	}
}

func TestParseExtractionBareArray(t *testing.T) {
	name := "Item A"
	raw := `[{"name":"Item A","uncertain":false}]`
	items, _, err := parseExtraction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name == nil || *items[0].Name != name {
		t.Fatalf("unexpected parse: %#v", items)
	}
}

func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
