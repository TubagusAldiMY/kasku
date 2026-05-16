// KasKu v3 — Dual Theme System (Dark + Light Gemini)

// Blue-only gradient for light mode (no purple)
const BLUE_GRAD = 'linear-gradient(135deg, #1a73e8 0%, #0ea5e9 55%, #06b6d4 100%)';

const DARK_THEME = {
  bgBase:       '#111318',           // charcoal netral, bukan navy
  bgSurf:       '#181d26',
  bgCard:       '#1e2435',
  bgAccentCard: 'linear-gradient(135deg, #1a2540 0%, #141c30 100%)',
  bgInput:      '#141920',
  bgHover:      'rgba(255,255,255,0.04)',
  border:       'rgba(255,255,255,0.07)',
  borderMd:     'rgba(255,255,255,0.11)',
  borderHi:     'rgba(120,173,245,0.28)',
  accent:       '#78adf5',           // sedikit lebih terang dari sebelumnya
  accentDim:    'rgba(120,173,245,0.12)',
  grad:         '#78adf5',
  logoGrad:     'linear-gradient(135deg, #78adf5 0%, #aac8fa 100%)',
  income:       '#7ec8a0',           // sage green sedikit lebih visible
  expense:      '#f09090',           // soft coral
  warning:      '#f5cc6a',
  text:         '#dce8f5',
  textSec:      '#4e6a88',
  textDim:      '#1c3048',
  bgLoginLeft:  'linear-gradient(160deg, #161c2a 0%, #111318 60%)',
  cardShadow:   'none',
  isDark:       true,
  radius:       12,
  radiusSm:     8,
};

const LIGHT_THEME = {
  bgBase:       '#f0f5ff',
  bgSurf:       '#ffffff',
  bgCard:       '#ffffff',
  bgAccentCard: 'linear-gradient(135deg, #eff6ff 0%, #e0f2fe 50%, #ecfeff 100%)',
  bgInput:      '#f3f6fa',
  bgHover:      'rgba(26,115,232,0.04)',
  border:       'rgba(0,0,0,0.08)',
  borderMd:     'rgba(0,0,0,0.14)',
  borderHi:     'rgba(14,165,233,0.35)',
  accent:       '#0ea5e9',           // sky blue for light mode
  accentDim:    'rgba(14,165,233,0.09)',
  grad:         BLUE_GRAD,           // blue gradient in light mode
  logoGrad:     BLUE_GRAD,
  income:       '#059669',
  expense:      '#dc2626',
  warning:      '#d97706',
  text:         '#0f172a',
  textSec:      '#64748b',
  textDim:      '#cbd5e1',
  bgLoginLeft:  'linear-gradient(160deg, #eff6ff 0%, #e0f2fe 60%)',
  cardShadow:   '0 1px 4px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04)',
  isDark:       false,
  radius:       12,
  radiusSm:     8,
};

// Mutable global theme object — mutated by applyTheme()
const T = { ...DARK_THEME };

function applyTheme(theme) {
  const src = theme === 'light' ? LIGHT_THEME : DARK_THEME;
  Object.assign(T, src);
  // Update body background
  document.body.style.background = src.bgBase;
}

// ─── Style helpers (functions — read T at call time) ──────────────────────────
const card = (extra) => ({
  background: T.bgCard,
  border: `1px solid ${T.border}`,
  borderRadius: T.radius,
  boxShadow: T.cardShadow,
  ...extra,
});

const accentCard = (extra) => ({
  background: T.bgAccentCard,
  border: `1px solid ${T.borderHi}`,
  borderRadius: T.radius,
  boxShadow: T.cardShadow,
  ...extra,
});

const gText = (col) => ({
  background: col || T.grad,
  WebkitBackgroundClip: 'text',
  WebkitTextFillColor: 'transparent',
  backgroundClip: 'text',
});

// Logo always uses gradient (even in dark mode)
const gLogoText = () => ({
  background: T.logoGrad,
  WebkitBackgroundClip: 'text',
  WebkitTextFillColor: 'transparent',
  backgroundClip: 'text',
});

const getINP = () => ({
  background: T.bgInput,
  border: `1px solid ${T.border}`,
  borderRadius: T.radiusSm,
  padding: '10px 14px',
  color: T.text,
  fontSize: 14,
  outline: 'none',
  width: '100%',
  boxSizing: 'border-box',
  fontFamily: 'inherit',
});

const getBtnPrimary = (extra) => ({
  background: T.accent,
  border: 'none',
  borderRadius: T.radiusSm,
  color: '#ffffff',
  fontWeight: 800,
  cursor: 'pointer',
  fontSize: 14,
  fontFamily: 'inherit',
  ...extra,
});

const getGradBtn = (extra) => ({
  background: T.grad,
  border: 'none',
  borderRadius: T.radiusSm,
  color: '#ffffff',
  fontWeight: 800,
  cursor: 'pointer',
  fontSize: 14,
  fontFamily: 'inherit',
  ...extra,
});

const MONO = "'JetBrains Mono', 'Fira Code', monospace";
