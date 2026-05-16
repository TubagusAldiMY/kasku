// KasKu v3 — Dashboard (theme-aware)

function DashboardPage({ state, onNav }) {
  const { accounts, investments, transactions, categories } = state;
  const total    = accounts.reduce((s, a) => s + a.balance, 0);
  const { from, to } = monthRange();
  const thisMth  = transactions.filter(t => t.date >= from && t.date <= to);
  const income   = thisMth.filter(t => t.type === 'INCOME').reduce((s, t) => s + t.amount, 0);
  const expense  = thisMth.filter(t => t.type === 'EXPENSE').reduce((s, t) => s + t.amount, 0);
  const net      = income - expense;
  const recent   = [...transactions].sort((a, b) => b.date.localeCompare(a.date)).slice(0, 8);
  const monthName = new Date().toLocaleDateString('id-ID', { month: 'long', year: 'numeric' });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>

      {/* ROW 1 — Hero: Total Keuangan */}
      <div style={{
        ...accentCard(), padding: '28px 32px',
        display: 'grid', gridTemplateColumns: '1fr auto', gap: 24, alignItems: 'center',
        position: 'relative', overflow: 'hidden',
      }}>
        <div style={{ position: 'absolute', top: 0, right: 0, width: 200, height: '100%', background: `radial-gradient(ellipse at right, ${T.accentDim} 0%, transparent 70%)`, pointerEvents: 'none' }} />
        <div>
          <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 10 }}>
            Total Saldo Keuangan
          </div>
          <div style={{ fontSize: 44, fontWeight: 900, letterSpacing: '-2px', fontFamily: MONO, ...gText(), lineHeight: 1 }}>
            {formatIDR(total)}
          </div>
          <div style={{ display: 'flex', gap: 8, marginTop: 16, flexWrap: 'wrap' }}>
            {accounts.map(acc => (
              <div key={acc.id} style={{
                display: 'flex', alignItems: 'center', gap: 7, padding: '5px 11px',
                borderRadius: 99, background: acc.color + (T.isDark ? '18' : '12'),
                border: `1px solid ${acc.color}30`,
              }}>
                <div style={{ width: 6, height: 6, borderRadius: '50%', background: acc.color }} />
                <span style={{ fontSize: 12, color: T.text, fontWeight: 600 }}>{acc.name}</span>
                <span style={{ fontSize: 11, color: T.textSec, fontFamily: MONO }}>{formatIDR(acc.balance)}</span>
              </div>
            ))}
          </div>
        </div>
        <div style={{ textAlign: 'right', flexShrink: 0 }}>
          <div style={{ fontSize: 11, color: T.textSec, marginBottom: 10 }}>{monthName}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div>
              <div style={{ fontSize: 10, color: T.income, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em' }}>Masuk</div>
              <div style={{ fontSize: 20, fontWeight: 800, color: T.income, fontFamily: MONO }}>{formatIDR(income)}</div>
            </div>
            <div>
              <div style={{ fontSize: 10, color: T.expense, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em' }}>Keluar</div>
              <div style={{ fontSize: 20, fontWeight: 800, color: T.expense, fontFamily: MONO }}>{formatIDR(expense)}</div>
            </div>
          </div>
        </div>
      </div>

      {/* ROW 2 — Asymmetric: Net + quick stats (2/5) | Investasi (3/5) */}
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 3fr', gap: 16 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {/* Net card */}
          <div style={{ ...card(), padding: '22px 24px', flex: 1 }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 8 }}>Net Bulan Ini</div>
            <div style={{ fontSize: 32, fontWeight: 900, letterSpacing: '-1px', fontFamily: MONO, color: net >= 0 ? T.income : T.expense }}>
              {net >= 0 ? '+' : ''}{formatIDR(net)}
            </div>
            {/* Bar */}
            <div style={{ marginTop: 16, height: 4, borderRadius: 2, background: T.bgInput, overflow: 'hidden', display: 'flex', gap: 2 }}>
              {income > 0 && <div style={{ height: '100%', background: T.income, width: `${(income / (income + expense || 1)) * 100}%`, borderRadius: 2 }} />}
              {expense > 0 && <div style={{ height: '100%', background: T.expense, flex: 1, borderRadius: 2 }} />}
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 6 }}>
              <span style={{ fontSize: 10, color: T.income }}>↑ {income > 0 ? Math.round((income / ((income + expense) || 1)) * 100) : 0}% masuk</span>
              <span style={{ fontSize: 10, color: T.textSec }}>{thisMth.length} trx</span>
            </div>
          </div>
          {/* Quick stats */}
          <div style={{ ...card(), padding: '16px 20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              {[['Rekening', accounts.length], ['Instrumen', investments.length], ['Bulan ini', thisMth.length]].map(([l, v]) => (
                <div key={l} style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: 24, fontWeight: 800, color: T.text, fontFamily: MONO }}>{v}</div>
                  <div style={{ fontSize: 10, color: T.textSec, marginTop: 2 }}>{l}</div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Investasi */}
        <div style={{ ...card(), padding: '22px 24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 18 }}>
            <div>
              <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 4 }}>Saldo Investasi</div>
              <div style={{ fontSize: 13, color: T.textSec }}>{investments.length} instrumen aktif</div>
            </div>
            <button onClick={() => onNav('investasi')} style={{ background: 'none', border: 'none', color: T.accent, cursor: 'pointer', fontSize: 12, padding: 0 }}>Lihat →</button>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {investments.map(inv => {
              const col = inv.type === 'GOLD' ? '#f59e0b' : inv.type === 'CRYPTO' ? '#818cf8' : T.accent;
              const em  = inv.type === 'GOLD' ? '🥇' : inv.type === 'CRYPTO' ? '◈' : '◆';
              return (
                <div key={inv.id} style={{
                  display: 'flex', alignItems: 'center', gap: 12,
                  padding: '11px 14px', borderRadius: T.radiusSm,
                  background: T.bgInput, border: `1px solid ${T.border}`,
                }}>
                  <div style={{ width: 34, height: 34, borderRadius: 8, flexShrink: 0, background: col + '15', display: 'flex', alignItems: 'center', justifyContent: 'center', color: col, fontSize: 15 }}>{em}</div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: T.text }}>{inv.name}</div>
                    <div style={{ fontSize: 10, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.08em' }}>{inv.symbol}</div>
                  </div>
                  <div style={{ fontSize: 17, fontWeight: 800, color: col, fontFamily: MONO }}>{formatQty(inv.quantity, inv.unitLabel)}</div>
                </div>
              );
            })}
          </div>
          <div style={{ marginTop: 10, fontSize: 10, color: T.textDim, borderTop: `1px solid ${T.border}`, paddingTop: 10 }}>
            Nilai real-time dari server saat online
          </div>
        </div>
      </div>

      {/* ROW 3 — Recent transactions grid */}
      <div style={{ ...card(), overflow: 'hidden' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '15px 22px', borderBottom: `1px solid ${T.border}` }}>
          <span style={{ fontSize: 13, fontWeight: 700, color: T.text }}>Transaksi Terkini</span>
          <button onClick={() => onNav('transaksi')} style={{ background: 'none', border: 'none', color: T.accent, cursor: 'pointer', fontSize: 12, padding: 0 }}>Semua →</button>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 0 }}>
          {recent.map((trx, i) => {
            const cat = categories.find(c => c.id === trx.categoryId);
            const isIncome = trx.type === 'INCOME';
            return (
              <div key={trx.id} style={{
                display: 'flex', alignItems: 'center', gap: 12, padding: '13px 22px',
                borderRight: i % 2 === 0 ? `1px solid ${T.border}` : 'none',
                borderBottom: `1px solid ${T.border}`,
                transition: 'background 0.1s',
              }}
                onMouseEnter={e => e.currentTarget.style.background = T.bgHover}
                onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                <div style={{ width: 34, height: 34, borderRadius: 8, flexShrink: 0, fontSize: 14, background: isIncome ? T.income + '15' : T.expense + '15', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  {cat?.icon || (isIncome ? '↑' : '↓')}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 12, fontWeight: 600, color: T.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{trx.note || cat?.name || '—'}</div>
                  <div style={{ fontSize: 10, color: T.textSec }}>{cat?.name} · {formatDateShort(trx.date)}</div>
                </div>
                <div style={{ fontSize: 13, fontWeight: 800, color: isIncome ? T.income : T.expense, fontFamily: MONO, whiteSpace: 'nowrap' }}>
                  {isIncome ? '+' : '−'}{formatIDR(trx.amount)}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { DashboardPage });
