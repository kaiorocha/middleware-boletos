import { useEffect, useMemo, useState } from 'react'

const API_DEFAULT = 'http://localhost:8080'
const SESSION_KEY = 'middleware-boletos-session'

const statusLabels = {
  CREATED: 'Criado',
  PROCESSING: 'Processando',
  ISSUED: 'Emitido',
  PAID: 'Pago',
  EXPIRED: 'Vencido',
  CANCELLED: 'Cancelado',
  FAILED: 'Falha',
}

const tenantNav = ['Dashboard', 'Transações', 'Boletos', 'Clientes', 'Providers', 'Compliance', 'Usuários']
const adminNav = ['Dashboard da Plataforma', 'Tenants', 'Usuários Administrativos']

const fmtCurrency = (cents) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(Number(cents || 0) / 100)

const fmtDate = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : new Intl.DateTimeFormat('pt-BR').format(date)
}

async function apiFetch(baseUrl, path, token, options: any = {}) {
  const response = await fetch(`${baseUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options.headers || {}),
    },
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error: any = new Error(payload?.error?.message || 'Falha ao executar operação')
    error.code = payload?.error?.code
    error.status = response.status
    throw error
  }
  return payload
}

function targetPath(session) {
  if (!session) return '/login'
  return session.user?.roles?.includes('PLATFORM_ADMIN') ? '/admin' : '/app'
}

function readSession() {
  if (typeof window === 'undefined') return null
  try {
    return JSON.parse(window.localStorage.getItem(SESSION_KEY) || 'null')
  } catch {
    return null
  }
}

function saveSession(session) {
  window.localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

function clearSession() {
  window.localStorage.removeItem(SESSION_KEY)
}

export default function SaaSPanel() {
  const [baseUrl, setBaseUrl] = useState(API_DEFAULT)
  const [session, setSession] = useState(null)
  const [route, setRoute] = useState('/login')
  const [error, setError] = useState('')

  useEffect(() => {
    const current = readSession()
    setSession(current)
    const path = window.location.pathname === '/' ? targetPath(current) : window.location.pathname
    setRoute(path)
    if (window.location.pathname !== path) window.history.replaceState(null, '', path)
  }, [])

  const navigate = (path) => {
    setRoute(path)
    window.history.pushState(null, '', path)
  }

  const logout = () => {
    clearSession()
    setSession(null)
    navigate('/login')
  }

  const login = async (email, password) => {
    setError('')
    const payload = await apiFetch(baseUrl, '/api/v1/auth/login', '', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
    const next = payload.data
    saveSession(next)
    setSession(next)
    navigate(targetPath(next))
  }

  if (route === '/login' || !session) {
    return <LoginView baseUrl={baseUrl} setBaseUrl={setBaseUrl} login={login} error={error} setError={setError} />
  }

  const shellProps = { baseUrl, setBaseUrl, session, logout, error, setError }
  if (route === '/admin') return <AdminView {...shellProps} />
  return <TenantView {...shellProps} />
}

function LoginView({ baseUrl, setBaseUrl, login, error, setError }) {
  const [email, setEmail] = useState('admin@middleware.local')
  const [password, setPassword] = useState('ChangeMe123456!')
  const [loading, setLoading] = useState(false)

  const submit = async (event) => {
    event.preventDefault()
    setLoading(true)
    setError('')
    try {
      await login(email, password)
    } catch (err) {
      setError(err.message || 'Credenciais inválidas.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="loginPage">
      <form className="loginCard" onSubmit={submit}>
        <strong>Middleware Boletos</strong>
        <h1>Entrar</h1>
        <label>API<input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} /></label>
        <label>E-mail<input value={email} onChange={(event) => setEmail(event.target.value)} /></label>
        <label>Senha<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label>
        {error && <div className="errorBox">{error}</div>}
        <button disabled={loading}>{loading ? 'Entrando...' : 'Entrar'}</button>
      </form>
      <style jsx>{styles}</style>
    </main>
  )
}

function Shell({ title, nav, active, setActive, children, baseUrl, setBaseUrl, session, logout, error }) {
  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand"><span>MB</span><strong>Middleware Boletos</strong></div>
        <nav>{nav.map((item) => <button key={item} className={active === item ? 'navActive' : ''} onClick={() => setActive(item)}>{item}</button>)}</nav>
      </aside>
      <section className="workspace">
        <header className="topbar">
          <div><h1>{title}</h1><p>{session.user.name} · {session.user.email}</p></div>
          <div className="controls">
            <label>API<input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} /></label>
            <button type="button" onClick={logout}>Sair</button>
          </div>
        </header>
        {error && <div className="errorBox">{error}</div>}
        {children}
      </section>
      <style jsx>{styles}</style>
    </main>
  )
}

function AdminView(props) {
  const { baseUrl, session, setError } = props
  const [active, setActive] = useState(adminNav[0])
  const [tenants, setTenants] = useState([])
  const [notice, setNotice] = useState('')
  const [form, setForm] = useState({
    name: 'Cliente Demonstração',
    adminName: 'Administrador Cliente',
    adminEmail: 'cliente@demo.local',
    adminPassword: 'Cliente123456!',
  })

  const load = async () => {
    try {
      const res = await apiFetch(baseUrl, '/api/v1/tenants', session.access_token)
      setTenants(res.data || [])
    } catch (err) {
      setError(err.message)
    }
  }
  useEffect(() => { load() }, [])

  const createTenant = async (event) => {
    event.preventDefault()
    setError('')
    const payload = {
      name: form.name,
      admin: { name: form.adminName, email: form.adminEmail, password: form.adminPassword },
    }
    try {
      await apiFetch(baseUrl, '/api/v1/admin/tenants', session.access_token, { method: 'POST', body: JSON.stringify(payload) })
      setNotice(`Tenant criado. Entregue login ${form.adminEmail} com a senha definida.`)
      await load()
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }

  return (
    <Shell {...props} title="Painel da Plataforma" nav={adminNav} active={active} setActive={setActive}>
      {notice && <div className="notice">{notice}</div>}
      {active === 'Dashboard da Plataforma' && <Metrics items={[['Tenants', tenants.length], ['Administradores', tenants.filter((t) => t.owner_id).length]]} />}
      {active === 'Tenants' && (
        <div className="split">
          <section><DataTable columns={['ID', 'Nome', 'Owner', 'Criado em']} rows={tenants.map((t) => [shortId(t.id), t.name, t.owner_id || '-', fmtDate(t.created_at)])} /></section>
          <FormPanel title="Novo Tenant" onSubmit={createTenant}>
            <label>Nome do Tenant<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
            <label>Nome do Administrador<input value={form.adminName} onChange={(e) => setForm({ ...form, adminName: e.target.value })} /></label>
            <label>E-mail do Administrador<input value={form.adminEmail} onChange={(e) => setForm({ ...form, adminEmail: e.target.value })} /></label>
            <label>Senha inicial<input type="password" value={form.adminPassword} onChange={(e) => setForm({ ...form, adminPassword: e.target.value })} /></label>
            <button>Criar tenant e admin</button>
          </FormPanel>
        </div>
      )}
      {active === 'Usuários Administrativos' && <section className="panel"><p>Usuários `PLATFORM_ADMIN` são gerenciados por bootstrap seguro nesta etapa.</p></section>}
    </Shell>
  )
}

function TenantView(props) {
  const { baseUrl, session, setError } = props
  const [active, setActive] = useState(tenantNav[0])
  const [tenantId, setTenantId] = useState(session.user.tenant_ids?.[0] || '')
  const [tenants, setTenants] = useState([])
  const [dashboard, setDashboard] = useState(null)
  const [boletos, setBoletos] = useState([])
  const [customers, setCustomers] = useState([])
  const [providers, setProviders] = useState([])
  const [blacklist, setBlacklist] = useState([])
  const [users, setUsers] = useState([])
  const [filters, setFilters] = useState({ search: '', status: '', provider: '', from: '', to: '' })
  const [customerForm, setCustomerForm] = useState(emptyCustomer())
  const [providerForm, setProviderForm] = useState({ name: 'Mock', config: '{"delay_ms":0}', status: 'ACTIVE' })
  const [boletoForm, setBoletoForm] = useState({ customer_id: '', provider_id: '', amount_cents: 15000, due_date: '2026-07-30', external_id: '' })
  const [blacklistForm, setBlacklistForm] = useState({ document: '', name: '', reason: 'Solicitação do cliente', source: 'MANUAL' })

  const call = (path, options = {}) => apiFetch(baseUrl, path, session.access_token, options)
  const load = async () => {
    if (!tenantId) return
    try {
      const q = new URLSearchParams()
      if (filters.from) q.set('from', filters.from)
      if (filters.to) q.set('to', filters.to)
      const [me, dash, b, c, p, bl, u] = await Promise.all([
        call('/api/v1/me/tenants'),
        call(`/api/v1/tenants/${tenantId}/dashboard?${q.toString()}`),
        call(`/api/v1/tenants/${tenantId}/boletos`),
        call(`/api/v1/tenants/${tenantId}/customers`),
        call(`/api/v1/tenants/${tenantId}/providers`),
        call(`/api/v1/tenants/${tenantId}/blacklist`),
        call(`/api/v1/tenants/${tenantId}/users`),
      ])
      setTenants(me.data || [])
      setDashboard(dash.data)
      setBoletos(b.data || [])
      setCustomers(c.data || [])
      setProviders(p.data || [])
      setBlacklist(bl.data || [])
      setUsers(u.data || [])
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }
  useEffect(() => { load() }, [tenantId])

  const mutate = async (fn) => { await fn(); await load() }
  const saveCustomer = (event) => {
    event.preventDefault()
    mutate(() => call(`/api/v1/tenants/${tenantId}/customers`, { method: 'POST', body: JSON.stringify(customerForm) }))
  }
  const saveProvider = (event) => {
    event.preventDefault()
    mutate(() => call(`/api/v1/tenants/${tenantId}/providers`, { method: 'POST', body: JSON.stringify(providerForm) }))
  }
  const saveBoleto = (event) => {
    event.preventDefault()
    const body = {
      ...boletoForm,
      provider_id: boletoForm.provider_id || undefined,
      status: 'CREATED',
      amount_cents: Number(boletoForm.amount_cents),
      external_id: boletoForm.external_id || undefined,
    }
    mutate(() => call(`/api/v1/tenants/${tenantId}/boletos`, { method: 'POST', body: JSON.stringify(body) }))
  }
  const emitBoleto = (boleto) => mutate(() => call(`/api/v1/tenants/${tenantId}/boletos/${boleto.id}/emit`, { method: 'POST' }))
  const saveBlacklist = (event) => {
    event.preventDefault()
    mutate(() => call(`/api/v1/tenants/${tenantId}/blacklist`, { method: 'POST', body: JSON.stringify(blacklistForm) }))
  }

  const customerById = useMemo(() => Object.fromEntries(customers.map((c) => [c.id, c])), [customers])
  const providerById = useMemo(() => Object.fromEntries(providers.map((p) => [p.id, p])), [providers])
  const transactions = boletos.filter((b) => {
    const customer = customerById[b.customer_id]
    const provider = b.provider_id ? providerById[b.provider_id] : null
    const haystack = [customer?.name, customer?.document, b.external_id, b.our_number].filter(Boolean).join(' ').toLowerCase()
    return (!filters.search || haystack.includes(filters.search.toLowerCase())) && (!filters.status || b.status === filters.status) && (!filters.provider || provider?.id === filters.provider)
  })

  return (
    <Shell {...props} title={tenants.find((t) => t.id === tenantId)?.name || 'Painel do Tenant'} nav={tenantNav} active={active} setActive={setActive}>
      {tenants.length > 1 && <label className="tenantSelect">Tenant<select value={tenantId} onChange={(e) => setTenantId(e.target.value)}>{tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></label>}
      {active === 'Dashboard' && <Dashboard dashboard={dashboard} filters={filters} setFilters={setFilters} />}
      {active === 'Transações' && <Transactions rows={transactions} customerById={customerById} providerById={providerById} filters={filters} setFilters={setFilters} providers={providers} />}
      {active === 'Boletos' && <Boletos rows={transactions} customers={customers} providers={providers} form={boletoForm} setForm={setBoletoForm} save={saveBoleto} emit={emitBoleto} customerById={customerById} providerById={providerById} />}
      {active === 'Clientes' && <Customers rows={customers} form={customerForm} setForm={setCustomerForm} save={saveCustomer} />}
      {active === 'Providers' && <Providers rows={providers} form={providerForm} setForm={setProviderForm} save={saveProvider} />}
      {active === 'Compliance' && <Compliance rows={blacklist} form={blacklistForm} setForm={setBlacklistForm} save={saveBlacklist} />}
      {active === 'Usuários' && <DataTable columns={['Nome', 'Email', 'Roles', 'Status']} rows={users.map((u) => [u.name, u.email, (u.roles || []).join(', '), u.status])} />}
    </Shell>
  )
}

function Dashboard({ dashboard, filters, setFilters }) {
  return <>
    <div className="toolbar"><label>De<input type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></label><label>Até<input type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></label></div>
    <Metrics items={[['Total de boletos', dashboard?.total_boletos], ['Emitidos', dashboard?.boletos_emitidos], ['Processando', dashboard?.boletos_em_processamento], ['Pagos', dashboard?.boletos_pagos], ['Vencidos', dashboard?.boletos_vencidos], ['Cancelados', dashboard?.boletos_cancelados], ['Falhas', dashboard?.boletos_com_falha], ['Valor emitido', fmtCurrency(dashboard?.valor_total_emitido)]]} />
  </>
}

function Transactions({ rows, customerById, providerById, filters, setFilters, providers }) {
  return <>
    <div className="toolbar"><input placeholder="CPF/CNPJ, cliente, external ID ou nosso número" value={filters.search} onChange={(e) => setFilters({ ...filters, search: e.target.value })} /><select value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}><option value="">Todos os status</option>{Object.keys(statusLabels).map((s) => <option key={s} value={s}>{statusLabels[s]}</option>)}</select><select value={filters.provider} onChange={(e) => setFilters({ ...filters, provider: e.target.value })}><option value="">Todos providers</option>{providers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
    <DataTable columns={['Data', 'Cliente/Pagador', 'CPF/CNPJ', 'Valor', 'Status', 'Provider', 'External ID', 'Nosso Número']} rows={rows.map((b) => [fmtDate(b.created_at), customerById[b.customer_id]?.name || '-', customerById[b.customer_id]?.document || '-', fmtCurrency(b.amount_cents), statusLabels[b.status] || b.status, b.provider_id ? providerById[b.provider_id]?.name : '-', b.external_id || '-', b.our_number || '-'])} />
  </>
}

function Boletos({ rows, customers, providers, form, setForm, save, emit, customerById, providerById }) {
  return <div className="split"><section><DataTable columns={['Cliente', 'Valor', 'Vencimento', 'Status', 'Provider', 'Linha Digitável', 'Ações']} rows={rows.map((b) => [customerById[b.customer_id]?.name || '-', fmtCurrency(b.amount_cents), fmtDate(b.due_date), statusLabels[b.status] || b.status, b.provider_id ? providerById[b.provider_id]?.name : '-', b.digitable_line || '-', <button key="emit" disabled={!['CREATED', 'FAILED'].includes(b.status)} onClick={() => emit(b)}>Emitir</button>])} /></section><FormPanel title="Novo boleto" onSubmit={save}><label>Cliente<select value={form.customer_id} onChange={(e) => setForm({ ...form, customer_id: e.target.value })}><option value="">Selecione</option>{customers.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></label><label>Provider<select value={form.provider_id} onChange={(e) => setForm({ ...form, provider_id: e.target.value })}><option value="">Selecione</option>{providers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label><label>Valor em centavos<input type="number" value={form.amount_cents} onChange={(e) => setForm({ ...form, amount_cents: e.target.value })} /></label><label>Vencimento<input type="date" value={form.due_date} onChange={(e) => setForm({ ...form, due_date: e.target.value })} /></label><label>External ID<input value={form.external_id} onChange={(e) => setForm({ ...form, external_id: e.target.value })} /></label><button>Criar boleto</button></FormPanel></div>
}

function Customers({ rows, form, setForm, save }) {
  return <div className="split"><DataTable columns={['Nome', 'CPF/CNPJ', 'Email', 'Cidade', 'Status']} rows={rows.map((c) => [c.name, c.document || '-', c.email || '-', c.city || '-', c.status])} /><FormPanel title="Novo cliente/pagador" onSubmit={save}>{Object.keys(emptyCustomer()).map((field) => <label key={field}>{customerLabels[field]}<input value={form[field] || ''} onChange={(e) => setForm({ ...form, [field]: e.target.value })} /></label>)}<button>Cadastrar cliente</button></FormPanel></div>
}

function Providers({ rows, form, setForm, save }) {
  return <div className="split"><DataTable columns={['Nome', 'Status', 'Configuração']} rows={rows.map((p) => [p.name, p.status, p.config || '***'])} /><FormPanel title="Novo provider" onSubmit={save}><label>Nome<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label><label>Status<select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}><option>ACTIVE</option><option>INACTIVE</option></select></label><label>Configuração<textarea value={form.config} onChange={(e) => setForm({ ...form, config: e.target.value })} /></label><button>Cadastrar provider</button></FormPanel></div>
}

function Compliance({ rows, form, setForm, save }) {
  return <div className="split"><DataTable columns={['Documento', 'Nome', 'Motivo', 'Origem', 'Status']} rows={rows.map((b) => [b.document, b.name || '-', b.reason || '-', b.source, b.active ? 'Ativo' : 'Inativo'])} /><FormPanel title="Novo bloqueio" onSubmit={save}><label>CPF/CNPJ<input value={form.document} onChange={(e) => setForm({ ...form, document: e.target.value })} /></label><label>Nome<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label><label>Motivo<input value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} /></label><label>Origem<input value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })} /></label><button>Bloquear</button></FormPanel></div>
}

function Metrics({ items }) {
  return <section className="metricGrid">{items.map(([label, value]) => <article className="metric" key={label}><span>{label}</span><strong>{value ?? 0}</strong></article>)}</section>
}

function DataTable({ columns, rows }) {
  return <div className="tableWrap"><table><thead><tr>{columns.map((c) => <th key={c}>{c}</th>)}</tr></thead><tbody>{rows.map((r, i) => <tr key={i}>{r.map((c, j) => <td key={j}>{c}</td>)}</tr>)}{!rows.length && <tr><td colSpan={columns.length} className="emptyCell">Nenhum registro encontrado.</td></tr>}</tbody></table></div>
}

function FormPanel({ title, onSubmit, children }) {
  return <form className="formPanel" onSubmit={onSubmit}><h2>{title}</h2>{children}</form>
}

function emptyCustomer() {
  return { name: '', document: '', email: '', address: '', number: '', complement: '', district: '', city: '', state: '', postal_code: '' }
}

const customerLabels = { name: 'Nome', document: 'CPF/CNPJ', email: 'Email', address: 'Endereço', number: 'Número', complement: 'Complemento', district: 'Bairro', city: 'Cidade', state: 'UF', postal_code: 'CEP' }
const shortId = (id) => (id ? `${id.slice(0, 8)}...` : '-')

const styles = `
:global(body){margin:0;background:#f4f6f8;color:#1f2933;font-family:Inter,Arial,sans-serif}*{box-sizing:border-box}.loginPage{min-height:100vh;display:grid;place-items:center;padding:24px}.loginCard{width:min(420px,100%);display:grid;gap:14px;background:white;border:1px solid #e2e8f0;border-radius:8px;padding:24px}.shell{min-height:100vh;display:grid;grid-template-columns:260px 1fr}.sidebar{background:#111827;color:white;padding:20px 14px}.brand{display:flex;gap:10px;align-items:center;margin-bottom:24px}.brand span{background:#0f766e;border-radius:8px;padding:10px;font-weight:800}nav{display:grid;gap:6px}nav button{background:transparent;border:0;color:#d1d5db;text-align:left}.navActive,nav button:hover{background:#1f2937;color:white}.workspace{padding:24px;min-width:0}.topbar{display:flex;justify-content:space-between;gap:20px;margin-bottom:18px}.controls,.toolbar{display:flex;gap:10px;flex-wrap:wrap;align-items:end}h1,h2{margin:0}p{color:#64748b}label{display:grid;gap:5px;font-size:12px;font-weight:700;color:#52606d}input,select,textarea{min-height:38px;border:1px solid #cbd5e1;border-radius:6px;padding:8px 10px;background:white;color:#1f2933;font:inherit;min-width:180px}textarea{min-height:90px}button{border:1px solid #0f766e;background:#0f766e;color:white;border-radius:6px;min-height:36px;padding:8px 12px;cursor:pointer;font-weight:700}button:disabled{opacity:.45;cursor:not-allowed}.notice,.errorBox{padding:12px 14px;border-radius:6px;margin-bottom:16px}.notice{background:#dcfce7;color:#166534}.errorBox{background:#fee2e2;color:#991b1b}.metricGrid{display:grid;grid-template-columns:repeat(4,minmax(150px,1fr));gap:12px;margin:16px 0}.metric,.panel,.formPanel{background:white;border:1px solid #e2e8f0;border-radius:8px;padding:16px}.metric span{display:block;color:#64748b;margin-bottom:8px}.metric strong{font-size:24px}.split{display:grid;grid-template-columns:minmax(0,1fr) 360px;gap:16px}.tableWrap{overflow:auto;background:white;border:1px solid #e2e8f0;border-radius:8px}table{width:100%;border-collapse:collapse}th,td{padding:11px;border-bottom:1px solid #e2e8f0;text-align:left;vertical-align:top}th{font-size:12px;color:#64748b;background:#f8fafc}.emptyCell{text-align:center;color:#64748b}.formPanel{display:grid;gap:10px;align-content:start}.tenantSelect{margin-bottom:16px}@media(max-width:900px){.shell{grid-template-columns:1fr}.sidebar{position:static}.topbar,.split{display:grid}.metricGrid{grid-template-columns:1fr 1fr}}`
