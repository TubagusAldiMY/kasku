// KasKu — Utility functions

const formatIDR = (amount) => {
  if (amount === null || amount === undefined) return 'Rp 0';
  return 'Rp ' + Math.floor(amount).toLocaleString('id-ID');
};

const formatQty = (qty, unit) => {
  const n = parseFloat(qty);
  if (n === 0) return `0 ${unit}`;
  if (Number.isInteger(n)) return `${n.toLocaleString('id-ID')} ${unit}`;
  // Show up to 8 significant decimals, trim trailing zeros
  const str = n.toPrecision(8).replace(/\.?0+$/, '');
  return `${str} ${unit}`;
};

const formatDate = (dateStr) => {
  if (!dateStr) return '';
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
};

const formatDateShort = (dateStr) => {
  if (!dateStr) return '';
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
};

const today = () => new Date().toISOString().split('T')[0];

const monthRange = () => {
  const now = new Date();
  const first = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().split('T')[0];
  const last = new Date(now.getFullYear(), now.getMonth() + 1, 0).toISOString().split('T')[0];
  return { from: first, to: last };
};

const assetTypeLabel = (type) => {
  const m = { GOLD: 'Emas', CRYPTO: 'Kripto', STOCK: 'Saham', OTHER: 'Lainnya' };
  return m[type] || type;
};

const accountTypeLabel = (type) => {
  const m = { BANK: 'Bank', EWALLET: 'E-Wallet', CASH: 'Tunai' };
  return m[type] || type;
};

const ACCENT_COLORS = [
  { label: 'Sky Blue', value: '#38bdf8', dark: '#0284c7' },
  { label: 'Emerald', value: '#34d399', dark: '#059669' },
  { label: 'Violet', value: '#a78bfa', dark: '#7c3aed' },
];
