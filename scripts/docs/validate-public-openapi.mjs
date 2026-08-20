import { readFile } from 'node:fs/promises'

const path = process.argv[2] || 'docs/openapi/middleware-boletos-public.openapi.json'
const raw = await readFile(path, 'utf8')
const spec = JSON.parse(raw)
const serialized = JSON.stringify(spec)
const forbidden = [
  ['/api/v1/admin', 'rota administrativa'],
  ['BOOTSTRAP_ADMIN', 'configuração de bootstrap'],
  ['DATABASE_URL', 'configuração de banco'],
  ['JWT_SECRET', 'segredo JWT'],
  ['MONCALIERI_API_KEY', 'credencial de provider'],
  ['client_secret', 'client secret'],
]

if (!String(spec.openapi || '').startsWith('3.')) throw new Error('OpenAPI 3.x obrigatório')
if (!spec.info?.version) throw new Error('info.version obrigatório')
for (const [needle, label] of forbidden) {
  if (serialized.toLowerCase().includes(needle.toLowerCase())) throw new Error(`Conteúdo proibido: ${label}`)
}
const paths = Object.keys(spec.paths || {})
if (!paths.some(p => p.includes('/boletos'))) throw new Error('Nenhuma rota de boleto encontrada')
if (!paths.some(p => p.endsWith('/emit'))) throw new Error('Nenhuma rota de emissão encontrada')
if (paths.some(p => p.startsWith('/api/v1/admin'))) throw new Error('Endpoint admin encontrado')

console.log(`public-safe OK: ${paths.length} rotas, OpenAPI ${spec.openapi}, API ${spec.info.version}`)
