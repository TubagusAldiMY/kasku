// KasKu v3 — Transaksi (theme-aware, INP as function)

function TransaksiPage({ state, onAdd, onEdit, onDelete }) {
  const { transactions, categories, accounts } = state;
  const [filter, setFilter] = React.useState({ type: 'ALL', categoryId: '', from: '', to: '', search: '' });
  const [pg, setPg] = React.useState(1);
  const PER = 25;

  const filtered = transactions.filter(t => {
    if (filter.type !== 'ALL' && t.type !== filter.type) return false;
    if (filter.categoryId && t.categoryId !== filter.categoryId) return false;
    if (filter.from && t.date < filter.from) return false;
    if (filter.to && t.date > filter.to) return false;
    if (filter.search) {
      const q = filter.search.toLowerCase();
      const cat = categories.find(c => c.id === t.categoryId);
      if (!(t.note || '').toLowerCase().includes(q) && !(cat?.name || '').toLowerCase().includes(q)) return false;
    }
    return true;
  }).sort((a, b) => b.date.localeCompare(a.date) || b.id.localeCompare(a.id));

  const income  = filtered.filter(t => t.type === 'INCOME').reduce((s, t) => s + t.amount, 0);
  const expense = filtered.filter(t => t.type === 'EXPENSE').reduce((s, t) => s + t.amount, 0);
  const paged = filtered.slice((pg - 1) * PER, pg * PER);
  const totalPages = Math.ceil(filtered.length / PER);
  const setF = (k, v) => { setFilter(f => ({ ...f, [k]: v })); setPg(1); };
  const hasFilter = filter.type !== 'ALL' || filter.categoryId || filter.from || filter.to || filter.search;

  const grouped = [];
  let lastDate = null;
  paged.forEach(t => {
    if (t.date !== lastDate) { grouped.push({ isDate: true, date: t.date }); lastDate = t.date; }
    grouped.push({ isDate: false, trx: t });
  });

  const fBtn = (active) => ({
    padding: '7px 14px', borderRadius: 6, cursor: 'pointer',
    fontSize: 12, fontWeight: 700, fontFamily: 'inherit',
    background: active ? T.accent : T.bgInput,
    color: active ? '#ffffff' : T.textSec,
    border: active ? 'none' : `1px solid ${T.border}`,
    transition: 'all 0.12s',
  });

  const inp = getINP();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>

      {/* Stat strip */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 10 }}>
        {[
          { label: 'Pemasukan', val: formatIDR(income), color: T.income },
          { label: 'Pengeluaran', val: formatIDR(expense), color: T.expense },
          { label: 'Net', val: (income - expense >= 0 ? '+' : '') + formatIDR(income - expense), color: income - expense >= 0 ? T.income : T.expense },
          { label: 'Transaksi', val: filtered.length, color: T.text },
        ].map(s => (
          <div key={s.label} style={{ ...card(), padding: '14px 16px' }}>
            <div style={{ fontSize: 10, color: T.textSec, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 5 }}>{s.label}</div>
            <div style={{ fontSize: 18, fontWeight: 800, color: s.color, fontFamily: MONO }}>{s.val}</div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div style={{ ...card(), padding: '14px 18px' }}>
        <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap', alignItems: 'center' }}>
          {[['ALL','Semua'], ['INCOME','↑ Masuk'], ['EXPENSE','↓ Keluar']].map(([v,l]) => (
            <button key={v} style={fBtn(filter.type === v)} onClick={() => setF('type', v)}>{l}</button>
          ))}
          <div style={{ flex: 1 }} />
          {hasFilter && (
            <button onClick={() => { setFilter({ type:'ALL', categoryId:'', from:'', to:'', search:'' }); setPg(1); }}
              style={{ background:'none', border:'none', color: T.expense, cursor:'pointer', fontSize:12, padding:0 }}>
              ✕ Reset
            </button>
          )}
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 2fr', gap: 8 }}>
          <input style={inp} type="date" value={filter.from} onChange={e => setF('from', e.target.value)} />
          <input style={inp} type="date" value={filter.to} onChange={e => setF('to', e.target.value)} />
          <select style={inp} value={filter.categoryId} onChange={e => setF('categoryId', e.target.value)}>
            <option value="">Semua kategori</option>
            {categories.map(c => <option key={c.id} value={c.id}>{c.icon} {c.name}</option>)}
          </select>
          <input style={inp} type="text" placeholder="Cari catatan atau kategori…" value={filter.search}
            onChange={e => setF('search', e.target.value)} />
        </div>
      </div>

      {/* List */}
      <div style={{ ...card(), overflow: 'hidden' }}>
        {grouped.length === 0 ? (
          <div style={{ padding: 48, textAlign: 'center', color: T.textSec }}>
            <div style={{ fontSize: 32, marginBottom: 8, opacity: 0.4 }}>≡</div>
            <div style={{ fontSize: 14 }}>Tidak ada transaksi</div>
          </div>
        ) : grouped.map((item, i) => {
          if (item.isDate) {
            const dayTotal = paged.filter(t => t.date === item.date).reduce((s, t) => s + (t.type === 'INCOME' ? t.amount : -t.amount), 0);
            return (
              <div key={'d' + item.date} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '7px 20px', background: T.bgInput, borderBottom: `1px solid ${T.border}`,
              }}>
                <span style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.1em' }}>{formatDate(item.date)}</span>
                <span style={{ fontSize: 11, fontWeight: 700, color: dayTotal >= 0 ? T.income : T.expense, fontFamily: MONO }}>
                  {dayTotal >= 0 ? '+' : ''}{formatIDR(dayTotal)}
                </span>
              </div>
            );
          }
          const { trx } = item;
          const cat = categories.find(c => c.id === trx.categoryId);
          const acc = accounts.find(a => a.id === trx.accountId);
          const isIncome = trx.type === 'INCOME';
          return (
            <div key={trx.id} style={{
              display: 'grid', gridTemplateColumns: 'auto 1fr auto auto auto',
              alignItems: 'center', gap: 14, padding: '12px 20px',
              borderBottom: `1px solid ${T.border}`, transition: 'background 0.1s',
            }}
              onMouseEnter={e => e.currentTarget.style.background = T.bgHover}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
              <div style={{ width: 34, height: 34, borderRadius: 8, flexShrink: 0, background: isIncome ? T.income + '15' : T.expense + '15', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 }}>
                {cat?.icon || (isIncome ? '↑' : '↓')}
              </div>
              <div style={{ minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: T.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {trx.note || cat?.name || '—'}
                </div>
                <div style={{ fontSize: 10, color: T.textSec, display: 'flex', gap: 6 }}>
                  <span>{cat?.name}</span>
                  {acc && <><span>·</span><span style={{ color: acc.color }}>{acc.name}</span></>}
                </div>
              </div>
              <div style={{ fontSize: 10, padding: '2px 8px', borderRadius: 99, fontWeight: 700, whiteSpace: 'nowrap', background: isIncome ? T.income + '15' : T.expense + '15', color: isIncome ? T.income : T.expense }}>
                {isIncome ? '↑ Masuk' : '↓ Keluar'}
              </div>
              <div style={{ fontSize: 14, fontWeight: 800, color: isIncome ? T.income : T.expense, fontFamily: MONO, textAlign: 'right', whiteSpace: 'nowrap' }}>
                {isIncome ? '+' : '−'}{formatIDR(trx.amount)}
              </div>
              <div style={{ display: 'flex', gap: 4 }}>
                <button onClick={() => onEdit(trx)} style={{ width: 28, height: 28, borderRadius: 6, border: `1px solid ${T.border}`, background: T.bgInput, cursor: 'pointer', color: T.textSec, fontSize: 11, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✏</button>
                <button onClick={() => onDelete(trx.id)} style={{ width: 28, height: 28, borderRadius: 6, border: `1px solid ${T.expense}25`, background: T.expense + '08', cursor: 'pointer', color: T.expense, fontSize: 11, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', justifyContent: 'center', gap: 6, alignItems: 'center' }}>
          <button onClick={() => setPg(p => Math.max(1, p - 1))} disabled={pg === 1}
            style={{ padding: '7px 12px', borderRadius: 6, border: `1px solid ${T.border}`, background: T.bgCard, color: pg === 1 ? T.textDim : T.text, cursor: pg === 1 ? 'default' : 'pointer', fontSize: 13, fontFamily: 'inherit' }}>←</button>
          <span style={{ fontSize: 12, color: T.textSec }}>hal {pg} / {totalPages}</span>
          <button onClick={() => setPg(p => Math.min(totalPages, p + 1))} disabled={pg === totalPages}
            style={{ padding: '7px 12px', borderRadius: 6, border: `1px solid ${T.border}`, background: T.bgCard, color: pg === totalPages ? T.textDim : T.text, cursor: pg === totalPages ? 'default' : 'pointer', fontSize: 13, fontFamily: 'inherit' }}>→</button>
        </div>
      )}
    </div>
  );
}

Object.assign(window, { TransaksiPage });
