/** @type {import('tailwindcss').Config} */
module.exports = {
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "var(--brand-primary)",
          hover: "var(--brand-hover)",
          pressed: "var(--brand-pressed)",
          dark: "var(--brand-accessible-dark)",
          soft: "var(--brand-soft)",
          border: "var(--brand-border)",
        },
        background: {
          DEFAULT: "var(--background)",
          subtle: "var(--background-subtle)",
        },
        surface: {
          DEFAULT: "var(--surface)",
          strong: "var(--surface-strong)",
        },
        foreground: {
          DEFAULT: "var(--foreground)",
          secondary: "var(--foreground-secondary)",
          muted: "var(--foreground-muted)",
          subtle: "var(--foreground-subtle)",
          inverse: "var(--foreground-inverse)",
        },
        dark: "var(--dark)",
        border: {
          DEFAULT: "var(--border)",
          strong: "var(--border-strong)",
        },
        success: {
          DEFAULT: "var(--success)",
          bg: "var(--success-bg)",
          border: "var(--success-border)",
        },
        warning: {
          DEFAULT: "var(--warning)",
          bg: "var(--warning-bg)",
          border: "var(--warning-border)",
        },
        error: {
          DEFAULT: "var(--error)",
          bg: "var(--error-bg)",
          border: "var(--error-border)",
        },
        info: {
          DEFAULT: "var(--info)",
          bg: "var(--info-bg)",
          border: "var(--info-border)",
        },
        neutral: {
          DEFAULT: "var(--neutral)",
          bg: "var(--neutral-bg)",
          border: "var(--neutral-border)",
        },
      },
      borderRadius: {
        xs: "var(--radius-xs)",
        sm: "var(--radius-sm)",
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
        xl: "var(--radius-xl)",
        "2xl": "var(--radius-2xl)",
      },
      boxShadow: {
        xs: "var(--shadow-xs)",
        sm: "var(--shadow-sm)",
        md: "var(--shadow-md)",
        overlay: "var(--shadow-overlay)",
      },
      fontFamily: {
        sans: ["var(--font-inter)", "Inter", "Arial", "sans-serif"],
        display: [
          "var(--font-jakarta)",
          "Plus Jakarta Sans",
          "var(--font-inter)",
          "Inter",
          "Arial",
          "sans-serif",
        ],
      },
      fontSize: {
        page: ["var(--text-page)", { lineHeight: "var(--text-page-leading)", fontWeight: "500" }],
        section: [
          "var(--text-section)",
          { lineHeight: "var(--text-section-leading)", fontWeight: "500" },
        ],
        card: ["var(--text-card)", { lineHeight: "var(--text-card-leading)", fontWeight: "500" }],
        body: ["var(--text-body)", { lineHeight: "var(--text-body-leading)", fontWeight: "400" }],
        table: ["var(--text-table)", { lineHeight: "var(--text-table-leading)", fontWeight: "400" }],
        label: ["var(--text-label)", { lineHeight: "var(--text-label-leading)", fontWeight: "500" }],
        caption: [
          "var(--text-caption)",
          { lineHeight: "var(--text-caption-leading)", fontWeight: "400" },
        ],
        "display-xl": [
          "var(--text-display-xl)",
          { lineHeight: "var(--text-display-xl-leading)", fontWeight: "500", letterSpacing: "-0.03em" },
        ],
        "display-lg": [
          "var(--text-display-lg)",
          { lineHeight: "var(--text-display-lg-leading)", fontWeight: "500", letterSpacing: "-0.02em" },
        ],
        "display-md": [
          "var(--text-display-md)",
          { lineHeight: "var(--text-display-md-leading)", fontWeight: "500", letterSpacing: "-0.02em" },
        ],
        "display-sm": [
          "var(--text-display-sm)",
          { lineHeight: "var(--text-display-sm-leading)", fontWeight: "500", letterSpacing: "-0.02em" },
        ],
      },
      spacing: {
        // 4px base scale (PRD §4) — Tailwind default 4px units; prefer 1/2/3/4/5/6/8/10/12/16/20/24/32
        gutter: "var(--content-gutter)",
      },
      maxWidth: {
        marketing: "1280px",
        reading: "760px",
        auth: "420px",
        dashboard: "1440px",
        checkout: "1120px",
      },
      width: {
        sidebar: "var(--sidebar-width)",
        "sidebar-collapsed": "var(--sidebar-collapsed)",
      },
      height: {
        topbar: "var(--topbar-height)",
      },
      transitionDuration: {
        hover: "150ms",
        standard: "200ms",
        dialog: "220ms",
      },
      keyframes: {
        "skeleton-pulse": {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0.45" },
        },
      },
      animation: {
        skeleton: "skeleton-pulse 1.4s ease-in-out infinite",
      },
    },
  },
};
