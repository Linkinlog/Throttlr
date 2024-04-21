/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/**/*.templ"],
  theme: {
    extend: {
      colors: {
        primary: "#865DFF",
        secondary: "#E384FF",
        accent: "#FFA3FD",
        dark: "#151515",
      },
      fontFamily: {
        sans: ["Outfit", "sans-serif"],
      },
    },
  },
  plugins: [],
};
