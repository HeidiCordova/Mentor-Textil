/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        edge: {
          50:  'rgb(var(--edge-50)  / <alpha-value>)',
          100: 'rgb(var(--edge-100) / <alpha-value>)',
          200: 'rgb(var(--edge-200) / <alpha-value>)',
          300: 'rgb(var(--edge-300) / <alpha-value>)',
          400: 'rgb(var(--edge-400) / <alpha-value>)',
          500: 'rgb(var(--edge-500) / <alpha-value>)',
          600: 'rgb(var(--edge-600) / <alpha-value>)',
          700: 'rgb(var(--edge-700) / <alpha-value>)',
          800: 'rgb(var(--edge-800) / <alpha-value>)',
          900: 'rgb(var(--edge-900) / <alpha-value>)',
          950: 'rgb(var(--edge-950) / <alpha-value>)'
        },
        stop: {
          assigned: '#dc2626',
          unassigned: '#eab308',
          justified: '#16a34a'
        },
        production: {
          active: '#3b82f6',
          idle: '#475569',
          event: '#ef4444'
        }
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'monospace'],
        sans: ['Inter', 'system-ui', 'sans-serif']
      }
    }
  },
  plugins: []
}
