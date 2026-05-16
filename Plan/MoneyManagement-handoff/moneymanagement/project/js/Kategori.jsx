// KasKu v3 — Kategori (theme-aware)

function KategoriPage({ state, onAdd, onEdit, onDelete }) {
  const { categories, transactions } = state;
  const usageCount = (catId) => transactions.filter(t => t.categoryId === catId).length;
  const groups = [
    { label: 'Pemasukan', type: 'INCOME', color: T.income },
    { label: 'Pengeluaran', type: 'EXPENSE', color: T.expense },
    { label: 'Keduanya', type: 'BOTH', color: T.accent },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontSize: 10, color: T.textSec, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 6 }}>Manajemen Kategori</div>
          <div style={{ fontSize: 22, fontWeight: 800, color: T.text }}>{categories.length} <span style={{ fontSize: 14, color: T.textSec, fontWeight: 400 }}>kategori</span></div>
        </div>
        <button onClick={onAdd} style={{ ...getBtnPrimary(), padding: '9px 18px', background: T.grad }}>+ Kategori Baru</button>
      </div>

      {groups.map(group => {
        const cats = categories.filter(c => c.type === group.type);
        if (!cats.length) return null;
        return (
          <div key={group.type}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
              <div style={{ width: 3, height: 16, borderRadius: 2, background: group.color }} />
              <span style={{ fontSize: 11, fontWeight: 700, color: group.color, textTransform: 'uppercase', letterSpacing: '0.1em' }}>{group.label}</span>
              <span style={{ fontSize: 11, color: T.textSec }}>{cats.length} kategori</span>
              <div style={{ flex: 1, height: 1, background: T.border }} />
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 8 }}>
              {cats.map(cat => {
                const count = usageCount(cat.id);
                return (
                  <div key={cat.id} style={{
                    ...card(), padding: '13px 16px',
                    display: 'flex', alignItems: 'center', gap: 12,
                    transition: 'all 0.12s',
                  }}
                    onMouseEnter={e => { e.currentTarget.style.borderColor = group.color + '30'; e.currentTarget.style.background = group.color + '06'; }}
                    onMouseLeave={e => { e.currentTarget.style.borderColor = T.border; e.currentTarget.style.background = T.bgCard; }}>
                    <div style={{
                      width: 36, height: 36, borderRadius: 8, flexShrink: 0,
                      background: group.color + '12', border: `1px solid ${group.color}20`,
                      display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18,
                    }}>{cat.icon || '📌'}</div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 13, fontWeight: 700, color: T.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{cat.name}</div>
                      <div style={{ fontSize: 10, color: T.textSec, marginTop: 2 }}>{count} trx{cat.isDefault ? ' · default' : ''}</div>
                    </div>
                    <div style={{ display: 'flex', gap: 3, flexShrink: 0 }}>
                      <button onClick={() => onEdit(cat)} style={{ width: 26, height: 26, borderRadius: 5, border: `1px solid ${T.border}`, background: T.bgInput, cursor: 'pointer', color: T.textSec, fontSize: 11, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✏</button>
                      {!cat.isDefault && count === 0 && (
                        <button onClick={() => onDelete(cat.id)} style={{ width: 26, height: 26, borderRadius: 5, border: `1px solid ${T.expense}25`, background: T.expense + '08', cursor: 'pointer', color: T.expense, fontSize: 11, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

Object.assign(window, { KategoriPage });
