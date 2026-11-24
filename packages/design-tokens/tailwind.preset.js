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
      },
      spacing: {
        // 4px base scale (PRD §13) — Tailwind already uses 4px units
      },
      maxWidth: {
        marketing: "1280px",
        reading: "760px",
        auth: "420px",
        dashboard: "1440px",
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
    },
  },
};
