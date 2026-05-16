// KasKu v3 — Keuangan + Investasi (theme-aware)

function KeuanganPage({ state, onEdit, onDelete, onAdd }) {
  const { accounts } = state;
  const total = accounts.reduce((s, a) => s + a.balance, 0);
  const typeIcon = { BANK: '🏦', EWALLET: '💳', CASH: '💵' };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Hero */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
        <div style={{ ...accentCard(), padding: '24px', gridColumn: '1 / 3' }}>
          <div style={{ fontSize: 10, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 10, ...gText() }}>Total Saldo</div>
          <div style={{ fontSize: 40, fontWeight: 900, letterSpacing: '-1.5px', fontFamily: MONO, ...gText() }}>{formatIDR(total)}</div>
          <div style={{ fontSize: 12, color: T.textSec, marginTop: 8 }}>{accounts.length} rekening aktif</div>
        </div>
        <div style={{ ...card(), padding: '20px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: 12 }}>
          <div>
            <div style={{ fontSize: 10, color: T.textSec, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 8 }}>Terbesar</div>
            {accounts.length > 0 && (() => {
              const top = [...accounts].sort((a, b) => b.balance - a.balance)[0];
              return (
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                    <div style={{ width: 8, height: 8, borderRadius: '50%', background: top.color }} />
                    <span style={{ fontSize: 13, fontWeight: 700, color: T.text }}>{top.name}</span>
                  </div>
                  <div style={{ fontSize: 18, fontWeight: 800, color: top.color, fontFamily: MONO }}>{formatIDR(top.balance)}</div>
                </div>
              );
            })()}
          </div>
          <button onClick={onAdd} style={{ ...getBtnPrimary(), padding: '9px 0', width: '100%', background: T.grad }}>
            + Tambah Akun
          </button>
        </div>
      </div>

      {/* Account rows */}
      <div style={{ ...card(), overflow: 'hidden' }}>
        <div style={{ padding: '12px 22px', borderBottom: `1px solid ${T.border}`, fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em' }}>
          Daftar Rekening
        </div>
        {accounts.length === 0 && (
          <div style={{ padding: 40, textAlign: 'center', color: T.textSec, fontSize: 14 }}>Belum ada rekening. Tambahkan yang pertama!</div>
        )}
        {accounts.map((acc, i) => {
          const pct = total > 0 ? (acc.balance / total) * 100 : 0;
          return (
            <div key={acc.id} style={{
              display: 'grid', gridTemplateColumns: 'auto 1fr auto auto',
              alignItems: 'center', gap: 16, padding: '16px 22px',
              borderBottom: i < accounts.length - 1 ? `1px solid ${T.border}` : 'none',
              transition: 'background 0.1s',
            }}
              onMouseEnter={e => e.currentTarget.style.background = T.bgHover}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: acc.color + '15', border: `1px solid ${acc.color}25`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18 }}>
                {typeIcon[acc.type]}
              </div>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 5 }}>
                  <span style={{ fontSize: 14, fontWeight: 700, color: T.text }}>{acc.name}</span>
                  <span style={{ fontSize: 10, padding: '1px 7px', borderRadius: 99, background: acc.color + '15', color: acc.color, fontWeight: 600 }}>{accountTypeLabel(acc.type)}</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <div style={{ flex: 1, height: 3, borderRadius: 2, background: T.bgInput, maxWidth: 160 }}>
                    <div style={{ height: '100%', borderRadius: 2, background: acc.color, width: `${pct}%` }} />
                  </div>
                  <span style={{ fontSize: 10, color: T.textSec }}>{pct.toFixed(1)}%</span>
                </div>
              </div>
              <div style={{ fontSize: 18, fontWeight: 800, color: acc.color, fontFamily: MONO, textAlign: 'right' }}>{formatIDR(acc.balance)}</div>
              <div style={{ display: 'flex', gap: 4 }}>
                <button onClick={() => onEdit(acc)} style={{ width: 30, height: 30, borderRadius: 6, border: `1px solid ${T.border}`, background: T.bgInput, cursor: 'pointer', color: T.textSec, fontSize: 12, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✏</button>
                <button onClick={() => onDelete(acc.id)} style={{ width: 30, height: 30, borderRadius: 6, border: `1px solid ${T.expense}25`, background: T.expense + '08', cursor: 'pointer', color: T.expense, fontSize: 12, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Breakdown */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {['BANK','EWALLET','CASH'].map(type => {
          const accs = accounts.filter(a => a.type === type);
          const sub = accs.reduce((s, a) => s + a.balance, 0);
          if (!accs.length) return null;
          return (
            <div key={type} style={{ ...card(), padding: '18px 20px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                <span style={{ fontSize: 20 }}>{typeIcon[type]}</span>
                <span style={{ fontSize: 12, color: T.textSec, fontWeight: 600 }}>{accountTypeLabel(type)}</span>
              </div>
              <div style={{ fontSize: 20, fontWeight: 800, color: T.text, fontFamily: MONO }}>{formatIDR(sub)}</div>
              <div style={{ fontSize: 11, color: T.textSec, marginTop: 4 }}>{accs.length} rekening</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── Investasi ────────────────────────────────────────────────────────────────
function InvestasiPage({ state, onEdit, onDelete, onAdd }) {
  const { investments } = state;
  const assetColor = { GOLD: '#e8a838', CRYPTO: '#7c9ef8', STOCK: '#5bbfb5', OTHER: '#8899a6' };
  const assetBg    = { GOLD: '#e8a83812', CRYPTO: '#7c9ef812', STOCK: '#5bbfb512', OTHER: '#8899a612' };
  const assetIcon  = { GOLD: '◆', CRYPTO: '◈', STOCK: '◎', OTHER: '○' };
  const typeLabel  = { GOLD: 'Emas', CRYPTO: 'Kripto', STOCK: 'Saham', OTHER: 'Lainnya' };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', paddingBottom: 4 }}>
        <div>
          <div style={{ fontSize: 10, color: T.textSec, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 6 }}>Portofolio Investasi</div>
          <div style={{ fontSize: 26, fontWeight: 800, color: T.text }}>
            {investments.length} <span style={{ fontSize: 15, color: T.textSec, fontWeight: 400 }}>instrumen</span>
          </div>
        </div>
        <button onClick={onAdd} style={{ ...getBtnPrimary(), padding: '9px 18px', background: T.grad }}>+ Instrumen</button>
      </div>

      {/* Info */}
      <div style={{ padding: '9px 14px', borderRadius: T.radiusSm, background: T.accentDim, border: `1px solid ${T.borderHi}`, fontSize: 11, color: T.textSec, display: 'flex', gap: 8, alignItems: 'center' }}>
        <span style={{ color: T.accent, fontWeight: 700 }}>ℹ</span>
        Nilai investasi dalam satuan unit. Harga real-time dari server saat koneksi tersedia.
      </div>

      {/* Clean list */}
      <div style={{ ...card(), overflow: 'hidden' }}>
        {investments.length === 0 && (
          <div style={{ padding: 48, textAlign: 'center', color: T.textSec, fontSize: 14 }}>
            Belum ada instrumen. Tambahkan yang pertama!
          </div>
        )}
        {investments.map((inv, i) => {
          const col = assetColor[inv.type] || assetColor.OTHER;
          const bg  = assetBg[inv.type]   || assetBg.OTHER;
          return (
            <div key={inv.id} style={{
              display: 'grid', gridTemplateColumns: 'auto 1fr auto auto',
              alignItems: 'center', gap: 16, padding: '16px 22px',
              borderBottom: i < investments.length - 1 ? `1px solid ${T.border}` : 'none',
              transition: 'background 0.1s',
            }}
              onMouseEnter={e => e.currentTarget.style.background = T.bgHover}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
              {/* Icon */}
              <div style={{ width: 44, height: 44, borderRadius: 10, flexShrink: 0, background: bg, border: `1px solid ${col}20`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: col, fontSize: 18, fontWeight: 700 }}>
                {assetIcon[inv.type]}
              </div>
              {/* Name + meta */}
              <div>
                <div style={{ fontSize: 14, fontWeight: 700, color: T.text, marginBottom: 4 }}>{inv.name}</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ fontSize: 10, padding: '1px 7px', borderRadius: 99, background: bg, color: col, fontWeight: 700, border: `1px solid ${col}20` }}>{typeLabel[inv.type]}</span>
                  <span style={{ fontSize: 11, color: T.textSec, letterSpacing: '0.06em' }}>{inv.symbol}</span>
                </div>
              </div>
              {/* Quantity */}
              <div style={{ textAlign: 'right', minWidth: 120 }}>
                <div style={{ fontSize: 22, fontWeight: 900, color: col, fontFamily: MONO, letterSpacing: '-0.5px' }}>
                  {formatQty(inv.quantity, inv.unitLabel)}
                </div>
                <div style={{ fontSize: 10, color: T.textSec, marginTop: 2 }}>jumlah unit</div>
              </div>
              {/* Actions */}
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                <button onClick={() => onEdit(inv)} style={{ width: 32, height: 32, borderRadius: T.radiusSm, border: `1px solid ${T.border}`, background: T.bgInput, cursor: 'pointer', color: T.textSec, fontSize: 13, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✏</button>
                <button onClick={() => onDelete(inv.id)} style={{ width: 32, height: 32, borderRadius: T.radiusSm, border: `1px solid ${T.expense}25`, background: T.expense + '0a', cursor: 'pointer', color: T.expense, fontSize: 13, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Type summary strip */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 10 }}>
        {['GOLD','CRYPTO','STOCK','OTHER'].map(type => {
          const items = investments.filter(i => i.type === type);
          if (!items.length) return null;
          const col = assetColor[type];
          return (
            <div key={type} style={{ ...card(), padding: '14px 18px', borderLeft: `3px solid ${col}` }}>
              <div style={{ fontSize: 10, color: col, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em', marginBottom: 5 }}>{typeLabel[type]}</div>
              <div style={{ fontSize: 22, fontWeight: 800, color: T.text, fontFamily: MONO }}>{items.length}</div>
              <div style={{ fontSize: 11, color: T.textSec, marginTop: 1 }}>instrumen</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

Object.assign(window, { KeuanganPage, InvestasiPage });
