// KasKu v3 — App Root (with working theme toggle)

function LoginPage({ onLogin, theme }) {
  const [user, setUser] = React.useState('admin');
  const [pass, setPass] = React.useState('');
  const [err, setErr] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const inp = getINP();

  const submit = () => {
    if (!user.trim() || pass.length < 4) return setErr('Username wajib diisi, password minimal 4 karakter');
    setLoading(true);
    setTimeout(() => {setLoading(false);onLogin(user);}, 800);
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', background: T.bgBase }}>
      {/* Left — brand panel */}
      <div style={{
        flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center',
        padding: '60px 72px',
        background: T.bgLoginLeft,
        borderRight: `1px solid ${T.border}`
      }}>
        <div style={{ fontSize: 52, fontWeight: 900, letterSpacing: '-2px', marginBottom: 10 }}>
          <span style={{ color: T.text }}>Kas</span>
          <span style={{ ...gLogoText(), display: 'inline-block', color: "rgb(14, 76, 184)" }}>Ku</span>
        </div>
        <div style={{ fontSize: 15, color: T.textSec, lineHeight: 1.8, maxWidth: 340, marginBottom: 48 }}>
          Catatan keuangan pribadi yang sederhana, privat, dan bekerja offline.
        </div>
        {[
        { icon: '◎', label: 'Saldo Keuangan', sub: 'Kelola semua rekening dalam satu tempat' },
        { icon: '◈', label: 'Saldo Investasi', sub: 'Pantau portofolio per satuan unit' },
        { icon: '≡', label: 'Buku Kas', sub: 'Catat transaksi sekecil apapun' }].
        map((f) =>
        <div key={f.label} style={{ display: 'flex', alignItems: 'flex-start', gap: 14, marginBottom: 18 }}>
            <div style={{
            width: 32, height: 32, borderRadius: 8, background: T.accentDim,
            border: `1px solid ${T.borderHi}`, display: 'flex', alignItems: 'center',
            justifyContent: 'center', color: T.accent, fontSize: 14, flexShrink: 0
          }}>{f.icon}</div>
            <div>
              <div style={{ fontSize: 13, fontWeight: 700, color: T.text }}>{f.label}</div>
              <div style={{ fontSize: 12, color: T.textSec }}>{f.sub}</div>
            </div>
          </div>
        )}
      </div>

      {/* Right — form */}
      <div style={{ width: 420, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '40px 52px', flexShrink: 0, background: T.bgSurf }}>
        <div style={{ width: '100%' }}>
          <div style={{ fontSize: 22, fontWeight: 800, color: T.text, marginBottom: 4 }}>Masuk</div>
          <div style={{ fontSize: 13, color: T.textSec, marginBottom: 32 }}>Akses dashboard keuangan Anda</div>
          <div style={{ marginBottom: 14 }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 7 }}>Username</div>
            <input style={inp} type="text" value={user} onChange={(e) => setUser(e.target.value)} placeholder="Username" />
          </div>
          <div style={{ marginBottom: 24 }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 7 }}>Password</div>
            <input style={inp} type="password" value={pass} onChange={(e) => setPass(e.target.value)} placeholder="••••••••" onKeyDown={(e) => e.key === 'Enter' && submit()} />
          </div>
          {err && <div style={{ fontSize: 12, color: T.expense, marginBottom: 16, padding: '8px 12px', borderRadius: 8, background: T.expense + '10', border: `1px solid ${T.expense}20` }}>⚠ {err}</div>}
          <button onClick={submit} disabled={loading} style={{
            width: '100%', padding: '12px', borderRadius: T.radiusSm, border: 'none',
            background: loading ? T.bgInput : T.grad,
            color: loading ? T.textSec : '#fff',
            fontSize: 14, fontWeight: 800, cursor: loading ? 'wait' : 'pointer', fontFamily: 'inherit'
          }}>{loading ? 'Memverifikasi…' : 'Masuk →'}</button>
          <div style={{ marginTop: 16, fontSize: 11, color: T.textDim, textAlign: 'center' }}>Password minimal 4 karakter</div>
        </div>
      </div>
    </div>);

}

function FAB({ onClick }) {
  const [hov, setHov] = React.useState(false);
  return (
    <button onClick={onClick} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
    style={{
      position: 'fixed', bottom: 72, right: 18, zIndex: 200,
      width: 52, height: 52, borderRadius: '50%', border: 'none', cursor: 'pointer',
      background: T.grad, color: '#fff', fontSize: 26, fontWeight: 900,
      boxShadow: `0 4px 20px ${T.accent}40`,
      transform: hov ? 'scale(1.08)' : 'scale(1)',
      transition: 'transform 0.12s', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'inherit'
    }}>+</button>);

}

function TweaksPanel({ show, tweaks, onChange }) {
  if (!show) return null;
  const accents = [
  { label: 'Gemini', val: '#4285f4' },
  { label: 'Indigo', val: '#818cf8' },
  { label: 'Teal', val: '#14b8a6' },
  { label: 'Amber', val: '#f59e0b' }];

  return (
    <div style={{ position: 'fixed', bottom: 20, right: 20, zIndex: 500, width: 240, ...card(), background: T.bgSurf, borderColor: T.borderMd, padding: 18, boxShadow: '0 8px 40px rgba(0,0,0,0.2)' }}>
      <div style={{ fontSize: 11, fontWeight: 800, color: T.text, marginBottom: 16, textTransform: 'uppercase', letterSpacing: '0.1em' }}>Tweaks</div>
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 10 }}>Warna Aksen</div>
        <div style={{ display: 'flex', gap: 8 }}>
          {accents.map((a) =>
          <div key={a.val} onClick={() => onChange('accent', a.val)} title={a.label} style={{
            flex: 1, height: 28, borderRadius: 7, background: a.val, cursor: 'pointer',
            outline: tweaks.accent === a.val ? '2px solid white' : '2px solid transparent',
            outlineOffset: 2
          }} />
          )}
        </div>
      </div>
      <div>
        <div style={{ fontSize: 10, fontWeight: 700, color: T.textSec, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 10 }}>Font Angka</div>
        <div style={{ display: 'flex', gap: 6 }}>
          {[['mono', 'Mono'], ['sans', 'Sans']].map(([v, l]) =>
          <button key={v} onClick={() => onChange('numFont', v)} style={{
            flex: 1, padding: '7px', borderRadius: 6, border: 'none', cursor: 'pointer', fontSize: 11, fontWeight: 700,
            background: tweaks.numFont === v ? T.accent : T.bgInput,
            color: tweaks.numFont === v ? '#fff' : T.textSec, fontFamily: v === 'mono' ? MONO : 'inherit'
          }}>{l}</button>
          )}
        </div>
      </div>
    </div>);

}

// ─── App ──────────────────────────────────────────────────────────────────────
function App() {
  const [loggedIn, setLoggedIn] = React.useState(() => !!localStorage.getItem('kasku_session'));
  const [data, setData] = React.useState(() => KasKuStore.load());
  const [page, setPage] = React.useState(() => localStorage.getItem('kasku_page') || 'dashboard');
  const [theme, setTheme] = React.useState(() => localStorage.getItem('kasku_theme') || 'dark');
  const [modal, setModal] = React.useState(null);
  const [confirm, setConfirm] = React.useState(null);
  const [tweaksVisible, setTweaksVisible] = React.useState(false);
  const [syncStatus, setSyncStatus] = React.useState('synced');
  const [tweaks, setTweaks] = React.useState({ accent: '#4285f4', numFont: 'mono' });

  // Apply theme on mount using persisted value
  React.useEffect(() => {applyTheme(theme);document.body.style.background = theme === 'light' ? LIGHT_THEME.bgBase : DARK_THEME.bgBase;}, []); // eslint-disable-line

  React.useEffect(() => {
    window.addEventListener('message', (e) => {
      if (e.data?.type === '__activate_edit_mode') setTweaksVisible(true);
      if (e.data?.type === '__deactivate_edit_mode') setTweaksVisible(false);
    });
    window.parent.postMessage({ type: '__edit_mode_available' }, '*');
  }, []);

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    applyTheme(next); // mutate T first
    localStorage.setItem('kasku_theme', next);
    setTheme(next); // triggers re-render; key change forces full remount
  };

  const chgTweaks = (k, v) => {
    setTweaks((t) => ({ ...t, [k]: v }));
    window.parent.postMessage({ type: '__edit_mode_set_keys', edits: { [k]: v } }, '*');
  };

  const nav = (p) => {setPage(p);localStorage.setItem('kasku_page', p);};
  const update = (key, value) => {setData((d) => {const nd = { ...d, [key]: value };KasKuStore.save(key, value);return nd;});};
  const triggerSync = () => {setSyncStatus('pending');setTimeout(() => {setSyncStatus('syncing');setTimeout(() => setSyncStatus('synced'), 1100);}, 200);};

  const saveTrx = (form, edit) => {const arr = [...data.transactions];if (edit) arr[arr.findIndex((t) => t.id === edit.id)] = { ...edit, ...form };else arr.unshift({ id: KasKuStore.genId('trx'), ...form });update('transactions', arr);setModal(null);triggerSync();};
  const delTrx = (id) => setConfirm({ message: 'Hapus transaksi ini?', onConfirm: () => {update('transactions', data.transactions.filter((t) => t.id !== id));setConfirm(null);triggerSync();} });
  const saveAcc = (form, edit) => {const arr = [...data.accounts];if (edit) arr[arr.findIndex((a) => a.id === edit.id)] = { ...edit, ...form };else arr.push({ id: KasKuStore.genId('acc'), ...form });update('accounts', arr);setModal(null);triggerSync();};
  const delAcc = (id) => setConfirm({ message: 'Hapus akun ini?', onConfirm: () => {update('accounts', data.accounts.filter((a) => a.id !== id));setConfirm(null);triggerSync();} });
  const saveInv = (form, edit) => {const arr = [...data.investments];if (edit) arr[arr.findIndex((a) => a.id === edit.id)] = { ...edit, ...form };else arr.push({ id: KasKuStore.genId('inv'), ...form });update('investments', arr);setModal(null);triggerSync();};
  const delInv = (id) => setConfirm({ message: 'Hapus instrumen investasi ini?', onConfirm: () => {update('investments', data.investments.filter((i) => i.id !== id));setConfirm(null);triggerSync();} });
  const saveCat = (form, edit) => {const arr = [...data.categories];if (edit) arr[arr.findIndex((c) => c.id === edit.id)] = { ...edit, ...form };else arr.push({ id: KasKuStore.genId('cat'), ...form, isDefault: false });update('categories', arr);setModal(null);};
  const delCat = (id) => setConfirm({ message: 'Hapus kategori ini?', onConfirm: () => {update('categories', data.categories.filter((c) => c.id !== id));setConfirm(null);} });

  const PAGE_TITLES = { dashboard: 'Dashboard', keuangan: 'Saldo Keuangan', investasi: 'Saldo Investasi', transaksi: 'Transaksi', kategori: 'Kategori' };
  const state = { ...data, theme, syncStatus };

  if (!loggedIn) return (
    <div key={theme}>
      <LoginPage theme={theme} onLogin={(u) => {localStorage.setItem('kasku_session', u);setLoggedIn(true);}} />
    </div>);


  return (
    <div key={theme}>
      <MainLayout page={page} onNav={nav} theme={theme} syncStatus={syncStatus}
      onToggleTheme={toggleTheme} onAddTrx={() => setModal({ type: 'transaction' })} pageTitle={PAGE_TITLES[page]}>
        {page === 'dashboard' && <DashboardPage state={state} onNav={nav} />}
        {page === 'keuangan' && <KeuanganPage state={state} onAdd={() => setModal({ type: 'account' })} onEdit={(i) => setModal({ type: 'account', item: i })} onDelete={delAcc} />}
        {page === 'investasi' && <InvestasiPage state={state} onAdd={() => setModal({ type: 'investment' })} onEdit={(i) => setModal({ type: 'investment', item: i })} onDelete={delInv} />}
        {page === 'transaksi' && <TransaksiPage state={state} onAdd={() => setModal({ type: 'transaction' })} onEdit={(i) => setModal({ type: 'transaction', item: i })} onDelete={delTrx} />}
        {page === 'kategori' && <KategoriPage state={state} onAdd={() => setModal({ type: 'category' })} onEdit={(i) => setModal({ type: 'category', item: i })} onDelete={delCat} />}
      </MainLayout>

      <div style={{ display: 'none' }} className="fab-wrapper">
        <FAB onClick={() => setModal({ type: 'transaction' })} />
      </div>

      {modal?.type === 'transaction' && <TransactionModal state={state} editItem={modal.item} onSave={(f) => saveTrx(f, modal.item)} onClose={() => setModal(null)} />}
      {modal?.type === 'account' && <AccountModal state={state} editItem={modal.item} onSave={(f) => saveAcc(f, modal.item)} onClose={() => setModal(null)} />}
      {modal?.type === 'investment' && <InvestmentModal state={state} editItem={modal.item} onSave={(f) => saveInv(f, modal.item)} onClose={() => setModal(null)} />}
      {modal?.type === 'category' && <CategoryModal state={state} editItem={modal.item} onSave={(f) => saveCat(f, modal.item)} onClose={() => setModal(null)} />}
      {confirm && <ConfirmModal message={confirm.message} onConfirm={confirm.onConfirm} onClose={() => setConfirm(null)} />}

      <TweaksPanel show={tweaksVisible} tweaks={tweaks} onChange={chgTweaks} />
    </div>);

}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(<App />);