/** @type {import('tailwindcss').Config} */
// Design tokens: see design/DESIGN.md section 1. Dark theme only.
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}', '!./src/views/_legacy/**'],
  theme: {
    extend: {
      colors: {
        bg: '#0A0E14',
        surface: {
          DEFAULT: 'rgba(15,21,32,0.85)',
          solid: '#0F1520',
          2: '#0B1018',
          3: '#141C2A'
        },
        raised: '#1B2535',
        terminal: '#070A0F',
        titlebar: '#111824',
        border: {
          DEFAULT: '#1F2A3A',
          strong: '#243247',
          stronger: '#2A3850'
        },
        divider: '#1A2332',
        text: {
          DEFAULT: '#E6EDF3',
          2: '#C9D4E0'
        },
        muted: '#94A3B8',
        faint: '#64748B',
        accent: {
          DEFAULT: '#3DDC97',
          hover: '#7EF0BE',
          ink: '#04130C'
        },
        info: '#38BDF8',
        warn: '#F5B342',
        danger: '#F87171',
        violet: '#A78BFA',
        rose: '#FB7185',
        teal: '#2DD4BF',
        gold: '#F5B342',
        silver: '#C9D4E0',
        bronze: '#E0915A'
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace']
      },
      fontSize: {
        meta: ['12.5px', { lineHeight: '1.45' }],
        hero: ['34px', { lineHeight: '1.15', letterSpacing: '-0.02em', fontWeight: '700' }],
        title: ['30px', { lineHeight: '1.2', letterSpacing: '-0.01em', fontWeight: '700' }]
      },
      borderRadius: {
        chip: '6px',
        ctl: '9px',
        btn: '10px',
        card: '14px',
        panel: '16px',
        hero: '18px'
      },
      boxShadow: {
        glow: '0 0 24px rgba(61,220,151,.35)',
        'glow-sm': '0 0 8px #3DDC97',
        ring: 'inset 0 0 0 1px #243247',
        'ring-strong': 'inset 0 0 0 1px #2A3850',
        selected: '0 0 0 3px rgba(61,220,151,.15)'
      },
      maxWidth: {
        container: '1320px'
      }
    }
  },
  plugins: []
}
