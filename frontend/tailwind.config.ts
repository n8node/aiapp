import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        bg: "#f5f5f5",
        surface: "#ffffff",
        text: "#09090b",
        muted: "#71717b",
        accent: "#2563eb",
        border: "#e5e5e5",
      },
      boxShadow: {
        sm: "0 1px 2px 0 rgb(0 0 0 / 0.04)",
      },
    },
  },
  plugins: [],
};

export default config;
