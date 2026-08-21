import Head from 'next/head'
import { useEffect, useMemo, useState } from 'react'

const methods = ['get', 'post', 'put', 'patch', 'delete']
const methodTone = { get: 'bg-sky-50 text-sky-700 ring-sky-200', post: 'bg-emerald-50 text-emerald-700 ring-emerald-200', put: 'bg-amber-50 text-amber-700 ring-amber-200', patch: 'bg-violet-50 text-violet-700 ring-violet-200', delete: 'bg-rose-50 text-rose-700 ring-rose-200' }

function resolveRef(spec: any, value: any): any {
  if (!value?.$ref) return value
  return value.$ref.replace(/^#\//, '').split('/').reduce((current, part) => current?.[part], spec)
}

function schemaExample(spec: any, original: any, depth = 0): any {
  if (!original || depth > 5) return {}
  const schema = resolveRef(spec, original)
  if (schema.example !== undefined) return schema.example
  if (schema.type === 'array') return [schemaExample(spec, schema.items, depth + 1)]
  if (schema.type === 'object' || schema.properties) {
    return Object.fromEntries(Object.entries(schema.properties || {}).map(([key, value]) => [key, schemaExample(spec, value, depth + 1)]))
  }
  if (schema.enum?.length) return schema.enum[0]
  if (schema.format === 'email') return 'cliente@empresa.com.br'
  if (schema.format === 'uuid') return '00000000-0000-4000-8000-000000000001'
  if (schema.format === 'date') return '2026-09-01'
  if (schema.format === 'date-time') return '2026-09-01T12:00:00Z'
  if (schema.format === 'byte') return 'JVBERi0xLjQK...'
  if (schema.type === 'integer' || schema.type === 'number') return schema.minimum || 10000
  if (schema.type === 'boolean') return true
  return 'string'
}

function Icon({ children }: { children: React.ReactNode }) {
  return <span className="grid size-9 place-items-center rounded-xl bg-slate-100 text-slate-600">{children}</span>
}

export default function DocsPage() {
  const [spec, setSpec] = useState<any>(null)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [environment, setEnvironment] = useState(0)

  useEffect(() => {
    fetch('/openapi.json').then((response) => {
      if (!response.ok) throw new Error('Não foi possível carregar o contrato OpenAPI.')
      return response.json()
    }).then(setSpec).catch((err) => setError(err.message))
  }, [])

  const operations = useMemo(() => {
    if (!spec) return []
    return Object.entries(spec.paths || {}).flatMap(([path, pathItem]: [string, any]) => methods.flatMap((method) => {
      const operation = pathItem[method]
      return operation ? [{ path, method, ...operation, tag: operation.tags?.[0] || 'API' }] : []
    }))
  }, [spec])

  const filtered = operations.filter((item) => `${item.summary} ${item.path} ${item.tag}`.toLowerCase().includes(query.toLowerCase()))
  const grouped = filtered.reduce((groups, operation) => ({ ...groups, [operation.tag]: [...(groups[operation.tag] || []), operation] }), {} as Record<string, any[]>)
  const server = spec?.servers?.[environment]?.url || ''

  return <>
    <Head><title>Documentação | Giga Pagamentos</title><meta name="description" content="Documentação pública da API de Boletos da Giga Pagamentos" /></Head>
    <div className="min-h-screen bg-white text-slate-700">
      <header className="sticky top-0 z-40 border-b border-slate-200/80 bg-white/90 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-[1600px] items-center gap-4 px-5 lg:px-8">
          <a href="/docs" className="flex shrink-0 items-center gap-3 text-slate-950">
            <span className="grid size-9 place-items-center rounded-xl bg-slate-950 text-xs font-black text-white">GP</span>
            <span className="hidden font-extrabold tracking-tight sm:inline">Giga Pagamentos</span>
            <span className="hidden h-5 w-px bg-slate-200 sm:block" /><span className="hidden text-sm font-semibold text-slate-500 sm:inline">Docs</span>
          </a>
          <label className="relative mx-auto block w-full max-w-xl">
            <span className="sr-only">Buscar na documentação</span>
            <svg className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
            <input className="!min-h-10 !w-full !min-w-0 !rounded-xl !border-slate-200 !bg-slate-50 !pl-10" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Buscar endpoint, recurso ou operação..." />
          </label>
          <a href="/login" className="shrink-0 rounded-xl border border-slate-200 px-3.5 py-2 text-sm font-bold text-slate-700 hover:border-brand-200 hover:text-brand-700">Acessar painel</a>
        </div>
      </header>

      <div className="mx-auto grid max-w-[1600px] lg:grid-cols-[270px_minmax(0,1fr)]">
        <aside className="hidden h-[calc(100vh-4rem)] overflow-y-auto border-r border-slate-200 px-6 py-8 lg:sticky lg:top-16 lg:block">
          <p className="mb-3 text-[11px] font-black uppercase tracking-[.16em] text-slate-400">Comece aqui</p>
          <nav className="mb-8 grid gap-1 text-sm"><a className="rounded-lg bg-brand-50 px-3 py-2 font-bold text-brand-700" href="#visao-geral">Visão geral</a><a className="rounded-lg px-3 py-2 font-medium text-slate-500 hover:bg-slate-50 hover:text-slate-900" href="#autenticacao">Autenticação</a><a className="rounded-lg px-3 py-2 font-medium text-slate-500 hover:bg-slate-50 hover:text-slate-900" href="/openapi.json" download>Baixar OpenAPI</a></nav>
          {(Object.entries(grouped) as [string, any[]][]).map(([tag, items]) => <div key={tag} className="mb-7"><p className="mb-2 text-[11px] font-black uppercase tracking-[.16em] text-slate-400">{tag}</p><nav className="grid gap-1">{items.map((item) => <a key={`${item.method}-${item.path}`} href={`#${item.method}-${item.path.replace(/[^a-z0-9]/gi, '-')}`} className="rounded-lg px-3 py-2 text-sm font-medium text-slate-500 hover:bg-slate-50 hover:text-slate-900">{item.summary}</a>)}</nav></div>)}
        </aside>

        <main className="min-w-0 px-5 py-10 sm:px-8 lg:px-12 xl:px-16">
          {error && <div className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-800">{error}</div>}
          {!spec && !error && <div className="animate-pulse text-sm text-slate-400">Carregando documentação…</div>}
          {spec && <>
            <section id="visao-geral" className="mx-auto max-w-5xl scroll-mt-24 border-b border-slate-100 pb-14">
              <span className="mb-5 inline-flex rounded-full bg-brand-50 px-3 py-1 text-xs font-extrabold text-brand-700 ring-1 ring-brand-100">API v{spec.info.version}</span>
              <h1 className="max-w-3xl text-4xl font-black tracking-[-.04em] text-slate-950 sm:text-5xl">Integre boletos com uma API direta e previsível.</h1>
              <p className="mt-5 max-w-3xl text-lg leading-8 text-slate-500">{spec.info.description}</p>
              <div className="mt-8 grid gap-4 sm:grid-cols-3">
                {[['Crie', 'Registre boleto, valor e vencimento.'], ['Emita', 'Receba linha digitável, código de barras e PDF.'], ['Acompanhe', 'Consulte boletos, transações e bloqueios.']].map(([title, text], index) => <article key={title} className="rounded-2xl border border-slate-200 p-5"><span className="mb-4 grid size-8 place-items-center rounded-full bg-slate-950 text-xs font-black text-white">{index + 1}</span><h2 className="font-extrabold text-slate-950">{title}</h2><p className="mt-1.5 text-sm leading-6">{text}</p></article>)}
              </div>
            </section>

            <section id="autenticacao" className="mx-auto grid max-w-5xl scroll-mt-24 gap-8 border-b border-slate-100 py-14 xl:grid-cols-[1fr_1.05fr]">
              <div><Icon><svg className="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="4" y="10" width="16" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg></Icon><h2 className="mt-5 text-2xl font-black tracking-tight text-slate-950">Autenticação</h2><p className="mt-3 leading-7">Use o token do ambiente no header <code className="rounded bg-slate-100 px-1.5 py-1 text-sm text-slate-800">Authorization</code>. O token já identifica o tenant e dispensa login, tenant_id ou provider_id.</p></div>
              <Code title="Header obrigatório" value={'Authorization: Bearer giga_hml_••••••••\nContent-Type: application/json'} />
            </section>

            <section className="mx-auto max-w-5xl py-14">
              <div className="mb-10 flex flex-col justify-between gap-4 sm:flex-row sm:items-end"><div><p className="text-xs font-black uppercase tracking-[.16em] text-brand-700">Referência da API</p><h2 className="mt-2 text-3xl font-black tracking-tight text-slate-950">Endpoints</h2></div><label>Ambiente<select className="!min-w-52" value={environment} onChange={(e) => setEnvironment(Number(e.target.value))}>{spec.servers?.map((item, index) => <option key={item.url} value={index}>{item.description}</option>)}</select></label></div>
              {!filtered.length && <div className="rounded-2xl border border-dashed border-slate-300 p-10 text-center text-slate-500">Nenhum endpoint encontrado para “{query}”.</div>}
              <div className="grid gap-8">{filtered.map((operation) => {
                const requestSchema = operation.requestBody?.content?.['application/json']?.schema
                const request = requestSchema ? JSON.stringify(schemaExample(spec, requestSchema), null, 2) : ''
                const endpoint = operation.path.replace('{boletoId}', '00000000-0000-4000-8000-000000000001').replace('{blockId}', '00000000-0000-4000-8000-000000000001')
                const curl = [
                  `curl --request ${operation.method.toUpperCase()} \\`,
                  `  --url '${server}${endpoint}' \\`,
                  `  --header 'Authorization: Bearer giga_hml_••••••••'${request ? ' \\' : ''}`,
                  ...(request ? [`  --header 'Content-Type: application/json' \\`, `  --data '${request.replace(/\n/g, '')}'`] : []),
                ].join('\n')
                return <article id={`${operation.method}-${operation.path.replace(/[^a-z0-9]/gi, '-')}`} key={`${operation.method}-${operation.path}`} className="scroll-mt-24 overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-[0_24px_65px_-45px_rgba(15,23,42,.35)]">
                  <div className="grid xl:grid-cols-[1fr_1.05fr]">
                    <div className="p-6 sm:p-8"><div className="mb-4 flex flex-wrap items-center gap-3"><span className={`rounded-lg px-2.5 py-1 text-[11px] font-black uppercase ring-1 ${methodTone[operation.method]}`}>{operation.method}</span><span className="text-xs font-bold uppercase tracking-wider text-slate-400">{operation.tag}</span></div><h3 className="text-2xl font-black tracking-tight text-slate-950">{operation.summary}</h3><p className="mt-3 leading-7">{operation.description || 'Execute esta operação usando o token Bearer do tenant.'}</p><div className="mt-6 overflow-x-auto rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-sm"><span className="mr-3 font-black text-brand-700">{operation.method.toUpperCase()}</span>{operation.path}</div>{operation.parameters?.length ? <div className="mt-6"><p className="mb-2 text-xs font-black uppercase tracking-wider text-slate-400">Parâmetros</p>{operation.parameters.map((raw, index) => { const param = resolveRef(spec, raw); return <div key={`${param.name}-${index}`} className="flex items-center justify-between border-t border-slate-100 py-3 text-sm"><code className="font-bold text-slate-800">{param.name}</code><span className="text-slate-400">{param.in} · {param.schema?.format || param.schema?.type}</span></div>})}</div> : null}</div>
                    <div className="bg-slate-950 p-5 sm:p-7"><Code dark title="cURL" value={curl} />{request && <Code dark title="Body" value={request} />}</div>
                  </div>
                </article>
              })}</div>
            </section>
          </>}
        </main>
      </div>
    </div>
  </>
}

function Code({ title, value, dark = false }: { title: string, value: string, dark?: boolean }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => { await navigator.clipboard.writeText(value); setCopied(true); window.setTimeout(() => setCopied(false), 1500) }
  return <div className={`mb-4 overflow-hidden rounded-2xl border ${dark ? 'border-white/10 bg-slate-900' : 'border-slate-200 bg-slate-950'}`}><div className="flex items-center justify-between border-b border-white/10 px-4 py-2.5"><span className="text-[11px] font-black uppercase tracking-wider text-slate-400">{title}</span><button onClick={copy} className="!min-h-0 !border-0 !bg-transparent !p-1 !text-xs !text-slate-400 !shadow-none hover:!text-white">{copied ? 'Copiado' : 'Copiar'}</button></div><pre className="overflow-x-auto p-4 text-[12px] leading-6 text-slate-200"><code>{value}</code></pre></div>
}
