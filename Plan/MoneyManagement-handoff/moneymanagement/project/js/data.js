// KasKu — Data Store & localStorage helpers

const daysAgo = (n) => {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().split('T')[0];
};

const SEED_CATEGORIES = [
  { id: 'cat-1', name: 'Gaji', type: 'INCOME', icon: '💼', isDefault: true },
  { id: 'cat-2', name: 'Freelance', type: 'INCOME', icon: '💻', isDefault: true },
  { id: 'cat-3', name: 'Bonus', type: 'INCOME', icon: '🎁', isDefault: true },
  { id: 'cat-4', name: 'Investasi Masuk', type: 'INCOME', icon: '📈', isDefault: true },
  { id: 'cat-5', name: 'Makan & Minum', type: 'EXPENSE', icon: '🍜', isDefault: true },
  { id: 'cat-6', name: 'Transport', type: 'EXPENSE', icon: '🚗', isDefault: true },
  { id: 'cat-7', name: 'Belanja', type: 'EXPENSE', icon: '🛍️', isDefault: true },
  { id: 'cat-8', name: 'Tagihan & Utilitas', type: 'EXPENSE', icon: '⚡', isDefault: true },
  { id: 'cat-9', name: 'Hiburan', type: 'EXPENSE', icon: '🎮', isDefault: true },
  { id: 'cat-10', name: 'Kesehatan', type: 'EXPENSE', icon: '💊', isDefault: true },
  { id: 'cat-11', name: 'Pendidikan', type: 'EXPENSE', icon: '📚', isDefault: true },
  { id: 'cat-12', name: 'Lain-lain', type: 'BOTH', icon: '📌', isDefault: true },
];

const SEED_ACCOUNTS = [
  { id: 'acc-1', name: 'BCA', type: 'BANK', balance: 5500000, currency: 'IDR', color: '#0ea5e9', icon: '🏦' },
  { id: 'acc-2', name: 'Seabank', type: 'BANK', balance: 2300000, currency: 'IDR', color: '#8b5cf6', icon: '🏦' },
  { id: 'acc-3', name: 'Dana', type: 'EWALLET', balance: 750000, currency: 'IDR', color: '#f59e0b', icon: '💳' },
  { id: 'acc-4', name: 'Cash', type: 'CASH', balance: 500000, currency: 'IDR', color: '#4ade80', icon: '💵' },
];

const SEED_INVESTMENTS = [
  { id: 'inv-1', name: 'Emas Antam', type: 'GOLD', symbol: 'XAU', coinId: '', quantity: 10.5, unitLabel: 'gram' },
  { id: 'inv-2', name: 'Bitcoin', type: 'CRYPTO', symbol: 'BTC', coinId: 'bitcoin', quantity: 0.0025, unitLabel: 'BTC' },
  { id: 'inv-3', name: 'Ethereum', type: 'CRYPTO', symbol: 'ETH', coinId: 'ethereum', quantity: 0.15, unitLabel: 'ETH' },
];

const SEED_TRANSACTIONS = [
  { id: 'trx-1', type: 'INCOME', amount: 8000000, categoryId: 'cat-1', accountId: 'acc-1', date: daysAgo(0), note: 'Gaji bulan April' },
  { id: 'trx-2', type: 'EXPENSE', amount: 45000, categoryId: 'cat-5', accountId: 'acc-3', date: daysAgo(0), note: 'Makan siang warteg' },
  { id: 'trx-3', type: 'EXPENSE', amount: 15000, categoryId: 'cat-6', accountId: 'acc-3', date: daysAgo(1), note: 'Ojol ke kantor' },
  { id: 'trx-4', type: 'EXPENSE', amount: 250000, categoryId: 'cat-7', accountId: 'acc-2', date: daysAgo(1), note: 'Belanja bulanan Indomaret' },
  { id: 'trx-5', type: 'EXPENSE', amount: 8500, categoryId: 'cat-5', accountId: 'acc-3', date: daysAgo(2), note: 'Kopi' },
  { id: 'trx-6', type: 'INCOME', amount: 500000, categoryId: 'cat-2', accountId: 'acc-1', date: daysAgo(3), note: 'Freelance desain logo' },
  { id: 'trx-7', type: 'EXPENSE', amount: 350000, categoryId: 'cat-8', accountId: 'acc-1', date: daysAgo(5), note: 'Listrik + Internet' },
  { id: 'trx-8', type: 'EXPENSE', amount: 75000, categoryId: 'cat-9', accountId: 'acc-3', date: daysAgo(6), note: 'Netflix' },
  { id: 'trx-9', type: 'EXPENSE', amount: 120000, categoryId: 'cat-10', accountId: 'acc-1', date: daysAgo(7), note: 'Vitamin & suplemen' },
  { id: 'trx-10', type: 'EXPENSE', amount: 25000, categoryId: 'cat-5', accountId: 'acc-3', date: daysAgo(8), note: 'Gorengan + es teh' },
  { id: 'trx-11', type: 'INCOME', amount: 200000, categoryId: 'cat-3', accountId: 'acc-1', date: daysAgo(10), note: 'Bonus project' },
  { id: 'trx-12', type: 'EXPENSE', amount: 180000, categoryId: 'cat-7', accountId: 'acc-2', date: daysAgo(12), note: 'Baju kerja' },
];

const KasKuStore = {
  load() {
    const get = (key, fallback) => {
      try {
        const v = localStorage.getItem('kasku_' + key);
        return v ? JSON.parse(v) : fallback;
      } catch { return fallback; }
    };
    return {
      accounts: get('accounts', SEED_ACCOUNTS),
      investments: get('investments', SEED_INVESTMENTS),
      transactions: get('transactions', SEED_TRANSACTIONS),
      categories: get('categories', SEED_CATEGORIES),
      theme: get('theme', 'dark'),
      accentColor: get('accentColor', '#38bdf8'),
      cardStyle: get('cardStyle', 'rounded'),
      syncStatus: 'synced',
      pendingCount: 0,
    };
  },
  save(key, value) {
    try {
      localStorage.setItem('kasku_' + key, JSON.stringify(value));
    } catch {}
  },
  genId(prefix) {
    return prefix + '-' + Date.now() + '-' + Math.random().toString(36).slice(2, 7);
  }
};
