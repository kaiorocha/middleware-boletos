import { useEffect, useMemo, useState } from 'react'

const API_DEFAULT = 'http://localhost:8080'
type ApiPayload = Record<string, any>
const IS_DEVELOPMENT =
  process.env.NEXT_PUBLIC_APP_ENV === 'development' || process.env.NODE_ENV === 'development'
const SESSION_TENANT_ID = process.env.NEXT_PUBLIC_TENANT_ID || ''
const SESSION_USER_ID = process.env.NEXT_PUBLIC_USER_ID || ''
const SESSION_ACCESS_TOKEN = process.env.NEXT_PUBLIC_ACCESS_TOKEN || ''

const navItems = [
  'Dashboard',
  'Boletos',
  'Clientes',
  'Providers',
  'Compliance',
  'Usuários',
  'Configurações',
]

const statusLabels: Record<string, string> = {
  CREATED: 'Criado',
  PROCESSING: 'Processando',
  ISSUED: 'Emitido',
  PAID: 'Pago',
  EXPIRED: 'Vencido',
  CANCELLED: 'Cancelado',
  FAILED: 'Falha',
}

const fmtCurrency = (cents: any) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format((Number(cents || 0) / 100))

const fmtDate = (value: any) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat('pt-BR').format(date)
}

async function apiFetch(baseUrl: string, path: string, options: RequestInit = {}): Promise<ApiPayload> {
  const response = await fetch(`${baseUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
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

function tenantsFromToken(token: string): string[] {
  const parts = token.trim().split('.')
  if (parts.length !== 3 || typeof window === 'undefined') return []
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const payload = JSON.parse(window.atob(base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')))
    const tenants = new Set<string>()
    if (typeof payload.tenant_id === 'string') tenants.add(payload.tenant_id)
    if (Array.isArray(payload.tenant_ids)) {
      payload.tenant_ids.forEach((id: unknown) => {
        if (typeof id === 'string') tenants.add(id)
      })
    }
    return Array.from(tenants)
  } catch {
    return []
  }
}

export default function AdminPanel() {
  const [baseUrl, setBaseUrl] = useState(API_DEFAULT)
  const [tenantId, setTenantId] = useState(SESSION_TENANT_ID)
  const [userId, setUserId] = useState(SESSION_USER_ID)
  const [accessToken, setAccessToken] = useState(SESSION_ACCESS_TOKEN)
  const [authorizedTenantIds, setAuthorizedTenantIds] = useState<string[]>([])
  const [active, setActive] = useState('Dashboard')
  const [loading, setLoading] = useState(false)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [dashboard, setDashboard] = useState<any>(null)
  const [boletos, setBoletos] = useState<any[]>([])
  const [customers, setCustomers] = useState<any[]>([])
  const [providers, setProviders] = useState<any[]>([])
  const [blacklist, setBlacklist] = useState<any[]>([])
  const [users, setUsers] = useState<any[]>([])
  const [selectedBoleto, setSelectedBoleto] = useState<any>(null)
  const [selectedCustomer, setSelectedCustomer] = useState<any>(null)
  const [selectedProvider, setSelectedProvider] = useState<any>(null)
  const [filters, setFilters] = useState({
    periodFrom: '',
    periodTo: '',
    boletoSearch: '',
    boletoStatus: '',
    boletoProvider: '',
    boletoCustomer: '',
    customerSearch: '',
    complianceSearch: '',
    complianceActive: '',
  })
  const [customerForm, setCustomerForm] = useState(emptyCustomer())
  const [providerForm, setProviderForm] = useState({ name: '', config: '', status: 'ACTIVE' })
  const [blacklistForm, setBlacklistForm] = useState({
    document: '',
    name: '',
    reason: '',
    notes: '',
    source: 'MANUAL',
  })

  const ready = Boolean(tenantId.trim() && (accessToken.trim() || (IS_DEVELOPMENT && userId.trim())))

  const authHeaders = () => {
    if (accessToken.trim()) {
      return { Authorization: `Bearer ${accessToken.trim()}` }
    }
    if (IS_DEVELOPMENT && tenantId.trim() && userId.trim()) {
      return {
        'X-Dev-User-ID': userId.trim(),
        'X-Dev-Tenant-ID': tenantId.trim(),
      }
    }
    return {}
  }

  const callApi = (path: string, options: RequestInit = {}) =>
    apiFetch(baseUrl, path, {
      ...options,
      headers: {
        ...authHeaders(),
        ...(options.headers || {}),
      },
    })

  const loadAll = async () => {
    if (!ready) return
    setLoading(true)
    setError('')
    try {
      const query = new URLSearchParams()
      if (filters.periodFrom) query.set('from', filters.periodFrom)
      if (filters.periodTo) query.set('to', filters.periodTo)
      const [dash, boletosRes, customersRes, providersRes, blacklistRes, usersRes] = await Promise.all([
        callApi(`/api/v1/tenants/${tenantId}/dashboard?${query.toString()}`),
        callApi(`/api/v1/tenants/${tenantId}/boletos`),
        callApi(`/api/v1/tenants/${tenantId}/customers`),
        callApi(`/api/v1/tenants/${tenantId}/providers`),
        callApi(`/api/v1/tenants/${tenantId}/blacklist`),
        callApi(`/api/v1/tenants/${tenantId}/users`),
      ])
      setDashboard(dash.data)
      setBoletos(boletosRes.data || [])
      setCustomers(customersRes.data || [])
      setProviders(providersRes.data || [])
      setBlacklist(blacklistRes.data || [])
      setUsers(usersRes.data || [])
    } catch (err) {
      setError(`${err.code || err.status || 'ERRO'}: ${err.message}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [tenantId])

  useEffect(() => {
    const tenants = tenantsFromToken(accessToken)
    setAuthorizedTenantIds(tenants)
    if (!IS_DEVELOPMENT && tenants.length > 0 && (!tenantId || !tenants.includes(tenantId))) {
      setTenantId(tenants[0])
    }
  }, [accessToken])

  const blockedDocuments = useMemo(() => {
    return new Set(blacklist.filter((entry) => entry.active).map((entry) => entry.document))
  }, [blacklist])

  const customerById = useMemo(() => {
    return Object.fromEntries(customers.map((customer) => [customer.id, customer]))
  }, [customers])

  const providerById = useMemo(() => {
    return Object.fromEntries(providers.map((provider) => [provider.id, provider]))
  }, [providers])

  const filteredBoletos = useMemo(() => {
    const search = filters.boletoSearch.trim().toLowerCase()
    return boletos.filter((boleto) => {
      const customer = customerById[boleto.customer_id]
      const provider = boleto.provider_id ? providerById[boleto.provider_id] : null
      const haystack = [
        boleto.id,
        boleto.our_number,
        boleto.digitable_line,
        boleto.barcode,
        boleto.external_id,
        customer?.document,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      return (
        (!search || haystack.includes(search)) &&
        (!filters.boletoStatus || boleto.status === filters.boletoStatus) &&
        (!filters.boletoProvider || provider?.id === filters.boletoProvider) &&
        (!filters.boletoCustomer || customer?.id === filters.boletoCustomer)
      )
    })
  }, [boletos, customerById, providerById, filters])

  const filteredCustomers = useMemo(() => {
    const search = filters.customerSearch.trim().toLowerCase()
    return customers.filter((customer) => {
      const haystack = [customer.name, customer.document].filter(Boolean).join(' ').toLowerCase()
      return !search || haystack.includes(search)
    })
  }, [customers, filters.customerSearch])

  const filteredBlacklist = useMemo(() => {
    const search = filters.complianceSearch.trim().toLowerCase()
    return blacklist.filter((entry) => {
      const haystack = [entry.document, entry.name, entry.reason].filter(Boolean).join(' ').toLowerCase()
      const activeMatch =
        filters.complianceActive === '' || String(entry.active) === filters.complianceActive
      return (!search || haystack.includes(search)) && activeMatch
    })
  }, [blacklist, filters.complianceSearch, filters.complianceActive])

  const flash = (message) => {
    setNotice(message)
    setTimeout(() => setNotice(''), 3500)
  }

  const saveCustomer = async (event) => {
    event.preventDefault()
    await mutate(async () => {
      const body = JSON.stringify(customerForm)
      if (customerForm.id) {
        await callApi(`/api/v1/tenants/${tenantId}/customers/${customerForm.id}`, {
          method: 'PUT',
          body,
        })
        flash('Cliente atualizado.')
      } else {
        await callApi(`/api/v1/tenants/${tenantId}/customers`, { method: 'POST', body })
        flash('Cliente cadastrado.')
      }
      setCustomerForm(emptyCustomer())
    })
  }

  const saveProvider = async (event) => {
    event.preventDefault()
    await mutate(async () => {
      await callApi(`/api/v1/tenants/${tenantId}/providers`, {
        method: 'POST',
        body: JSON.stringify(providerForm),
      })
      setProviderForm({ name: '', config: '', status: 'ACTIVE' })
      flash('Provider cadastrado.')
    })
  }

  const saveBlacklistEntry = async (event) => {
    event.preventDefault()
    await mutate(async () => {
      await callApi(`/api/v1/tenants/${tenantId}/blacklist`, {
        method: 'POST',
        body: JSON.stringify(blacklistForm),
      })
      setBlacklistForm({ document: '', name: '', reason: '', notes: '', source: 'MANUAL' })
      flash('Bloqueio cadastrado.')
    })
  }

  const mutate = async (fn) => {
    if (!ready) {
      setError('Informe um tenantId para operar o painel.')
      return
    }
    setLoading(true)
    setError('')
    try {
      await fn()
      await loadAll()
    } catch (err) {
      setError(`${err.code || err.status || 'ERRO'}: ${err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const emitBoleto = async (boleto) => {
    await mutate(async () => {
      await callApi(`/api/v1/tenants/${tenantId}/boletos/${boleto.id}/emit`, { method: 'POST' })
      flash('Emissão solicitada.')
    })
  }

  const blacklistAction = async (entry, action) => {
    await mutate(async () => {
      await callApi(`/api/v1/tenants/${tenantId}/blacklist/${entry.id}/${action}`, {
        method: 'POST',
      })
      flash(action === 'block' ? 'Bloqueio ativado.' : 'Bloqueio desativado.')
    })
  }

  const deleteBlacklistEntry = async (entry) => {
    await mutate(async () => {
      await callApi(`/api/v1/tenants/${tenantId}/blacklist/${entry.id}`, { method: 'DELETE' })
      flash('Bloqueio excluído logicamente.')
    })
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brandMark">MB</span>
          <div>
            <strong>Middleware Boletos</strong>
            <small>Painel Administrativo</small>
          </div>
        </div>
        <nav>
          {navItems.map((item) => (
            <button
              key={item}
              className={active === item ? 'navActive' : ''}
              type="button"
              onClick={() => setActive(item)}
            >
              {item}
            </button>
          ))}
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>{active}</h1>
            <p>Operação multi-tenant para emissão, provedores e compliance.</p>
          </div>
          <div className="tenantControls">
            <label>
              API
              <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} />
            </label>
            {IS_DEVELOPMENT ? (
              <>
                <label>
                  Tenant ID
                  <input
                    value={tenantId}
                    placeholder="UUID do tenant"
                    onChange={(event) => setTenantId(event.target.value)}
                  />
                </label>
                <label>
                  User ID
                  <input
                    value={userId}
                    placeholder="UUID do usuário"
                    onChange={(event) => setUserId(event.target.value)}
                  />
                </label>
                <label>
                  Access Token
                  <input
                    value={accessToken}
                    placeholder="JWT Bearer opcional"
                    onChange={(event) => setAccessToken(event.target.value)}
                  />
                </label>
              </>
            ) : authorizedTenantIds.length > 1 ? (
              <label>
                Tenant da sessão
                <select value={tenantId} onChange={(event) => setTenantId(event.target.value)}>
                  {authorizedTenantIds.map((id) => (
                    <option key={id} value={id}>
                      {id}
                    </option>
                  ))}
                </select>
              </label>
            ) : (
              <div className="sessionTenant">
                <span>Tenant da sessão</span>
                <strong>{tenantId ? `${tenantId.slice(0, 8)}...` : 'indisponível'}</strong>
              </div>
            )}
            <button type="button" onClick={loadAll} disabled={!ready || loading}>
              Atualizar
            </button>
          </div>
        </header>

        {notice && <div className="notice">{notice}</div>}
        {error && <div className="errorBox">{error}</div>}
        {!ready && (
          <div className="empty">
            {IS_DEVELOPMENT
              ? 'Informe Tenant ID e User ID, ou um JWT Bearer, para carregar os dados operacionais.'
              : 'Sessão autenticada indisponível para carregar os dados operacionais.'}
          </div>
        )}

        {ready && active === 'Dashboard' && (
          <Dashboard dashboard={dashboard} filters={filters} setFilters={setFilters} loading={loading} />
        )}
        {ready && active === 'Boletos' && (
          <BoletosView
            boletos={filteredBoletos}
            customers={customers}
            providers={providers}
            customerById={customerById}
            providerById={providerById}
            filters={filters}
            setFilters={setFilters}
            selectedBoleto={selectedBoleto}
            setSelectedBoleto={setSelectedBoleto}
            emitBoleto={emitBoleto}
          />
        )}
        {ready && active === 'Clientes' && (
          <ClientesView
            customers={filteredCustomers}
            blockedDocuments={blockedDocuments}
            filters={filters}
            setFilters={setFilters}
            form={customerForm}
            setForm={setCustomerForm}
            saveCustomer={saveCustomer}
            selectedCustomer={selectedCustomer}
            setSelectedCustomer={setSelectedCustomer}
            setActive={setActive}
            setComplianceSearch={(document) =>
              setFilters((current) => ({ ...current, complianceSearch: document || '', complianceActive: '' }))
            }
          />
        )}
        {ready && active === 'Providers' && (
          <ProvidersView
            providers={providers}
            form={providerForm}
            setForm={setProviderForm}
            saveProvider={saveProvider}
            selectedProvider={selectedProvider}
            setSelectedProvider={setSelectedProvider}
          />
        )}
        {ready && active === 'Compliance' && (
          <ComplianceView
            blacklist={filteredBlacklist}
            filters={filters}
            setFilters={setFilters}
            form={blacklistForm}
            setForm={setBlacklistForm}
            saveBlacklistEntry={saveBlacklistEntry}
            blacklistAction={blacklistAction}
            deleteBlacklistEntry={deleteBlacklistEntry}
          />
        )}
        {ready && active === 'Usuários' && <UsersView users={users} />}
        {ready && active === 'Configurações' && (
          <SettingsView baseUrl={baseUrl} tenantId={tenantId} providers={providers} />
        )}
      </section>

      <style jsx>{styles}</style>
    </main>
  )
}

function Dashboard({ dashboard, filters, setFilters, loading }) {
  const metrics = [
    ['Total de boletos', dashboard?.total_boletos],
    ['Emitidos', dashboard?.boletos_emitidos],
    ['Processando', dashboard?.boletos_em_processamento],
    ['Pagos', dashboard?.boletos_pagos],
    ['Vencidos', dashboard?.boletos_vencidos],
    ['Cancelados', dashboard?.boletos_cancelados],
    ['Com falha', dashboard?.boletos_com_falha],
    ['Valor emitido', fmtCurrency(dashboard?.valor_total_emitido)],
  ]
  return (
    <>
      <div className="toolbar">
        <label>
          De
          <input
            type="date"
            value={filters.periodFrom}
            onChange={(event) => setFilters((current) => ({ ...current, periodFrom: event.target.value }))}
          />
        </label>
        <label>
          Até
          <input
            type="date"
            value={filters.periodTo}
            onChange={(event) => setFilters((current) => ({ ...current, periodTo: event.target.value }))}
          />
        </label>
      </div>
      <section className="metricGrid">
        {metrics.map(([label, value]) => (
          <article className="metric" key={label}>
            <span>{label}</span>
            <strong>{loading ? '...' : value ?? 0}</strong>
          </article>
        ))}
      </section>
      <section className="panel">
        <h2>Distribuição por status</h2>
        <div className="statusBars">
          {Object.entries(dashboard?.by_status || {}).map(([status, count]) => (
            <div className="statusLine" key={status}>
              <span>{statusLabels[status] || status}</span>
              <strong>{String(count)}</strong>
            </div>
          ))}
          {!Object.keys(dashboard?.by_status || {}).length && <p className="muted">Sem boletos no período.</p>}
        </div>
      </section>
    </>
  )
}

function BoletosView({
  boletos,
  customers,
  providers,
  customerById,
  providerById,
  filters,
  setFilters,
  selectedBoleto,
  setSelectedBoleto,
  emitBoleto,
}) {
  return (
    <>
      <div className="toolbar">
        <input
          placeholder="Buscar por nosso número, linha digitável, código de barras, external ID ou CPF/CNPJ"
          value={filters.boletoSearch}
          onChange={(event) => setFilters((current) => ({ ...current, boletoSearch: event.target.value }))}
        />
        <select
          value={filters.boletoStatus}
          onChange={(event) => setFilters((current) => ({ ...current, boletoStatus: event.target.value }))}
        >
          <option value="">Todos os status</option>
          {Object.keys(statusLabels).map((status) => (
            <option value={status} key={status}>
              {statusLabels[status]}
            </option>
          ))}
        </select>
        <select
          value={filters.boletoProvider}
          onChange={(event) => setFilters((current) => ({ ...current, boletoProvider: event.target.value }))}
        >
          <option value="">Todos os providers</option>
          {providers.map((provider) => (
            <option value={provider.id} key={provider.id}>
              {provider.name}
            </option>
          ))}
        </select>
        <select
          value={filters.boletoCustomer}
          onChange={(event) => setFilters((current) => ({ ...current, boletoCustomer: event.target.value }))}
        >
          <option value="">Todos os clientes</option>
          {customers.map((customer) => (
            <option value={customer.id} key={customer.id}>
              {customer.name}
            </option>
          ))}
        </select>
      </div>
      <DataTable
        columns={['ID', 'Cliente', 'CPF/CNPJ', 'Valor', 'Vencimento', 'Status', 'Provider', 'Nosso Número', 'Criação', 'Ações']}
        rows={boletos.map((boleto) => {
          const customer = customerById[boleto.customer_id]
          const provider = boleto.provider_id ? providerById[boleto.provider_id] : null
          const canEmit = ['CREATED', 'FAILED'].includes(boleto.status)
          return [
            shortId(boleto.id),
            customer?.name || '-',
            customer?.document || '-',
            fmtCurrency(boleto.amount_cents),
            fmtDate(boleto.due_date),
            <StatusBadge key="status" status={boleto.status} />,
            provider?.name || '-',
            boleto.our_number || '-',
            fmtDate(boleto.created_at),
            <div className="rowActions" key="actions">
              <button type="button" onClick={() => setSelectedBoleto(boleto)}>
                Detalhes
              </button>
              <button type="button" disabled={!canEmit} onClick={() => emitBoleto(boleto)}>
                Emitir
              </button>
              <button type="button" disabled>
                Consultar
              </button>
              <button type="button" disabled>
                Baixar
              </button>
            </div>,
          ]
        })}
      />
      {selectedBoleto && (
        <DetailPanel title="Detalhes do boleto" onClose={() => setSelectedBoleto(null)}>
          <KeyValues
            values={{
              ID: selectedBoleto.id,
              Cliente: customerById[selectedBoleto.customer_id]?.name || '-',
              Provider: selectedBoleto.provider_id ? providerById[selectedBoleto.provider_id]?.name : '-',
              Valor: fmtCurrency(selectedBoleto.amount_cents),
              Vencimento: fmtDate(selectedBoleto.due_date),
              Status: selectedBoleto.status,
              'Nosso Número': selectedBoleto.our_number || '-',
              'Linha Digitável': selectedBoleto.digitable_line || '-',
              'Código de Barras': selectedBoleto.barcode || '-',
              'External ID': selectedBoleto.external_id || '-',
              'Data de emissão': fmtDate(selectedBoleto.issued_at),
              'Data de criação': fmtDate(selectedBoleto.created_at),
            }}
          />
          <Timeline status={selectedBoleto.status} />
        </DetailPanel>
      )}
    </>
  )
}

function ClientesView({
  customers,
  blockedDocuments,
  filters,
  setFilters,
  form,
  setForm,
  saveCustomer,
  selectedCustomer,
  setSelectedCustomer,
  setActive,
  setComplianceSearch,
}) {
  return (
    <div className="split">
      <section>
        <div className="toolbar">
          <input
            placeholder="Buscar por nome ou CPF/CNPJ"
            value={filters.customerSearch}
            onChange={(event) => setFilters((current) => ({ ...current, customerSearch: event.target.value }))}
          />
        </div>
        <DataTable
          columns={['Nome', 'CPF/CNPJ', 'Email', 'Cidade', 'Status', 'Compliance', 'Ações']}
          rows={customers.map((customer) => {
            const blocked = blockedDocuments.has(customer.document)
            return [
              customer.name,
              customer.document || '-',
              customer.email || '-',
              customer.city || '-',
              customer.status,
              blocked ? <span className="blockedBadge">BLOQUEADO PARA EMISSÃO</span> : '-',
              <div className="rowActions" key="actions">
                <button type="button" onClick={() => setSelectedCustomer(customer)}>
                  Detalhes
                </button>
                <button type="button" onClick={() => setForm(customerToForm(customer))}>
                  Editar
                </button>
                {blocked && (
                  <button
                    type="button"
                    onClick={() => {
                      setComplianceSearch(customer.document)
                      setActive('Compliance')
                    }}
                  >
                    Gerenciar bloqueio
                  </button>
                )}
              </div>,
            ]
          })}
        />
      </section>
      <FormPanel title={form.id ? 'Editar cliente' : 'Novo cliente'} onSubmit={saveCustomer}>
        {Object.keys(emptyCustomer()).map((field) => (
          field !== 'id' && (
            <label key={field}>
              {customerLabels[field]}
              <input value={form[field] || ''} onChange={(event) => setForm({ ...form, [field]: event.target.value })} />
            </label>
          )
        ))}
        <button type="submit">{form.id ? 'Salvar cliente' : 'Cadastrar cliente'}</button>
      </FormPanel>
      {selectedCustomer && (
        <DetailPanel title="Detalhes do cliente" onClose={() => setSelectedCustomer(null)}>
          <KeyValues values={selectedCustomer} />
        </DetailPanel>
      )}
    </div>
  )
}

function ProvidersView({ providers, form, setForm, saveProvider, selectedProvider, setSelectedProvider }) {
  return (
    <div className="split">
      <section>
        <DataTable
          columns={['Nome', 'Status', 'Health', 'Tipo', 'Cadastro', 'Ações']}
          rows={providers.map((provider) => [
            provider.name,
            provider.status,
            <span className="health" key="health">Configurar health check</span>,
            provider.name?.toLowerCase().includes('moncalieri') ? 'Moncalieri Capital' : 'Mock/Outro',
            fmtDate(provider.created_at),
            <button type="button" key="details" onClick={() => setSelectedProvider(provider)}>
              Detalhes
            </button>,
          ])}
        />
      </section>
      <FormPanel title="Novo provider" onSubmit={saveProvider}>
        <label>
          Nome
          <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
        </label>
        <label>
          Status
          <select value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
            <option value="ACTIVE">ACTIVE</option>
            <option value="INACTIVE">INACTIVE</option>
          </select>
        </label>
        <label>
          Configuração
          <textarea
            value={form.config}
            placeholder="JSON de configuração. Credenciais serão mascaradas nas consultas."
            onChange={(event) => setForm({ ...form, config: event.target.value })}
          />
        </label>
        <button type="submit">Cadastrar provider</button>
      </FormPanel>
      {selectedProvider && (
        <DetailPanel title="Detalhes do provider" onClose={() => setSelectedProvider(null)}>
          <KeyValues
            values={{
              ID: selectedProvider.id,
              Nome: selectedProvider.name,
              Status: selectedProvider.status,
              Config: selectedProvider.config || '***',
              Cadastro: fmtDate(selectedProvider.created_at),
            }}
          />
          <p className="muted">API keys e credenciais não são exibidas em texto aberto.</p>
        </DetailPanel>
      )}
    </div>
  )
}

function ComplianceView({
  blacklist,
  filters,
  setFilters,
  form,
  setForm,
  saveBlacklistEntry,
  blacklistAction,
  deleteBlacklistEntry,
}) {
  return (
    <div className="split">
      <section>
        <div className="toolbar">
          <input
            placeholder="Buscar por CPF/CNPJ ou nome"
            value={filters.complianceSearch}
            onChange={(event) => setFilters((current) => ({ ...current, complianceSearch: event.target.value }))}
          />
          <select
            value={filters.complianceActive}
            onChange={(event) => setFilters((current) => ({ ...current, complianceActive: event.target.value }))}
          >
            <option value="">Ativos e inativos</option>
            <option value="true">Somente ativos</option>
            <option value="false">Somente inativos</option>
          </select>
        </div>
        <DataTable
          columns={['Documento', 'Nome', 'Motivo', 'Origem', 'Data', 'Status', 'Ações']}
          rows={blacklist.map((entry) => [
            entry.document,
            entry.name || '-',
            entry.reason || '-',
            entry.source,
            fmtDate(entry.created_at),
            entry.active ? <span className="blockedBadge">Ativo</span> : <span className="muted">Inativo</span>,
            <div className="rowActions" key="actions">
              <button type="button" onClick={() => blacklistAction(entry, entry.active ? 'unblock' : 'block')}>
                {entry.active ? 'Desbloquear' : 'Bloquear'}
              </button>
              <button type="button" onClick={() => deleteBlacklistEntry(entry)}>
                Excluir
              </button>
            </div>,
          ])}
        />
      </section>
      <FormPanel title="Novo bloqueio" onSubmit={saveBlacklistEntry}>
        <label>
          CPF/CNPJ
          <input value={form.document} onChange={(event) => setForm({ ...form, document: event.target.value })} />
        </label>
        <label>
          Nome
          <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
        </label>
        <label>
          Motivo
          <input value={form.reason} onChange={(event) => setForm({ ...form, reason: event.target.value })} />
        </label>
        <label>
          Origem
          <select value={form.source} onChange={(event) => setForm({ ...form, source: event.target.value })}>
            <option value="MANUAL">MANUAL</option>
            <option value="API">API</option>
            <option value="IMPORT">IMPORT</option>
          </select>
        </label>
        <label>
          Observações
          <textarea value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} />
        </label>
        <button type="submit">Cadastrar bloqueio</button>
      </FormPanel>
    </div>
  )
}

function UsersView({ users }) {
  return (
    <DataTable
      columns={['Nome', 'Email', 'Status', 'Cadastro']}
      rows={users.map((user) => [user.name || '-', user.email, user.status, fmtDate(user.created_at)])}
    />
  )
}

function SettingsView({ baseUrl, tenantId, providers }) {
  return (
    <section className="panel">
      <h2>Configurações</h2>
      <KeyValues
        values={{
          'Base URL': baseUrl,
          'Tenant ativo': tenantId,
          'Providers configurados': providers.length,
          'Credenciais sensíveis': 'Mascaradas nas consultas e fora de logs da interface.',
        }}
      />
    </section>
  )
}

function DataTable({ columns, rows }) {
  return (
    <div className="tableWrap">
      <table>
        <thead>
          <tr>{columns.map((column) => <th key={column}>{column}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={index}>{row.map((cell, cellIndex) => <td key={cellIndex}>{cell}</td>)}</tr>
          ))}
          {!rows.length && (
            <tr>
              <td colSpan={columns.length} className="emptyCell">Nenhum registro encontrado.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

function FormPanel({ title, onSubmit, children }) {
  return (
    <form className="formPanel" onSubmit={onSubmit}>
      <h2>{title}</h2>
      {children}
    </form>
  )
}

function DetailPanel({ title, onClose, children }) {
  return (
    <aside className="detailPanel">
      <div className="detailHeader">
        <h2>{title}</h2>
        <button type="button" onClick={onClose}>Fechar</button>
      </div>
      {children}
    </aside>
  )
}

function KeyValues({ values }) {
  return (
    <dl className="keyValues">
      {Object.entries(values || {}).map(([key, value]) => (
        <div key={key}>
          <dt>{key}</dt>
          <dd>{String(value ?? '-')}</dd>
        </div>
      ))}
    </dl>
  )
}

function StatusBadge({ status }) {
  return <span className={`statusBadge ${String(status || '').toLowerCase()}`}>{statusLabels[status] || status}</span>
}

function Timeline({ status }) {
  const steps = ['CREATED', 'PROCESSING', 'ISSUED', 'PAID']
  const activeIndex = Math.max(0, steps.indexOf(status))
  return (
    <div className="timeline">
      {steps.map((step, index) => (
        <span className={index <= activeIndex ? 'stepDone' : ''} key={step}>
          {statusLabels[step]}
        </span>
      ))}
    </div>
  )
}

function emptyCustomer() {
  return {
    id: '',
    name: '',
    document: '',
    email: '',
    address: '',
    number: '',
    complement: '',
    district: '',
    city: '',
    state: '',
    postal_code: '',
  }
}

function customerToForm(customer) {
  return { ...emptyCustomer(), ...customer }
}

function shortId(id) {
  return id ? `${id.slice(0, 8)}...` : '-'
}

const customerLabels = {
  name: 'Nome',
  document: 'CPF/CNPJ',
  email: 'Email',
  address: 'Endereço',
  number: 'Número',
  complement: 'Complemento',
  district: 'Bairro',
  city: 'Cidade',
  state: 'UF',
  postal_code: 'CEP',
}

const styles = `
  :global(body) {
    margin: 0;
    background: #f4f6f8;
    color: #1f2933;
    font-family: Inter, Arial, sans-serif;
  }
  * { box-sizing: border-box; }
  .shell { min-height: 100vh; display: grid; grid-template-columns: 260px 1fr; }
  .sidebar { background: #111827; color: #f9fafb; padding: 20px 14px; position: sticky; top: 0; height: 100vh; }
  .brand { display: flex; align-items: center; gap: 12px; padding: 8px 8px 22px; }
  .brandMark { width: 42px; height: 42px; display: grid; place-items: center; background: #0ea5e9; color: white; border-radius: 8px; font-weight: 800; }
  .brand small { display: block; color: #9ca3af; margin-top: 4px; }
  nav { display: grid; gap: 6px; }
  nav button { width: 100%; border: 0; background: transparent; color: #d1d5db; text-align: left; padding: 11px 12px; border-radius: 6px; cursor: pointer; font-weight: 600; }
  nav button:hover, .navActive { background: #1f2937; color: white; }
  .workspace { min-width: 0; padding: 24px; }
  .topbar { display: flex; justify-content: space-between; gap: 20px; align-items: flex-start; margin-bottom: 20px; }
  h1, h2 { margin: 0; }
  h1 { font-size: 28px; }
  h2 { font-size: 18px; margin-bottom: 14px; }
  p { margin: 6px 0 0; color: #64748b; }
  .tenantControls, .toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: end; }
  label { display: grid; gap: 5px; font-size: 12px; color: #52606d; font-weight: 700; }
  input, select, textarea { min-height: 38px; border: 1px solid #cbd5e1; border-radius: 6px; padding: 8px 10px; background: white; color: #1f2933; font: inherit; min-width: 180px; }
  textarea { min-height: 92px; resize: vertical; }
  button { border: 1px solid #0f766e; background: #0f766e; color: white; border-radius: 6px; min-height: 36px; padding: 8px 12px; cursor: pointer; font-weight: 700; }
  button:disabled { opacity: .45; cursor: not-allowed; background: #94a3b8; border-color: #94a3b8; }
  .notice, .errorBox, .empty { padding: 12px 14px; border-radius: 6px; margin-bottom: 16px; }
  .notice { background: #dcfce7; color: #166534; }
  .errorBox { background: #fee2e2; color: #991b1b; }
  .empty { background: #e0f2fe; color: #075985; }
  .metricGrid { display: grid; grid-template-columns: repeat(4, minmax(150px, 1fr)); gap: 12px; margin: 16px 0; }
  .metric, .panel, .formPanel, .detailPanel { background: white; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; }
  .metric span { display: block; color: #64748b; font-size: 13px; margin-bottom: 8px; }
  .metric strong { font-size: 24px; }
  .statusBars { display: grid; gap: 8px; }
  .statusLine { display: flex; justify-content: space-between; border-bottom: 1px solid #e2e8f0; padding: 8px 0; }
  .tableWrap { overflow-x: auto; background: white; border: 1px solid #e2e8f0; border-radius: 8px; margin-top: 14px; }
  table { width: 100%; border-collapse: collapse; min-width: 900px; }
  th, td { padding: 11px 12px; border-bottom: 1px solid #e2e8f0; text-align: left; vertical-align: top; font-size: 14px; }
  th { background: #f8fafc; color: #475569; font-size: 12px; text-transform: uppercase; }
  .emptyCell { text-align: center; color: #64748b; padding: 28px; }
  .rowActions { display: flex; gap: 6px; flex-wrap: wrap; }
  .rowActions button { min-height: 30px; padding: 5px 8px; font-size: 12px; }
  .split { display: grid; grid-template-columns: minmax(0, 1fr) 340px; gap: 16px; align-items: start; }
  .formPanel { display: grid; gap: 10px; position: sticky; top: 18px; }
  .detailPanel { margin-top: 14px; }
  .detailHeader { display: flex; justify-content: space-between; gap: 12px; align-items: center; }
  .keyValues { display: grid; gap: 8px; margin: 0; }
  .keyValues div { display: grid; grid-template-columns: 180px 1fr; gap: 12px; border-bottom: 1px solid #e2e8f0; padding: 8px 0; }
  dt { color: #64748b; font-weight: 700; }
  dd { margin: 0; word-break: break-word; }
  .statusBadge, .blockedBadge, .health { display: inline-block; padding: 4px 8px; border-radius: 6px; font-size: 12px; font-weight: 800; background: #e2e8f0; color: #334155; }
  .statusBadge.issued, .statusBadge.paid { background: #dcfce7; color: #166534; }
  .statusBadge.processing { background: #fef3c7; color: #92400e; }
  .statusBadge.failed, .blockedBadge { background: #fee2e2; color: #991b1b; }
  .muted { color: #64748b; }
  .timeline { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-top: 16px; }
  .timeline span { background: #e2e8f0; padding: 8px; border-radius: 6px; text-align: center; font-size: 12px; font-weight: 800; }
  .timeline .stepDone { background: #ccfbf1; color: #115e59; }
  @media (max-width: 980px) {
    .shell { grid-template-columns: 1fr; }
    .sidebar { position: static; height: auto; }
    .topbar, .split { grid-template-columns: 1fr; display: grid; }
    .metricGrid { grid-template-columns: repeat(2, minmax(140px, 1fr)); }
    .formPanel { position: static; }
  }
  @media (max-width: 640px) {
    .workspace { padding: 14px; }
    .metricGrid { grid-template-columns: 1fr; }
    input, select, textarea { min-width: 100%; }
    .tenantControls, .toolbar { display: grid; }
    .keyValues div { grid-template-columns: 1fr; }
  }
`
