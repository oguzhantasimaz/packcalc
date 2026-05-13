import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        display: ["var(--font-display)", "ui-serif", "Georgia", "serif"],
        sans: ["var(--font-sans)", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["var(--font-mono)", "ui-monospace", "SFMono-Regular", "monospace"],
      },
      colors: {
        // Brand cyan and its accents. Hue locked at #00ACDE; lightness
        // varied for hover / glow / dim states.
        brand: {
          50: "#E6F8FE",
          100: "#B8E9FA",
          200: "#7FD5F4",
          300: "#3FBEEC",
          400: "#00ACDE", // primary — exact spec
          500: "#0098C3",
          600: "#007DA8",
          700: "#005F82",
          800: "#02425C",
          900: "#062B3D",
        },
        // Cool dark canvas tuned so cyan accents pop.
        ink: {
          950: "#050A10",
          900: "#080F18",
          850: "#0C1320",
          800: "#111A28",
          700: "#172234",
          600: "#1E2D44",
          500: "#2A3D58",
          400: "#4E6480",
          300: "#7B8FAA",
          200: "#B0C0D4",
          100: "#DCE6F2",
          50: "#EFF4FA",
        },
      },
      animation: {
        "drift-a": "driftA 22s ease-in-out infinite alternate",
        "drift-b": "driftB 28s ease-in-out infinite alternate",
        "drift-c": "driftC 36s linear infinite",
        "scan": "scan 6s linear infinite",
        "fade-up": "fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) both",
        "pulse-glow": "pulseGlow 3.5s ease-in-out infinite",
      },
      keyframes: {
        driftA: {
          "0%": { transform: "translate3d(0,0,0) scale(1)" },
          "100%": { transform: "translate3d(10vw, 8vh, 0) scale(1.15)" },
        },
        driftB: {
          "0%": { transform: "translate3d(0,0,0) scale(1)" },
          "100%": { transform: "translate3d(-12vw, -6vh, 0) scale(1.1)" },
        },
        driftC: {
          "0%": { transform: "translate(0, 0)" },
          "100%": { transform: "translate(-40px, -40px)" },
        },
        scan: {
          "0%, 100%": { opacity: "0.25" },
          "50%": { opacity: "0.5" },
        },
        fadeUp: {
          "0%": { opacity: "0", transform: "translate3d(0, 14px, 0)" },
          "100%": { opacity: "1", transform: "translate3d(0, 0, 0)" },
        },
        pulseGlow: {
          "0%, 100%": { boxShadow: "0 0 0 0 rgba(0,172,222,0.0)" },
          "50%": { boxShadow: "0 0 32px 0 rgba(0,172,222,0.35)" },
        },
      },
      boxShadow: {
        "glow-sm": "0 0 12px rgba(0,172,222,0.35)",
        "glow-md": "0 0 32px rgba(0,172,222,0.30)",
        "glow-lg": "0 0 60px rgba(0,172,222,0.25)",
      },
    },
  },
  plugins: [],
};

export default config;
