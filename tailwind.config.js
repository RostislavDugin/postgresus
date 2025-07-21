/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './*.html', './storages/**/*.html', './pages/**/*.html'],
  theme: {
    extend: {
      fontFamily: {
        jost: ['Jost', 'sans-serif'],
        bebas: ['Bebas Neue', 'cursive'],
      },
    },
  },
  plugins: [],
};
