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
        carbon: {
          950: '#090A0D', // App Canvas (Main background)
          900: '#111318', // Card / Panel (Cards, modals, sidebar)
          850: '#15181F',
          800: '#191C24', // Elevated / Hover (Inputs, table row hover)
          750: '#20242F', // Active / Selected (Selected explorer row)
          700: '#262A36', // Hairline Border (Card & table dividers)
          600: '#333947', // Secondary border / Steel pill border
          500: '#64748B', // Muted text / Placeholders / Inactive icons
          400: '#94A3B8', // Text Secondary (Timestamps, column headers)
          300: '#CBD5E1', // Steel pill text
          100: '#F1F5F9',
          50: '#F8FAFC',  // Text Primary (Headers, file names, hashes)
        },
        electric: {
          300: '#7DD3FC',
          400: '#38BDF8', // Primary Hover (Hover states, active links)
          500: '#0EA5E9', // Primary Accent (Active tab, CTAs, progress)
          600: '#0284C7', // Primary Pressed (Active button fill)
          700: '#0369A1',
          900: '#0C4A6E',
          950: '#082F49',
        },
        status: {
          success: '#10B981',
          warning: '#F59E0B',
          danger: '#EF4444',
        }
      }
    },
  },
  plugins: [],
}
