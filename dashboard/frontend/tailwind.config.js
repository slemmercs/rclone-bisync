/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./cockpit.html",
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        slate: {
          950: '#020617',
        }
      }
    },
  },
  plugins: [],
}
