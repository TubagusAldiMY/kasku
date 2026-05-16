// KasKu v3 — Layout with collapsible sidebar

const NAV_ITEMS = [
  { id: 'dashboard', label: 'Dashboard',  icon: '⌘' },
  { id: 'keuangan',  label: 'Keuangan',   icon: '◎' },
  { id: 'investasi', label: 'Investasi',  icon: '◈' },
  { id: 'transaksi', label: 'Transaksi',  icon: '≡' },
  { id: 'kategori',  label: 'Kategori',   icon: '⊞' },
];

const SIDEBAR_FULL = 220;
const SIDEBAR_MINI = 60;

function getSyncMeta() {
  return {
    synced:  { label: 'Synced',   color: T.income  },
    pending: { label: 'Pending…', color: T.warning },
    syncing: { label: 'Syncing…', color: T.accent  },
    failed:  { label: 'Failed',   color: T.expense },
  };
}

function Sidebar({ page, onNav, syncStatus, collapsed, onToggleCollapse }) {
  const m = getSyncMeta()[syncStatus] || getSyncMeta().synced;
  const w = collapsed ? SIDEBAR_MINI : SIDEBAR_FULL;

  return (
    <aside style={{
      width: w, minHeight: '100vh', position: 'fixed', top: 0, left: 0, zIndex: 100,
      background: T.bgSurf,
      borderRight: `1px solid ${T.border}`,
      display: 'flex', flexDirection: 'column',
      boxShadow: T.isDark ? 'none' : '2px 0 12px rgba(0,0,0,0.06)',
      transition: 'width 0.22s cubic-bezier(0.4,0,0.2,1)',
      overflow: 'hidden',
    }}>
      {/* Logo */}
      <div style={{
        padding: collapsed ? '28px 0 22px' : '28px 24px 22px',
        display: 'flex', alignItems: 'center',
        justifyContent: collapsed ? 'center' : 'flex-start',
        minHeight: 72, flexShrink: 0,
      }}>
        {collapsed ? (
          <div style={{ fontSize: 20, fontWeight: 900, ...gLogoText(), display: 'block' }}>K</div>
        ) : (
          <div style={{ fontSize: 22, fontWeight: 900, letterSpacing: '-0.5px', whiteSpace: 'nowrap' }}>
            <span style={{ color: T.text }}>Kas</span>
            <span style={{ ...gLogoText(), display: 'inline-block' }}>Ku</span>
          </div>
        )}
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, padding: collapsed ? '4px 8px' : '4px 12px', overflow: 'hidden' }}>
        {NAV_ITEMS.map(item => {
          const active = page === item.id;
          return (
            <div key={item.id} onClick={() => onNav(item.id)}
              title={collapsed ? item.label : ''}
              style={{
                display: 'flex', alignItems: 'center',
                gap: collapsed ? 0 : 10,
                justifyContent: collapsed ? 'center' : 'flex-start',
                padding: collapsed ? '11px 0' : '10px 12px',
                borderRadius: T.radiusSm, cursor: 'pointer', marginBottom: 2,
                background: active ? T.accentDim : 'transparent',
                color: active ? T.accent : T.textSec,
                fontWeight: active ? 700 : 400, fontSize: 14,
                border: active ? `1px solid ${T.borderHi}` : '1px solid transparent',
                transition: 'all 0.12s',
                whiteSpace: 'nowrap', overflow: 'hidden',
              }}
              onMouseEnter={e => { if (!active) { e.currentTarget.style.background = T.bgHover; e.currentTarget.style.color = T.text; }}}
              onMouseLeave={e => { if (!active) { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = T.textSec; }}}>
              <span style={{
                fontSize: 17, width: collapsed ? 'auto' : 20, textAlign: 'center', flexShrink: 0,
                ...(active ? gText() : { color: 'inherit' }),
              }}>{item.icon}</span>
              {!collapsed && (
                <span style={active ? gText() : { color: 'inherit' }}>{item.label}</span>
              )}
            </div>
          );
        })}
      </nav>

      {/* Footer: sync + collapse toggle */}
      <div style={{
        padding: collapsed ? '14px 0' : '14px 18px',
        borderTop: `1px solid ${T.border}`,
        display: 'flex', flexDirection: 'column', gap: 10, alignItems: collapsed ? 'center' : 'stretch',
        flexShrink: 0,
      }}>
        {!collapsed && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
            <div style={{ width: 6, height: 6, borderRadius: '50%', background: m.color, flexShrink: 0 }} />
            <span style={{ fontSize: 11, color: m.color }}>{m.label}</span>
          </div>
        )}
        {collapsed && (
          <div title={m.label} style={{ width: 7, height: 7, borderRadius: '50%', background: m.color, margin: '0 auto' }} />
        )}
        {/* Collapse toggle */}
        <button onClick={onToggleCollapse} title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} style={{
          width: collapsed ? 36 : '100%', height: 30, borderRadius: T.radiusSm,
          border: `1px solid ${T.border}`, background: T.bgHover,
          cursor: 'pointer', color: T.textSec, fontSize: 13,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          gap: 6, transition: 'all 0.15s',
        }}
          onMouseEnter={e => { e.currentTarget.style.color = T.text; e.currentTarget.style.borderColor = T.borderMd; }}
          onMouseLeave={e => { e.currentTarget.style.color = T.textSec; e.currentTarget.style.borderColor = T.border; }}>
          <span style={{ fontSize: 12 }}>{collapsed ? '→' : '←'}</span>
          {!collapsed && <span style={{ fontSize: 11 }}>Collapse</span>}
        </button>
      </div>
    </aside>
  );
}

function BottomNav({ page, onNav }) {
  return (
    <nav style={{
      position: 'fixed', bottom: 0, left: 0, right: 0, zIndex: 100, height: 60,
      background: T.bgSurf, borderTop: `1px solid ${T.border}`,
      display: 'flex',
      boxShadow: T.isDark ? 'none' : '0 -2px 12px rgba(0,0,0,0.08)',
    }}>
      {NAV_ITEMS.map(item => {
        const active = page === item.id;
        return (
          <div key={item.id} onClick={() => onNav(item.id)} style={{
            flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center',
            justifyContent: 'center', gap: 3, cursor: 'pointer',
            color: active ? T.accent : T.textSec,
            fontSize: 9, fontWeight: active ? 700 : 400, textTransform: 'uppercase', letterSpacing: '0.06em',
          }}>
            <span style={{ fontSize: 18, ...(active ? gText() : {}) }}>{item.icon}</span>
            <span style={active ? gText() : {}}>{item.label}</span>
          </div>
        );
      })}
    </nav>
  );
}

function TopBar({ title, theme, onToggleTheme, onAddTrx, isMobile, syncStatus }) {
  const m = getSyncMeta()[syncStatus] || getSyncMeta().synced;
  return (
    <div style={{
      height: 58, display: 'flex', alignItems: 'center',
      padding: isMobile ? '0 16px' : '0 28px', gap: 12,
      background: T.bgSurf, borderBottom: `1px solid ${T.border}`,
      position: 'sticky', top: 0, zIndex: 50,
      boxShadow: T.isDark ? 'none' : '0 1px 8px rgba(0,0,0,0.06)',
    }}>
      {isMobile && (
        <div style={{ fontSize: 18, fontWeight: 900 }}>
          <span style={{ color: T.text }}>Kas</span>
          <span style={{ ...gLogoText(), display: 'inline-block' }}>Ku</span>
        </div>
      )}
      <div style={{ flex: 1, fontSize: 15, fontWeight: 700, color: T.text }}>{title}</div>
      {!isMobile && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, color: m.color }}>
          <div style={{ width: 5, height: 5, borderRadius: '50%', background: m.color }} />
          {m.label}
        </div>
      )}
      <button onClick={onToggleTheme} style={{
        width: 34, height: 34, borderRadius: T.radiusSm,
        background: T.bgCard, border: `1px solid ${T.border}`,
        cursor: 'pointer', color: T.textSec, fontSize: 15,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>{theme === 'dark' ? '☀' : '🌙'}</button>
      <button onClick={onAddTrx} style={{
        ...getBtnPrimary(), padding: '8px 16px', fontSize: 13,
        display: 'flex', alignItems: 'center', gap: 5, background: T.grad,
      }}>
        <span style={{ fontSize: 16, lineHeight: 1 }}>+</span> Transaksi
      </button>
    </div>
  );
}

function MainLayout({ page, onNav, theme, syncStatus, children, onToggleTheme, onAddTrx, pageTitle }) {
  const [isMobile, setIsMobile] = React.useState(window.innerWidth < 768);
  const [collapsed, setCollapsed] = React.useState(() => localStorage.getItem('kasku_sidebar_collapsed') === 'true');

  React.useEffect(() => {
    const h = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener('resize', h); return () => window.removeEventListener('resize', h);
  }, []);

  const toggleCollapse = () => {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem('kasku_sidebar_collapsed', String(next));
  };

  const sidebarW = isMobile ? 0 : (collapsed ? SIDEBAR_MINI : SIDEBAR_FULL);

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: T.bgBase }}>
      {!isMobile && (
        <Sidebar
          key={'sb-' + page + theme}
          page={page} onNav={onNav} syncStatus={syncStatus}
          collapsed={collapsed} onToggleCollapse={toggleCollapse}
        />
      )}
      <div style={{
        marginLeft: sidebarW, flex: 1, display: 'flex', flexDirection: 'column', minHeight: '100vh',
        transition: 'margin-left 0.22s cubic-bezier(0.4,0,0.2,1)',
      }}>
        <TopBar title={pageTitle} theme={theme} onToggleTheme={onToggleTheme} onAddTrx={onAddTrx} isMobile={isMobile} syncStatus={syncStatus} />
        <div style={{ flex: 1, padding: isMobile ? '20px 14px 80px' : '28px 32px 48px', maxWidth: 1120 }}>
          {children}
        </div>
      </div>
      {isMobile && <BottomNav key={'bn-' + page + theme} page={page} onNav={onNav} />}
    </div>
  );
}

Object.assign(window, { Sidebar, BottomNav, TopBar, MainLayout, NAV_ITEMS });
