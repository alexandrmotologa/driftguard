/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        sre: {
          950: '#070a11',
          900: '#0c111d',
          850: '#111726',
          800: '#161f33',
          700: '#222f4c',
          600: '#334366',
        }
      }
    },
  },
  plugins: [],
}
