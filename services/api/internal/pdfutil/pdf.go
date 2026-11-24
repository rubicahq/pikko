package pdfutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultMaxBytes  = 25 << 20 // 25 MB
	DefaultMaxPages  = 50
	DefaultMaxItems  = 500
)

var pdfMagic = []byte("%PDF-")

func IsPDF(head []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(head), pdfMagic)
}

// CountPages estimates page count from PDF content (no external tools).
func CountPages(data []byte) (int, error) {
	if !IsPDF(data) {
		return 0, fmt.Errorf("not a PDF")
	}
	// Prefer /Type /Pages Count
	reCount := regexp.MustCompile(`/Type\s*/Pages[^>]*?/Count\s+(\d+)`)
	if m := reCount.FindSubmatch(data); len(m) == 2 {
		n, err := strconv.Atoi(string(m[1]))
		if err == nil && n > 0 {
			return n, nil
		}
	}
	// Fallback: count page objects
	rePage := regexp.MustCompile(`/Type\s*/Page[^s]`)
	matches := rePage.FindAllIndex(data, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("unable to determine PDF page count")
	}
	return len(matches), nil
}

type PageImage struct {
	PageNumber int
	PNG        []byte
	Width      int
	Height     int
}

// Renderer turns PDF pages into PNG images.
type Renderer interface {
	RenderPages(pdf []byte, maxPages int) ([]PageImage, error)
}

// PopplerRenderer uses pdftoppm when available.
type PopplerRenderer struct {
	Bin string // default pdftoppm
	DPI int
}

func (r *PopplerRenderer) RenderPages(pdf []byte, maxPages int) ([]PageImage, error) {
	bin := r.Bin
	if bin == "" {
		bin = "pdftoppm"
	}
	dpi := r.DPI
	if dpi <= 0 {
		dpi = 120
	}
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("pdftoppm not available")
	}
	dir, err := os.MkdirTemp("", "pikko-pdf-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	in := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(in, pdf, 0o600); err != nil {
		return nil, err
	}
	outPrefix := filepath.Join(dir, "page")
	args := []string{"-png", "-r", strconv.Itoa(dpi)}
	if maxPages > 0 {
		args = append(args, "-f", "1", "-l", strconv.Itoa(maxPages))
	}
	args = append(args, in, outPrefix)
	cmd := exec.Command(bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	var pages []PageImage
	for i := 1; i <= maxPages+5; i++ {
		// pdftoppm names: page-1.png or page-01.png
		candidates := []string{
			fmt.Sprintf("%s-%d.png", outPrefix, i),
			fmt.Sprintf("%s-%02d.png", outPrefix, i),
			fmt.Sprintf("%s-%03d.png", outPrefix, i),
		}
		var data []byte
		for _, c := range candidates {
			b, err := os.ReadFile(c)
			if err == nil {
				data = b
				break
			}
		}
		if data == nil {
			if i == 1 {
				return nil, fmt.Errorf("pdftoppm produced no pages")
			}
			break
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		pages = append(pages, PageImage{PageNumber: i, PNG: data, Width: cfg.Width, Height: cfg.Height})
		if maxPages > 0 && len(pages) >= maxPages {
			break
		}
	}
	return pages, nil
}

// PlaceholderRenderer generates solid PNG pages (tests / fallback without poppler).
type PlaceholderRenderer struct {
	Width  int
	Height int
}

func (r *PlaceholderRenderer) RenderPages(pdf []byte, maxPages int) ([]PageImage, error) {
	n, err := CountPages(pdf)
	if err != nil {
		n = 1
	}
	if maxPages > 0 && n > maxPages {
		n = maxPages
	}
	w, h := r.Width, r.Height
	if w <= 0 {
		w = 600
	}
	if h <= 0 {
		h = 800
	}
	out := make([]PageImage, 0, n)
	for i := 1; i <= n; i++ {
		pngBytes, err := solidPNG(w, h, color.RGBA{R: 243, G: 243, B: 247, A: 255})
		if err != nil {
			return nil, err
		}
		out = append(out, PageImage{PageNumber: i, PNG: pngBytes, Width: w, Height: h})
	}
	return out, nil
}

func solidPNG(w, h int, c color.RGBA) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewRenderer prefers poppler; falls back to placeholders.
func NewRenderer() Renderer {
	if _, err := exec.LookPath("pdftoppm"); err == nil {
		return &PopplerRenderer{DPI: 120}
	}
	return &PlaceholderRenderer{}
}

// MinimalPDF returns a tiny valid 1-page PDF for tests.
func MinimalPDF() []byte {
	return []byte(`%PDF-1.4
1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj
2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj
3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources<< /Font<< /F1 5 0 R >> >> >>endobj
4 0 obj<< /Length 44 >>stream
BT /F1 24 Tf 100 700 Td (Hello) Tj ET
endstream
endobj
5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj
xref
0 6
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000266 00000 n 
0000000361 00000 n 
trailer<< /Size 6 /Root 1 0 R >>
startxref
429
%%EOF
`)
}
