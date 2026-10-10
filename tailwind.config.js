/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{js,ts,jsx,tsx}", "./components/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        paper: "#191516",
        ink: "#f5eeee",
        muted: "#bba5a7",
        rose: "#e76a6a",
        line: "#483337",
      },
      fontFamily: {
        sans: ["Manrope Variable", "sans-serif"],
        serif: ["Space Grotesk Variable", "sans-serif"],
      },
    },
  },
  plugins: [],
};
