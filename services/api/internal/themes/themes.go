package themes

import (
	"fmt"
	"strings"
)

// Theme is a curated storefront palette (PRD §16).
type Theme struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	PrimaryColor    string `json:"primary_color"`
	ButtonForeground string `json:"button_foreground"`
	SurfaceColor    string `json:"surface_color"`
	BorderColor     string `json:"border_color"`
}

var all = []Theme{
	def("pikko_signature", "Pikko Signature", "#FFFFFF", "#15191E", "#FF3D00"),
	def("mono_studio", "Mono Studio", "#FFFFFF", "#171717", "#171717"),
	def("warm_handmade", "Warm Handmade", "#FFF8F0", "#3D3028", "#A65E36"),
	def("nordic_living", "Nordic Living", "#F7F6F2", "#292B28", "#52634D"),
	def("soft_baby", "Soft Baby", "#FFF7FA", "#503A45", "#A84E76"),
	def("modern_baby", "Modern Baby", "#F3F8FC", "#263D52", "#426B92"),
	def("bold_streetwear", "Bold Streetwear", "#171717", "#FAFAFA", "#E7C94E"),
	def("classic_menswear", "Classic Menswear", "#F4F4F2", "#263039", "#354C60"),
	def("clean_beauty", "Clean Beauty", "#FFF9F6", "#473530", "#B85046"),
	def("botanical", "Botanical", "#F4F7EF", "#233E2C", "#426A45"),
	def("earth_studio", "Earth Studio", "#F7F1E8", "#45392E", "#936347"),
	def("ocean_breeze", "Ocean Breeze", "#F1F8FA", "#193E4D", "#267489"),
	def("coastal", "Coastal", "#F8F6F0", "#30434A", "#326F86"),
	def("premium_noir", "Premium Noir", "#121212", "#F7F5EF", "#C6A467"),
	def("quiet_luxury", "Quiet Luxury", "#FAF8F3", "#342D29", "#786247"),
	def("cherry_pop", "Cherry Pop", "#FFF5F5", "#4C202A", "#B62840"),
	def("tangerine", "Tangerine", "#FFF7ED", "#402C20", "#D64B16"),
	def("lavender", "Lavender", "#F8F5FF", "#382D54", "#7853B0"),
	def("fresh_market", "Fresh Market", "#F8FBEF", "#344020", "#56762B"),
	def("retro_club", "Retro Club", "#FBF1DC", "#573C31", "#B44D32"),
	def("chocolate", "Chocolate", "#F8EEE5", "#3E271C", "#865336"),
	def("midnight_blue", "Midnight Blue", "#101C2B", "#F5F8FC", "#87B9EE"),
	def("sakura", "Sakura", "#FFF3F6", "#583642", "#AD5573"),
	def("contemporary", "Contemporary", "#F4F4F7", "#272A35", "#5057A6"),
}

var byID map[string]Theme

func init() {
	byID = make(map[string]Theme, len(all))
	for _, t := range all {
		byID[t.ID] = t
	}
}

func def(id, name, bg, text, primary string) Theme {
	return Theme{
		ID:               id,
		Name:             name,
		BackgroundColor:  strings.ToUpper(bg),
		TextColor:        strings.ToUpper(text),
		PrimaryColor:     strings.ToUpper(primary),
		ButtonForeground: AccessibleOn(primary),
		SurfaceColor:     DeriveSurface(bg),
		BorderColor:      DeriveBorder(bg, text),
	}
}

// All returns the 24 curated themes.
func All() []Theme {
	out := make([]Theme, len(all))
	copy(out, all)
	return out
}

func Get(id string) (Theme, bool) {
	t, ok := byID[strings.ToLower(strings.TrimSpace(id))]
	return t, ok
}

func MustGet(id string) Theme {
	t, ok := Get(id)
	if !ok {
		t, _ = Get("pikko_signature")
	}
	return t
}

func ValidHex(color string) bool {
	c := strings.TrimSpace(color)
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for _, ch := range c[1:] {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

func NormalizeHex(color string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(color))
	if !ValidHex(c) {
		return "", fmt.Errorf("invalid hex color")
	}
	return c, nil
}

func parseRGB(hex string) (r, g, b float64) {
	var ri, gi, bi int
	_, _ = fmt.Sscanf(strings.TrimPrefix(hex, "#"), "%02x%02x%02x", &ri, &gi, &bi)
	return float64(ri), float64(gi), float64(bi)
}

func relativeLuminance(hex string) float64 {
	r, g, b := parseRGB(hex)
	toLin := func(c float64) float64 {
		c = c / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return ((c + 0.055) / 1.055) * ((c + 0.055) / 1.055) * ((c + 0.055) / 1.055) // approx pow
	}
	// Use proper power
	toLin = func(c float64) float64 {
		c = c / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		x := (c + 0.055) / 1.055
		return x * x * x
	}
	R, G, B := toLin(r), toLin(g), toLin(b)
	return 0.2126*R + 0.7152*G + 0.0722*B
}

func contrastRatio(a, b string) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// AccessibleOn picks white or near-black button label for a primary fill.
func AccessibleOn(primary string) string {
	p := strings.ToUpper(strings.TrimSpace(primary))
	if !ValidHex(p) {
		return "#15191E"
	}
	if contrastRatio(p, "#FCFCFD") >= 4.5 {
		return "#FCFCFD"
	}
	if contrastRatio(p, "#15191E") >= 4.5 {
		return "#15191E"
	}
	// Prefer dark text for bright brand oranges.
	return "#15191E"
}

func DeriveSurface(bg string) string {
	r, g, b := parseRGB(bg)
	lum := relativeLuminance(bg)
	if lum < 0.2 {
		return fmt.Sprintf("#%02X%02X%02X", min255(r+18), min255(g+18), min255(b+22))
	}
	return fmt.Sprintf("#%02X%02X%02X", max0(r-8), max0(g-8), max0(b-6))
}

func DeriveBorder(bg, text string) string {
	r, g, b := parseRGB(bg)
	tr, tg, tb := parseRGB(text)
	nr := r*0.85 + tr*0.15
	ng := g*0.85 + tg*0.15
	nb := b*0.85 + tb*0.15
	return fmt.Sprintf("#%02X%02X%02X", nr, ng, nb)
}

func min255(v float64) int {
	if v > 255 {
		return 255
	}
	return int(v)
}

func max0(v float64) int {
	if v < 0 {
		return 0
	}
	return int(v)
}

// ApplyThemeColors returns theme defaults, optionally overridden by custom colors.
func ApplyThemeColors(themeID string, bg, text, primary *string) (Theme, error) {
	base, ok := Get(themeID)
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme")
	}
	out := base
	var err error
	if bg != nil {
		out.BackgroundColor, err = NormalizeHex(*bg)
		if err != nil {
			return Theme{}, err
		}
	}
	if text != nil {
		out.TextColor, err = NormalizeHex(*text)
		if err != nil {
			return Theme{}, err
		}
	}
	if primary != nil {
		out.PrimaryColor, err = NormalizeHex(*primary)
		if err != nil {
			return Theme{}, err
		}
	}
	out.ButtonForeground = AccessibleOn(out.PrimaryColor)
	out.SurfaceColor = DeriveSurface(out.BackgroundColor)
	out.BorderColor = DeriveBorder(out.BackgroundColor, out.TextColor)
	return out, nil
}
