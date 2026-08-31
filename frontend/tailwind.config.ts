import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        bg: "#0b1220",
        surface: "#121a2b",
        text: "#eef2ff",
        muted: "#93a0bb",
        accent: "#3b82f6",
        border: "#243049",
      },
    },
  },
  plugins: [],
};

export default config;
