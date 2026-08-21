import { useEffect, useMemo, useState } from 'react'

const API_DEFAULT = process.env.NEXT_PUBLIC_API_URL || (process.env.NODE_ENV === 'development' ? 'http://localhost:8080' : '')
const SESSION_KEY = 'middleware-boletos-session'
const IS_DEVELOPMENT = process.env.NEXT_PUBLIC_APP_ENV === 'development'
const DEFAULT_EMAIL = IS_DEVELOPMENT
  ? process.env.NEXT_PUBLIC_DEMO_ADMIN_EMAIL || 'admin@middleware.local'
  : ''
const DEFAULT_PASSWORD = IS_DEVELOPMENT ? process.env.NEXT_PUBLIC_DEMO_ADMIN_PASSWORD || '' : ''

const statusLabels = {
  CREATED: 'Criado',
  PROCESSING: 'Processando',
  ISSUED: 'Emitido',
  PAID: 'Pago',
  EXPIRED: 'Vencido',
  CANCELLED: 'Cancelado',
  FAILED: 'Falha',
}

const tenantNavBase = ['Dashboard', 'Transações', 'Boletos', 'Clientes', 'Providers', 'Compliance']
const adminNav = ['Dashboard da Plataforma', 'Transações', 'Tenants', 'Providers', 'Usuários Administrativos']

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
  const [email, setEmail] = useState(DEFAULT_EMAIL)
  const [password, setPassword] = useState(DEFAULT_PASSWORD)
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
  const [providers, setProviders] = useState([])
  const [dashboard, setDashboard] = useState(null)
  const [transactions, setTransactions] = useState({ items: [], limit: 50, offset: 0, total: 0 })
  const [filters, setFilters] = useState({ from: '', to: '', tenant_id: '', provider_id: '', status: '', document: '', external_id: '', our_number: '' })
  const [txOffset, setTxOffset] = useState(0)
  const [notice, setNotice] = useState('')
  const [form, setForm] = useState({
    name: 'Cliente Demonstração',
    document: '',
    address: '',
    district: '',
    city: '',
    postalCode: '',
    state: '',
    countryCode: '55',
    areaCode: '',
    phoneNumber: '',
    webhookUrl: '',
    adminName: 'Administrador Cliente',
    adminEmail: 'cliente@demo.local',
    adminPassword: 'Cliente123456!',
    providerIds: [],
  })
  const [providerForm, setProviderForm] = useState({ name: 'Mock', type: 'BANK', status: 'ACTIVE', config: '{"delay_ms":0}' })
  const [providerEdit, setProviderEdit] = useState(null)
  const [providerConfigs, setProviderConfigs] = useState({})

  const load = async () => {
    try {
      const q = new URLSearchParams()
      if (filters.from) q.set('from', filters.from)
      if (filters.to) q.set('to', filters.to)
      if (filters.tenant_id) q.set('tenant_id', filters.tenant_id)
      if (filters.provider_id) q.set('provider_id', filters.provider_id)
      if (filters.status) q.set('status', filters.status)
      if (filters.document) q.set('document', filters.document)
      if (filters.external_id) q.set('external_id', filters.external_id)
      if (filters.our_number) q.set('our_number', filters.our_number)
      const [tenantRes, providerRes, dashRes, txRes] = await Promise.all([
        apiFetch(baseUrl, '/api/v1/tenants', session.access_token),
        apiFetch(baseUrl, '/api/v1/admin/providers', session.access_token),
        apiFetch(baseUrl, `/api/v1/admin/dashboard?${q.toString()}`, session.access_token),
        apiFetch(baseUrl, `/api/v1/admin/transactions?${q.toString()}&limit=50&offset=${txOffset}`, session.access_token),
      ])
      setTenants(tenantRes.data || [])
      setProviders(providerRes.data || [])
      setDashboard(dashRes.data)
      setTransactions(txRes.data || { items: [], limit: 50, offset: 0, total: 0 })
    } catch (err) {
      setError(err.message)
    }
  }
  useEffect(() => { load() }, [txOffset])

  const createTenant = async (event) => {
    event.preventDefault()
    setError('')
    const payload = {
      name: form.name,
      document: form.document,
      address: form.address,
      district: form.district,
      city: form.city,
      postal_code: form.postalCode,
      state: form.state,
      country_code: form.countryCode,
      area_code: form.areaCode,
      phone_number: form.phoneNumber,
      webhook_url: form.webhookUrl || undefined,
      admin: { name: form.adminName, email: form.adminEmail, password: form.adminPassword },
      providers: form.providerIds.map((provider_id) => ({ provider_id, active: true, config: providerConfigs[provider_id] || undefined })),
    }
    try {
      await apiFetch(baseUrl, '/api/v1/admin/tenants', session.access_token, { method: 'POST', body: JSON.stringify(payload) })
      setNotice(`Tenant criado. Entregue login ${form.adminEmail} com a senha definida.`)
      await load()
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }

  const createProvider = async (event) => {
    event.preventDefault()
    setError('')
    try {
      await apiFetch(baseUrl, '/api/v1/admin/providers', session.access_token, { method: 'POST', body: JSON.stringify(providerForm) })
      setNotice('Provider criado no catálogo da plataforma.')
      await load()
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }

  const updateProvider = async (event) => {
    event.preventDefault()
    if (!providerEdit) return
    setError('')
    try {
      await apiFetch(baseUrl, `/api/v1/admin/providers/${providerEdit.id}`, session.access_token, { method: 'PUT', body: JSON.stringify(providerEdit) })
      setNotice('Provider atualizado.')
      setProviderEdit(null)
      await load()
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }

  const setProviderStatus = async (provider, action) => {
    setError('')
    try {
      await apiFetch(baseUrl, `/api/v1/admin/providers/${provider.id}/${action}`, session.access_token, { method: 'POST' })
      await load()
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }

  return (
    <Shell {...props} title="Painel da Plataforma" nav={adminNav} active={active} setActive={setActive}>
      {notice && <div className="notice">{notice}</div>}
      {active === 'Dashboard da Plataforma' && <AdminDashboard dashboard={dashboard} filters={filters} setFilters={setFilters} tenants={tenants} providers={providers} reload={load} />}
      {active === 'Transações' && <AdminTransactions rows={transactions.items || []} filters={filters} setFilters={setFilters} tenants={tenants} providers={providers} reload={() => { setTxOffset(0); load() }} total={transactions.total} limit={transactions.limit || 50} offset={transactions.offset || txOffset} setOffset={setTxOffset} />}
      {active === 'Tenants' && (
        <div className="split">
          <section><DataTable columns={['ID', 'Nome', 'Owner', 'Criado em']} rows={tenants.map((t) => [shortId(t.id), t.name, t.owner_id || '-', fmtDate(t.created_at)])} /></section>
          <FormPanel title="Novo Tenant" onSubmit={createTenant}>
            <label>Nome do Tenant<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
            <label>CNPJ<input value={form.document} onChange={(e) => setForm({ ...form, document: e.target.value })} /></label>
            <label>Endereço<input value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} /></label>
            <label>Bairro<input value={form.district} onChange={(e) => setForm({ ...form, district: e.target.value })} /></label>
            <label>Cidade<input value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })} /></label>
            <label>CEP<input value={form.postalCode} onChange={(e) => setForm({ ...form, postalCode: e.target.value })} /></label>
            <label>UF<input maxLength={2} value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value })} /></label>
            <div className="threeCols"><label>DDI<input value={form.countryCode} onChange={(e) => setForm({ ...form, countryCode: e.target.value })} /></label><label>DDD<input value={form.areaCode} onChange={(e) => setForm({ ...form, areaCode: e.target.value })} /></label><label>Celular<input value={form.phoneNumber} onChange={(e) => setForm({ ...form, phoneNumber: e.target.value })} /></label></div>
            <label>URL de webhooks<input type="url" value={form.webhookUrl} onChange={(e) => setForm({ ...form, webhookUrl: e.target.value })} placeholder="https://cliente.exemplo/webhooks" /></label>
            <label>Nome do Administrador<input value={form.adminName} onChange={(e) => setForm({ ...form, adminName: e.target.value })} /></label>
            <label>E-mail do Administrador<input value={form.adminEmail} onChange={(e) => setForm({ ...form, adminEmail: e.target.value })} /></label>
            <label>Senha inicial<input type="password" value={form.adminPassword} onChange={(e) => setForm({ ...form, adminPassword: e.target.value })} /></label>
            <fieldset><legend>Providers habilitados</legend>{providers.map((p) => <div key={p.id} className="providerChoice"><label className="checkRow"><input type="checkbox" checked={form.providerIds.includes(p.id)} onChange={(e) => setForm({ ...form, providerIds: e.target.checked ? [...form.providerIds, p.id] : form.providerIds.filter((id) => id !== p.id) })} />{p.name}</label>{form.providerIds.includes(p.id) && <label>Configuração do tenant<input type="password" value={providerConfigs[p.id] || ''} onChange={(e) => setProviderConfigs({ ...providerConfigs, [p.id]: e.target.value })} placeholder="JSON ou segredo específico do tenant" /></label>}</div>)}</fieldset>
            <button>Criar tenant e admin</button>
          </FormPanel>
        </div>
      )}
      {active === 'Providers' && <ProvidersAdmin rows={providers} form={providerForm} setForm={setProviderForm} save={createProvider} edit={providerEdit} setEdit={setProviderEdit} update={updateProvider} setStatus={setProviderStatus} />}
      {active === 'Usuários Administrativos' && <section className="panel"><p>Usuários `PLATFORM_ADMIN` são gerenciados por bootstrap seguro nesta etapa.</p></section>}
    </Shell>
  )
}

function AdminDashboard({ dashboard, filters, setFilters, tenants, providers, reload }) {
  const totals = dashboard?.totals || {}
  return <>
    <GlobalFilters filters={filters} setFilters={setFilters} tenants={tenants} providers={providers} reload={reload} />
    <Metrics items={[
      ['Tenants ativos', totals.tenants],
      ['Total de boletos', totals.boletos],
      ['Emissões', totals.issued],
      ['Volume emitido', fmtCurrency(totals.amount_cents)],
      ['Criados', totals.created],
      ['Processando', totals.processing],
      ['Pagos', totals.paid],
      ['Falhas', totals.failed],
      ['Taxa de sucesso', `${Math.round((totals.success_rate || 0) * 100)}%`],
      ['Taxa de falha', `${Math.round((totals.failure_rate || 0) * 100)}%`],
      ['Ticket médio', fmtCurrency(totals.average_ticket_cents)],
      ['Cancelados', totals.cancelled],
    ]} />
    <div className="threeCols">
      <SimpleBars title="Top tenants por emissão" rows={dashboard?.by_tenant || []} />
      <SimpleBars title="Emissões por provider" rows={dashboard?.by_provider || []} />
      <SimpleBars title="Status das emissões" rows={dashboard?.by_status || []} />
    </div>
  </>
}

function AdminTransactions({ rows, filters, setFilters, tenants, providers, reload, total, limit, offset, setOffset }) {
  return <>
    <GlobalFilters filters={filters} setFilters={setFilters} tenants={tenants} providers={providers} reload={reload} />
    <section className="panel"><strong>{total || 0}</strong> transações encontradas</section>
    <DataTable columns={['Data', 'Tenant', 'Cliente/Email', 'Valor', 'Status', 'Provider', 'External ID', 'Nosso Número']} rows={rows.map((b) => [fmtDate(b.created_at), b.tenant_name || shortId(b.tenant_id), b.customer_name || b.recipient_email || shortId(b.customer_id), fmtCurrency(b.amount_cents), statusLabels[b.status] || b.status, b.provider_name || '-', b.external_id || '-', b.our_number || '-'])} />
    <Pagination limit={limit} offset={offset} total={total} shown={rows.length} setOffset={setOffset} />
  </>
}

function GlobalFilters({ filters, setFilters, tenants, providers, reload }) {
  return <div className="toolbar">
    <label>De<input type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></label>
    <label>Até<input type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></label>
    <label>Tenant<select value={filters.tenant_id} onChange={(e) => setFilters({ ...filters, tenant_id: e.target.value })}><option value="">Todos</option>{tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></label>
    <label>Provider<select value={filters.provider_id} onChange={(e) => setFilters({ ...filters, provider_id: e.target.value })}><option value="">Todos</option>{providers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label>
    <label>Status<select value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}><option value="">Todos</option>{Object.keys(statusLabels).map((s) => <option key={s} value={s}>{statusLabels[s]}</option>)}</select></label>
    <label>CPF/CNPJ<input value={filters.document || ''} onChange={(e) => setFilters({ ...filters, document: e.target.value })} /></label>
    <label>External ID<input value={filters.external_id || ''} onChange={(e) => setFilters({ ...filters, external_id: e.target.value })} /></label>
    <label>Nosso Número<input value={filters.our_number || ''} onChange={(e) => setFilters({ ...filters, our_number: e.target.value })} /></label>
    <button type="button" onClick={reload}>Aplicar</button>
  </div>
}

function Pagination({ limit, offset, total, shown, setOffset }) {
  const page = Math.floor((offset || 0) / (limit || 50)) + 1
  return <div className="pager"><button type="button" disabled={(offset || 0) <= 0} onClick={() => setOffset(Math.max(0, (offset || 0) - (limit || 50)))}>Anterior</button><span>Página {page} · {shown || 0} exibidos · {total || 0} no total</span><button type="button" disabled={(offset || 0) + (limit || 50) >= (total || 0)} onClick={() => setOffset((offset || 0) + (limit || 50))}>Próxima</button></div>
}

function SimpleBars({ title, rows }) {
  const max = Math.max(1, ...rows.map((r) => r.count || 0))
  return <section className="panel"><h2>{title}</h2><div className="bars">{rows.map((r) => <div key={`${r.id}-${r.label}`}><span>{r.label}</span><div><i style={{ width: `${Math.max(4, ((r.count || 0) / max) * 100)}%` }} /></div><strong>{r.count}</strong></div>)}</div></section>
}

function ProvidersAdmin({ rows, form, setForm, save, edit, setEdit, update, setStatus }) {
  return <div className="split">
    <DataTable columns={['Nome', 'Tipo', 'Status', 'External ID', 'Ações']} rows={rows.map((p) => [p.name, p.type || '-', p.status, p.external_id || '-', <div className="rowActions" key={p.id}><button type="button" onClick={() => setEdit({ id: p.id, name: p.name, type: p.type || '', status: p.status, metadata: p.metadata || '' })}>Editar</button><button type="button" onClick={() => setStatus(p, p.status === 'ACTIVE' ? 'deactivate' : 'activate')}>{p.status === 'ACTIVE' ? 'Desativar' : 'Ativar'}</button></div>])} />
    <div className="formStack">
      <FormPanel title="Novo provider do catálogo" onSubmit={save}>
        <label>Nome<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
        <label>Tipo<input value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })} /></label>
        <label>Status<select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}><option>ACTIVE</option><option>INACTIVE</option></select></label>
        <label>Configuração global<input type="password" value={form.config} onChange={(e) => setForm({ ...form, config: e.target.value })} /></label>
        <button>Criar provider</button>
      </FormPanel>
      {edit && <FormPanel title="Editar provider" onSubmit={update}>
        <label>Nome<input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} /></label>
        <label>Tipo<input value={edit.type} onChange={(e) => setEdit({ ...edit, type: e.target.value })} /></label>
        <label>Metadata<textarea value={edit.metadata} onChange={(e) => setEdit({ ...edit, metadata: e.target.value })} /></label>
        <button>Salvar alterações</button>
      </FormPanel>}
    </div>
  </div>
}

function TenantView(props) {
  const { baseUrl, session, setError } = props
  const canManageTenant = session.user.roles?.includes('TENANT_ADMIN')
  const tenantNav = canManageTenant ? [...tenantNavBase, 'Usuários'] : tenantNavBase
  const [active, setActive] = useState(tenantNav[0])
  const [tenantId, setTenantId] = useState(session.user.tenant_ids?.[0] || '')
  const [tenants, setTenants] = useState([])
  const [dashboard, setDashboard] = useState(null)
  const [boletos, setBoletos] = useState([])
  const [tenantTransactions, setTenantTransactions] = useState({ items: [], limit: 50, offset: 0, total: 0 })
  const [customers, setCustomers] = useState([])
  const [providers, setProviders] = useState([])
  const [blacklist, setBlacklist] = useState([])
  const [users, setUsers] = useState([])
  const [filters, setFilters] = useState({ status: '', provider_id: '', from: '', to: '', document: '', external_id: '', our_number: '' })
  const [tenantTxOffset, setTenantTxOffset] = useState(0)
  const [blacklistForm, setBlacklistForm] = useState({ document: '', name: '', reason: 'Solicitação do cliente', source: 'MANUAL' })

  const call = (path, options = {}) => apiFetch(baseUrl, path, session.access_token, options)
  const load = async () => {
    if (!tenantId) return
    try {
      const q = new URLSearchParams()
      if (filters.from) q.set('from', filters.from)
      if (filters.to) q.set('to', filters.to)
      if (filters.status) q.set('status', filters.status)
      if (filters.provider_id) q.set('provider_id', filters.provider_id)
      if (filters.document) q.set('document', filters.document)
      if (filters.external_id) q.set('external_id', filters.external_id)
      if (filters.our_number) q.set('our_number', filters.our_number)
      const [me, dash, tx, c, p, bl, u] = await Promise.all([
        call('/api/v1/me/tenants'),
        call(`/api/v1/tenants/${tenantId}/dashboard?${q.toString()}`),
        call(`/api/v1/tenants/${tenantId}/transactions?${q.toString()}&limit=50&offset=${tenantTxOffset}`),
        call(`/api/v1/tenants/${tenantId}/customers`),
        call(`/api/v1/tenants/${tenantId}/providers`),
        call(`/api/v1/tenants/${tenantId}/blacklist`),
        canManageTenant ? call(`/api/v1/tenants/${tenantId}/users`) : Promise.resolve({ data: [] }),
      ])
      setTenants(me.data || [])
      setDashboard(dash.data)
      setTenantTransactions(tx.data || { items: [], limit: 50, offset: tenantTxOffset, total: 0 })
      setBoletos(tx.data?.items || [])
      setCustomers(c.data || [])
      setProviders(p.data || [])
      setBlacklist(bl.data || [])
      setUsers(u.data || [])
    } catch (err) {
      setError(`${err.code || err.status}: ${err.message}`)
    }
  }
  useEffect(() => { load() }, [tenantId, tenantTxOffset])

  const mutate = async (fn) => { await fn(); await load() }
  const saveBlacklist = (event) => {
    event.preventDefault()
    mutate(() => call(`/api/v1/tenants/${tenantId}/blacklist`, { method: 'POST', body: JSON.stringify(blacklistForm) }))
  }

  const providerById = useMemo(() => Object.fromEntries(providers.map((p) => [p.id, p])), [providers])
  const transactions = tenantTransactions.items || []

  return (
    <Shell {...props} title={tenants.find((t) => t.id === tenantId)?.name || 'Painel do Tenant'} nav={tenantNav} active={active} setActive={setActive}>
      {tenants.length > 1 && <label className="tenantSelect">Tenant<select value={tenantId} onChange={(e) => setTenantId(e.target.value)}>{tenants.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></label>}
      {active === 'Dashboard' && <Dashboard dashboard={dashboard} filters={filters} setFilters={setFilters} />}
      {active === 'Transações' && <Transactions rows={transactions} providerById={providerById} filters={filters} setFilters={setFilters} providers={providers} reload={() => { setTenantTxOffset(0); load() }} pagination={tenantTransactions} setOffset={setTenantTxOffset} />}
      {active === 'Boletos' && <Boletos rows={boletos} providerById={providerById} />}
      {active === 'Clientes' && <Customers rows={customers} />}
      {active === 'Providers' && <Providers rows={providers} />}
      {active === 'Compliance' && <Compliance rows={blacklist} form={blacklistForm} setForm={setBlacklistForm} save={saveBlacklist} canManage={canManageTenant} />}
      {active === 'Usuários' && canManageTenant && <DataTable columns={['Nome', 'Email', 'Roles', 'Status']} rows={users.map((u) => [u.name, u.email, (u.roles || []).join(', '), u.status])} />}
    </Shell>
  )
}

function Dashboard({ dashboard, filters, setFilters }) {
  return <>
    <div className="toolbar"><label>De<input type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></label><label>Até<input type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></label></div>
    <Metrics items={[['Total de boletos', dashboard?.total_boletos], ['Emitidos', dashboard?.boletos_emitidos], ['Processando', dashboard?.boletos_em_processamento], ['Pagos', dashboard?.boletos_pagos], ['Vencidos', dashboard?.boletos_vencidos], ['Cancelados', dashboard?.boletos_cancelados], ['Falhas', dashboard?.boletos_com_falha], ['Valor emitido', fmtCurrency(dashboard?.valor_total_emitido)], ['Taxa de sucesso', `${Math.round((dashboard?.taxa_sucesso || 0) * 100)}%`], ['Taxa de falha', `${Math.round((dashboard?.taxa_falha || 0) * 100)}%`], ['Ticket médio', fmtCurrency(dashboard?.ticket_medio)]]} />
  </>
}

function Transactions({ rows, providerById, filters, setFilters, providers, reload, pagination, setOffset }) {
  return <>
    <div className="toolbar"><label>De<input type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></label><label>Até<input type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></label><label>Status<select value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}><option value="">Todos os status</option>{Object.keys(statusLabels).map((s) => <option key={s} value={s}>{statusLabels[s]}</option>)}</select></label><label>Provider<select value={filters.provider_id} onChange={(e) => setFilters({ ...filters, provider_id: e.target.value })}><option value="">Todos providers</option>{providers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label><label>CPF/CNPJ<input value={filters.document} onChange={(e) => setFilters({ ...filters, document: e.target.value })} /></label><label>Email<input value={filters.email} onChange={(e) => setFilters({ ...filters, email: e.target.value })} /></label><label>External ID<input value={filters.external_id} onChange={(e) => setFilters({ ...filters, external_id: e.target.value })} /></label><label>Nosso Número<input value={filters.our_number} onChange={(e) => setFilters({ ...filters, our_number: e.target.value })} /></label><button type="button" onClick={reload}>Aplicar</button></div>
    <DataTable columns={['Data', 'Cliente/Email', 'CPF/CNPJ', 'Valor', 'Status', 'Provider', 'External ID', 'Nosso Número']} rows={rows.map((b) => [fmtDate(b.created_at), b.customer_name || b.recipient_email || '-', b.customer_document || '-', fmtCurrency(b.amount_cents), statusLabels[b.status] || b.status, b.provider_name || (b.provider_id ? providerById[b.provider_id]?.name : '-'), b.external_id || '-', b.our_number || '-'])} />
    <Pagination limit={pagination?.limit || 50} offset={pagination?.offset || 0} total={pagination?.total || 0} shown={rows.length} setOffset={setOffset} />
  </>
}

function Boletos({ rows, providerById }) {
  return <DataTable columns={['Cliente/Email', 'Valor', 'Vencimento', 'Status', 'Provider', 'Linha Digitável', 'External ID']} rows={rows.map((b) => [b.customer_name || b.recipient_email || '-', fmtCurrency(b.amount_cents), fmtDate(b.due_date), statusLabels[b.status] || b.status, b.provider_name || (b.provider_id ? providerById[b.provider_id]?.name : '-'), b.digitable_line || '-', b.external_id || '-'])} />
}

function Customers({ rows }) {
  return <DataTable columns={['Nome', 'CPF/CNPJ', 'Email', 'Cidade', 'Status']} rows={rows.map((c) => [c.name, c.document || '-', c.email || '-', c.city || '-', c.status])} />
}

function Providers({ rows }) {
  return <DataTable columns={['Nome', 'Tipo', 'Status']} rows={rows.map((p) => [p.name, p.type || '-', p.status])} />
}

function Compliance({ rows, form, setForm, save, canManage }) {
  return <div className={canManage ? 'split' : ''}><DataTable columns={['Documento', 'Nome', 'Motivo', 'Origem', 'Status']} rows={rows.map((b) => [b.document, b.name || '-', b.reason || '-', b.source, b.active ? 'Ativo' : 'Inativo'])} />{canManage && <FormPanel title="Novo bloqueio" onSubmit={save}><label>CPF/CNPJ<input value={form.document} onChange={(e) => setForm({ ...form, document: e.target.value })} /></label><label>Nome<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label><label>Motivo<input value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} /></label><label>Origem<input value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })} /></label><button>Bloquear</button></FormPanel>}</div>
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

const shortId = (id) => (id ? `${id.slice(0, 8)}...` : '-')

const styles = `
:global(body){margin:0;background:#f4f6f8;color:#1f2933;font-family:Inter,Arial,sans-serif}*{box-sizing:border-box}.loginPage{min-height:100vh;display:grid;place-items:center;padding:32px}.loginCard{width:min(420px,100%);display:grid;gap:16px;background:white;border:1px solid #e2e8f0;border-radius:8px;padding:24px}.shell{min-height:100vh;display:grid;grid-template-columns:260px 1fr}.sidebar{background:#111827;color:white;padding:24px 16px}.brand{display:flex;gap:12px;align-items:center;margin-bottom:28px}.brand span{background:#0f766e;border-radius:8px;padding:10px;font-weight:800}nav{display:grid;gap:8px}nav button{background:transparent;border:0;color:#d1d5db;text-align:left}.navActive,nav button:hover{background:#1f2937;color:white}.workspace{padding:32px;min-width:0}.topbar{display:flex;justify-content:space-between;gap:24px;margin-bottom:24px}.controls,.toolbar{display:flex;gap:12px;flex-wrap:wrap;align-items:end;margin-bottom:24px}h1,h2{margin:0}h2{font-size:16px}p{color:#64748b}label{display:grid;gap:6px;font-size:12px;font-weight:700;color:#52606d}input,select,textarea{min-height:38px;border:1px solid #cbd5e1;border-radius:6px;padding:8px 10px;background:white;color:#1f2933;font:inherit;min-width:180px}textarea{min-height:92px}button{border:1px solid #0f766e;background:#0f766e;color:white;border-radius:6px;min-height:36px;padding:8px 12px;cursor:pointer;font-weight:700}button:disabled{opacity:.45;cursor:not-allowed}.notice,.errorBox{padding:12px 14px;border-radius:6px;margin-bottom:16px}.notice{background:#dcfce7;color:#166534}.errorBox{background:#fee2e2;color:#991b1b}.metricGrid{display:grid;grid-template-columns:repeat(4,minmax(150px,1fr));gap:16px;margin:24px 0}.metric,.panel,.formPanel{background:white;border:1px solid #e2e8f0;border-radius:8px;padding:20px}.metric span{display:block;color:#64748b;margin-bottom:8px}.metric strong{font-size:24px}.split{display:grid;grid-template-columns:minmax(0,1fr) 380px;gap:24px}.threeCols{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.tableWrap{overflow:auto;background:white;border:1px solid #e2e8f0;border-radius:8px}table{width:100%;border-collapse:collapse}th,td{padding:12px 14px;border-bottom:1px solid #e2e8f0;text-align:left;vertical-align:top}th{font-size:12px;color:#64748b;background:#f8fafc}.emptyCell{text-align:center;color:#64748b}.formPanel,.formStack{display:grid;gap:16px;align-content:start}.tenantSelect{margin-bottom:16px}fieldset{border:1px solid #e2e8f0;border-radius:8px;padding:12px;display:grid;gap:12px}legend{font-size:12px;font-weight:800;color:#52606d}.checkRow{display:flex;align-items:center;gap:8px}.checkRow input{min-width:auto;min-height:auto}.providerChoice{display:grid;gap:8px}.rowActions{display:flex;gap:8px;flex-wrap:wrap}.pager{display:flex;justify-content:flex-end;align-items:center;gap:12px;margin-top:16px}.bars{display:grid;gap:10px;margin-top:12px}.bars>div{display:grid;grid-template-columns:1fr 3fr 42px;gap:8px;align-items:center;font-size:12px}.bars div div{height:8px;background:#e2e8f0;border-radius:999px;overflow:hidden}.bars i{display:block;height:100%;background:#0f766e}@media(max-width:900px){.shell{grid-template-columns:1fr}.sidebar{position:static}.topbar,.split{display:grid}.metricGrid,.threeCols{grid-template-columns:1fr 1fr}}@media(max-width:640px){.workspace{padding:24px}.metricGrid,.threeCols{grid-template-columns:1fr}.toolbar,.controls,.pager{align-items:stretch;justify-content:stretch}.toolbar label,.controls label,input,select,textarea{width:100%;min-width:0}.pager{display:grid}}`
