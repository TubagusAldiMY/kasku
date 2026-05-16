// KasKu v3 — Modals (theme-aware, bCancel/bSubmit as inline)

function Overlay({ onClose, children }) {
  React.useEffect(() => {
    const h = e => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', h);
    return () => window.removeEventListener('keydown', h);
  }, [onClose]);
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 1000,
      background: T.isDark ? 'rgba(3,8,18,0.88)' : 'rgba(0,10,40,0.45)',
      backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 16,
    }} onClick={e => e.target === e.currentTarget && onClose()}>
      {children}
    </div>
  );
}

function ModalBox({ title, sub, onClose, children }) {
  return (
    <Overlay onClose={onClose}>
      <div style={{
        ...card(), width: '100%', maxWidth: 450,
        background: T.bgSurf, maxHeight: '90vh', overflowY: 'auto',
        borderColor: T.borderMd,
        boxShadow: T.isDark ? '0 24px 80px rgba(0,0,0,0.6)' : '0 8px 40px rgba(0,0,80,0.15)',
      }}>
        <div style={{
          display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
          padding: '20px 24px', borderBottom: `1px solid ${T.border}`,
        }}>
          <div>
            <div style={{ fontSize: 16, fontWeight: 800, color: T.text }}>{title}</div>
            {sub && <div style={{ fontSize: 11, color: T.textSec, marginTop: 2 }}>{sub}</div>}
          </div>
          <button onClick={onClose} style={{
            width: 28, height: 28, borderRadius: 6, border: `1px solid ${T.border}`,
            background: T.bgInput, cursor: 'pointer', color: T.textSec, fontSize: 16,
            display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, marginLeft: 12,
          }}>×</button>
        </div>
        <div style={{ padding: '20px 24px' }}>{children}</div>
      </div>
    </Overlay>
  );
}

function F({ label, children, hint }) {
  return (
    <div style={{ marginBottom: 16 }}>
      <label style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: 7 }}>
        {label}
      </label>
      {children}
      {hint && <div style={{ fontSize: 10, color: T.textSec, marginTop: 5 }}>{hint}</div>}
    </div>
  );
}

function Err({ msg }) {
  return msg ? (
    <div style={{ fontSize: 12, color: T.expense, marginBottom: 14, padding: '8px 12px', borderRadius: 8, background: T.expense + '10', border: `1px solid ${T.expense}20` }}>
      ⚠ {msg}
    </div>
  ) : null;
}

function BtnsRow({ onClose, onSubmit, submitLabel }) {
  return (
    <div style={{ display: 'flex', gap: 8 }}>
      <button onClick={onClose} style={{
        flex: 1, padding: '11px', borderRadius: T.radiusSm,
        border: `1px solid ${T.border}`, background: 'transparent',
        color: T.textSec, cursor: 'pointer', fontSize: 14, fontFamily: 'inherit',
      }}>Batal</button>
      <button onClick={onSubmit} style={{
        flex: 2, padding: '11px', borderRadius: T.radiusSm,
        border: 'none', cursor: 'pointer', background: T.grad,
        color: '#fff', fontSize: 14, fontWeight: 800, fontFamily: 'inherit',
      }}>{submitLabel}</button>
    </div>
  );
}

// ─── Transaction Modal ────────────────────────────────────────────────────────
function TransactionModal({ state, onSave, onClose, editItem }) {
  const { categories, accounts } = state;
  const [form, setForm] = React.useState(editItem
    ? { type: editItem.type, amount: String(editItem.amount), categoryId: editItem.categoryId, accountId: editItem.accountId || '', date: editItem.date, note: editItem.note || '' }
    : { type: 'EXPENSE', amount: '', categoryId: '', accountId: '', date: today(), note: '' });
  const [err, setErr] = React.useState('');
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));
  const inp = getINP();
  const visibleCats = categories.filter(c => c.type === form.type || c.type === 'BOTH');

  const submit = () => {
    if (!form.amount || parseFloat(form.amount) <= 0) return setErr('Nominal harus lebih dari 0');
    if (!form.categoryId) return setErr('Pilih kategori');
    if (!form.date) return setErr('Tanggal wajib diisi');
    onSave({ ...form, amount: parseFloat(form.amount) });
  };

  const tBtn = (t, label, color) => ({
    flex: 1, padding: '10px', borderRadius: T.radiusSm, cursor: 'pointer',
    fontSize: 13, fontWeight: 700, fontFamily: 'inherit',
    background: form.type === t ? color : T.bgInput,
    color: form.type === t ? '#ffffff' : T.textSec,
    border: `1px solid ${form.type === t ? color : T.border}`,
    transition: 'all 0.12s',
  });

  return (
    <ModalBox title={editItem ? 'Edit Transaksi' : 'Catat Transaksi'} sub="Masukkan detail transaksi" onClose={onClose}>
      <F label="Tipe">
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={tBtn('INCOME', '↑ Pemasukan', T.income)} onClick={() => { set('type', 'INCOME'); set('categoryId', ''); }}>↑ Pemasukan</button>
          <button style={tBtn('EXPENSE', '↓ Pengeluaran', T.expense)} onClick={() => { set('type', 'EXPENSE'); set('categoryId', ''); }}>↓ Pengeluaran</button>
        </div>
      </F>
      <F label="Nominal (Rp)">
        <input style={inp} type="number" min="1" step="500" placeholder="0" value={form.amount}
          onChange={e => set('amount', e.target.value)} autoFocus />
        {form.amount && parseFloat(form.amount) > 0 && (
          <div style={{ fontSize: 13, fontWeight: 700, color: form.type === 'INCOME' ? T.income : T.expense, marginTop: 6, fontFamily: MONO }}>
            {formatIDR(parseFloat(form.amount))}
          </div>
        )}
      </F>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <F label="Kategori">
          <select style={inp} value={form.categoryId} onChange={e => set('categoryId', e.target.value)}>
            <option value="">Pilih…</option>
            {visibleCats.map(c => <option key={c.id} value={c.id}>{c.icon} {c.name}</option>)}
          </select>
        </F>
        <F label="Tanggal">
          <input style={inp} type="date" value={form.date} onChange={e => set('date', e.target.value)} />
        </F>
      </div>
      <F label="Akun (opsional)">
        <select style={inp} value={form.accountId} onChange={e => set('accountId', e.target.value)}>
          <option value="">— Tanpa akun —</option>
          {accounts.map(a => <option key={a.id} value={a.id}>{a.icon} {a.name}</option>)}
        </select>
      </F>
      <F label="Catatan">
        <textarea style={{ ...inp, resize: 'vertical', minHeight: 68 }} placeholder="Opsional…" value={form.note}
          onChange={e => set('note', e.target.value)} maxLength={500} />
      </F>
      <Err msg={err} />
      <BtnsRow onClose={onClose} onSubmit={submit} submitLabel={editItem ? 'Simpan' : 'Catat Transaksi'} />
    </ModalBox>
  );
}

// ─── Account Modal ────────────────────────────────────────────────────────────
function AccountModal({ state, onSave, onClose, editItem }) {
  const ICONS = { BANK: '🏦', EWALLET: '💳', CASH: '💵' };
  const COLORS = ['#4285f4','#9b5de5','#f59e0b','#1ed9a0','#f05a6e','#38bdf8','#fb923c','#34d399'];
  const [form, setForm] = React.useState(editItem
    ? { name: editItem.name, type: editItem.type, balance: String(editItem.balance), color: editItem.color, icon: editItem.icon }
    : { name: '', type: 'BANK', balance: '0', color: '#4285f4', icon: '🏦' });
  const [err, setErr] = React.useState('');
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));
  const inp = getINP();
  const submit = () => { if (!form.name.trim()) return setErr('Nama akun wajib diisi'); onSave({ ...form, balance: parseFloat(form.balance) || 0 }); };

  return (
    <ModalBox title={editItem ? 'Edit Akun' : 'Tambah Akun'} onClose={onClose}>
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 12 }}>
        <F label="Nama Akun"><input style={inp} type="text" placeholder="BCA, Seabank…" value={form.name} onChange={e => set('name', e.target.value)} autoFocus maxLength={100} /></F>
        <F label="Tipe">
          <select style={inp} value={form.type} onChange={e => { set('type', e.target.value); set('icon', ICONS[e.target.value] || '💰'); }}>
            <option value="BANK">Bank</option><option value="EWALLET">E-Wallet</option><option value="CASH">Tunai</option>
          </select>
        </F>
      </div>
      <F label="Saldo (Rp)">
        <input style={inp} type="number" min="0" step="1000" placeholder="0" value={form.balance} onChange={e => set('balance', e.target.value)} />
        {form.balance && <div style={{ fontSize: 12, color: T.accent, marginTop: 5, fontFamily: MONO }}>{formatIDR(parseFloat(form.balance) || 0)}</div>}
      </F>
      <F label="Warna">
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {COLORS.map(c => (
            <div key={c} onClick={() => set('color', c)} style={{ width: 28, height: 28, borderRadius: 7, background: c, cursor: 'pointer', outline: form.color === c ? '2px solid white' : '2px solid transparent', outlineOffset: 2, transition: 'all 0.12s' }} />
          ))}
        </div>
      </F>
      <Err msg={err} />
      <BtnsRow onClose={onClose} onSubmit={submit} submitLabel={editItem ? 'Simpan' : 'Tambah Akun'} />
    </ModalBox>
  );
}

// ─── Investment Modal ─────────────────────────────────────────────────────────
function InvestmentModal({ state, onSave, onClose, editItem }) {
  const TYPE_DEF = { GOLD:{symbol:'XAU',unitLabel:'gram',coinId:''}, CRYPTO:{symbol:'BTC',unitLabel:'BTC',coinId:'bitcoin'}, STOCK:{symbol:'BBCA',unitLabel:'lot',coinId:''}, OTHER:{symbol:'',unitLabel:'unit',coinId:''} };
  const [form, setForm] = React.useState(editItem
    ? { name:editItem.name, type:editItem.type, symbol:editItem.symbol, coinId:editItem.coinId||'', quantity:String(editItem.quantity), unitLabel:editItem.unitLabel }
    : { name:'', type:'GOLD', symbol:'XAU', coinId:'', quantity:'0', unitLabel:'gram' });
  const [err, setErr] = React.useState('');
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));
  const inp = getINP();
  const submit = () => { if (!form.name.trim()) return setErr('Nama wajib diisi'); if (!form.symbol.trim()) return setErr('Simbol wajib diisi'); onSave({ ...form, quantity: parseFloat(form.quantity)||0 }); };

  return (
    <ModalBox title={editItem ? 'Edit Investasi' : 'Tambah Investasi'} onClose={onClose}>
      <F label="Tipe">
        <select style={inp} value={form.type} onChange={e => setForm(f => ({ ...f, type:e.target.value, ...TYPE_DEF[e.target.value] }))}>
          <option value="GOLD">🥇 Emas</option><option value="CRYPTO">🪙 Kripto</option><option value="STOCK">📊 Saham</option><option value="OTHER">💼 Lainnya</option>
        </select>
      </F>
      <F label="Nama"><input style={inp} type="text" placeholder="Emas Antam, Bitcoin…" value={form.name} onChange={e => set('name', e.target.value)} autoFocus maxLength={100} /></F>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
        <F label="Simbol"><input style={inp} type="text" value={form.symbol} onChange={e => set('symbol', e.target.value.toUpperCase())} maxLength={20} /></F>
        <F label="Satuan"><input style={inp} type="text" value={form.unitLabel} onChange={e => set('unitLabel', e.target.value)} maxLength={20} /></F>
        <F label="Jumlah"><input style={inp} type="number" min="0" step="any" value={form.quantity} onChange={e => set('quantity', e.target.value)} /></F>
      </div>
      {form.quantity && parseFloat(form.quantity) > 0 && <div style={{ fontSize: 12, color: '#818cf8', marginBottom: 14, fontFamily: MONO }}>{formatQty(parseFloat(form.quantity)||0, form.unitLabel)}</div>}
      {form.type === 'CRYPTO' && <F label="CoinGecko ID" hint="Untuk harga real-time. Contoh: bitcoin, ethereum"><input style={inp} type="text" placeholder="bitcoin, ethereum…" value={form.coinId} onChange={e => set('coinId', e.target.value.toLowerCase())} maxLength={50} /></F>}
      <Err msg={err} />
      <BtnsRow onClose={onClose} onSubmit={submit} submitLabel={editItem ? 'Simpan' : 'Tambah'} />
    </ModalBox>
  );
}

// ─── Category Modal ───────────────────────────────────────────────────────────
function CategoryModal({ state, onSave, onClose, editItem }) {
  const [form, setForm] = React.useState(editItem ? { name:editItem.name, type:editItem.type, icon:editItem.icon||'' } : { name:'', type:'EXPENSE', icon:'' });
  const [err, setErr] = React.useState('');
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));
  const inp = getINP();
  const ICONS = ['🍜','🚗','🛍️','⚡','🎮','💊','📚','📌','💼','💻','🎁','📈','🏠','✈️','🎓','💇','🐶','🎵','⚽','🍕','☕','🧾','💐','🧴'];
  const submit = () => { if (!form.name.trim()) return setErr('Nama wajib diisi'); onSave(form); };

  return (
    <ModalBox title={editItem ? 'Edit Kategori' : 'Tambah Kategori'} onClose={onClose}>
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 12 }}>
        <F label="Nama"><input style={inp} type="text" placeholder="Nama kategori…" value={form.name} onChange={e => set('name', e.target.value)} autoFocus maxLength={100} /></F>
        <F label="Tipe">
          <select style={inp} value={form.type} onChange={e => set('type', e.target.value)}>
            <option value="INCOME">↑ Masuk</option><option value="EXPENSE">↓ Keluar</option><option value="BOTH">Keduanya</option>
          </select>
        </F>
      </div>
      <F label="Ikon">
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
          {ICONS.map(icon => (
            <div key={icon} onClick={() => set('icon', icon)} style={{
              width: 36, height: 36, borderRadius: 8, cursor: 'pointer', fontSize: 17,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: form.icon === icon ? T.accentDim : T.bgInput,
              border: `1px solid ${form.icon === icon ? T.borderHi : T.border}`,
              transition: 'all 0.1s',
            }}>{icon}</div>
          ))}
        </div>
      </F>
      <Err msg={err} />
      <BtnsRow onClose={onClose} onSubmit={submit} submitLabel={editItem ? 'Simpan' : 'Tambah'} />
    </ModalBox>
  );
}

// ─── Confirm ──────────────────────────────────────────────────────────────────
function ConfirmModal({ message, onConfirm, onClose }) {
  return (
    <Overlay onClose={onClose}>
      <div style={{ ...card(), width: '100%', maxWidth: 360, background: T.bgSurf, borderColor: T.borderMd, padding: 24, boxShadow: T.isDark ? '0 24px 80px rgba(0,0,0,0.6)' : '0 8px 40px rgba(0,0,80,0.15)' }}>
        <div style={{ fontSize: 15, fontWeight: 800, color: T.text, marginBottom: 8 }}>Konfirmasi Hapus</div>
        <div style={{ fontSize: 13, color: T.textSec, marginBottom: 24, lineHeight: 1.6 }}>{message}</div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: T.radiusSm, border: `1px solid ${T.border}`, background: 'transparent', color: T.textSec, cursor: 'pointer', fontSize: 14, fontFamily: 'inherit' }}>Batal</button>
          <button onClick={onConfirm} style={{ flex: 1, padding: '10px', borderRadius: T.radiusSm, border: 'none', cursor: 'pointer', background: T.expense, color: '#fff', fontSize: 14, fontWeight: 800, fontFamily: 'inherit' }}>Hapus</button>
        </div>
      </div>
    </Overlay>
  );
}

Object.assign(window, { TransactionModal, AccountModal, InvestmentModal, CategoryModal, ConfirmModal });
