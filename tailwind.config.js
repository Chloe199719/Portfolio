/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{js,ts,jsx,tsx}", "./components/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        paper: "#171917",
        ink: "#f1f2e9",
        muted: "#a8aea1",
        rose: "#c5ef78",
        line: "#383d34",
      },
      fontFamily: {
        sans: ["Manrope Variable", "sans-serif"],
        serif: ["Space Grotesk Variable", "sans-serif"],
      },
    },
  },
  plugins: [],
};
