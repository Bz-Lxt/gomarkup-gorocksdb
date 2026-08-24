/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        abyss: "#041018",
        trench: "#0B1F2C",
        foam: "#E7F6F2",
        sonar: "#3EE0C8",
        amber: "#F2B84B",
        coral: "#FF6B4A",
        mist: "#7A9AAD",
      },
      fontFamily: {
        display: ["Oxanium", "sans-serif"],
        sans: ["Sora", "sans-serif"],
        mono: ["IBM Plex Mono", "monospace"],
      },
      boxShadow: {
        sonar: "0 0 24px rgba(62, 224, 200, 0.25)",
        amber: "0 0 20px rgba(242, 184, 75, 0.28)",
      },
    },
  },
  plugins: [],
};
