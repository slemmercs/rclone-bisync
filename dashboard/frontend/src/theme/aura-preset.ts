import { definePreset } from '@primevue/themes'
import Aura from '@primevue/themes/aura'

/**
 * Agnoster Carbon & Electric Blue custom PrimeVue v4 preset.
 * Neutral obsidian/carbon surfaces with zero purple tint, paired with crisp Sky/Electric Blue accents.
 */
export const CarbonPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '#F0F9FF',
      100: '#E0F2FE',
      200: '#BAE6FD',
      300: '#7DD3FC',
      400: '#38BDF8', // Primary Hover (Hover states, active links)
      500: '#0EA5E9', // Primary Accent (Active tab, CTAs, progress)
      600: '#0284C7', // Primary Pressed (Active button fill)
      700: '#0369A1',
      800: '#075985',
      900: '#0C4A6E',
      950: '#082F49',
    },
    colorScheme: {
      dark: {
        surface: {
          0: '#ffffff',
          50: '#F8FAFC',  // Text Primary (#F8FAFC)
          100: '#F1F5F9',
          200: '#E2E8F0',
          300: '#CBD5E1', // Steel pill text
          400: '#94A3B8', // Text Secondary (#94A3B8)
          500: '#64748B', // Low-noise metadata / Inactive
          600: '#333947', // Steel pill border
          700: '#262A36', // Hairline Border (Card & table dividers)
          750: '#20242F', // Selected explorer row
          800: '#191C24', // Inputs, table row hover
          850: '#15181F',
          900: '#111318', // Cards, modals, sidebar
          950: '#090A0D', // Main app background
        },
      },
    },
  },
})
