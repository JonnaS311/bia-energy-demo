import type { Config } from 'tailwindcss';

// Tokens de color de la paleta bia.app (fondo oscuro, acento menta). Fuente de verdad: src/lib/colors.ts
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        app: '#0a0a0a',
        canvas: '#0f0f0f',
        panel: '#141414',
        raised: '#1a1a1a',
        hover: '#1f1f1f',
        line: '#27272a',
        'line-strong': '#3f3f46',
        ink: '#ffffff',
        'ink-2': '#a1a1aa',
        'ink-3': '#71717a',
        mint: '#08ddbc',
        'mint-bright': '#17ffdb',
        violet: '#8b5cf6',
        amber: '#f59e0b',
        red: '#ef4444',
        blue: '#3b82f6',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'sans-serif'],
      },
      fontSize: {
        xs: ['12px', '16px'],
        sm: ['14px', '20px'],
        base: ['14px', '20px'],
        md: ['16px', '24px'],
        lg: ['20px', '28px'],
        xl: ['28px', '32px'],
      },
      borderRadius: { DEFAULT: '6px', md: '8px', lg: '10px', badge: '4px' },
      boxShadow: {
        'glow-mint': '0 0 0 1px #08ddbc, 0 0 16px rgba(8, 221, 188, 0.25)',
        'glow-red': '0 0 0 1px #ef4444, 0 0 16px rgba(239, 68, 68, 0.28)',
        'glow-amber': '0 0 0 1px #f59e0b, 0 0 12px rgba(245, 158, 11, 0.22)',
        panel: '0 1px 2px rgba(0, 0, 0, 0.6), 0 8px 24px rgba(0, 0, 0, 0.35)',
      },
      maxWidth: { content: '1440px' },
      transitionTimingFunction: { out: 'cubic-bezier(0.16, 1, 0.3, 1)' },
    },
  },
  plugins: [],
} satisfies Config;
