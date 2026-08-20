import { build } from 'esbuild'
import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const repo = resolve(root, '..')
const dist = resolve(root, 'dist')
const sourceSpec = resolve(repo, 'docs/openapi/middleware-boletos-public.openapi.json')
const productionUrl = process.env.PUBLIC_API_PRODUCTION_URL || 'https://api.example.com'
const hmlUrl = process.env.PUBLIC_API_HML_URL

await mkdir(resolve(dist, 'assets'), { recursive: true })
const spec = JSON.parse(await readFile(sourceSpec, 'utf8'))
spec.servers = [{ url: productionUrl, description: 'Production' }]
if (hmlUrl) spec.servers.push({ url: hmlUrl, description: 'Homologation' })

await Promise.all([
  cp(resolve(root, 'index.html'), resolve(dist, 'index.html')),
  cp(resolve(root, 'assets/portal.css'), resolve(dist, 'assets/portal.css')),
  writeFile(resolve(dist, 'openapi.json'), `${JSON.stringify(spec, null, 2)}\n`),
  build({ entryPoints: [resolve(root, 'src/app.js')], bundle: true, minify: true, format: 'esm', outfile: resolve(dist, 'assets/portal.js') }),
])

console.log(`Portal v${spec.info.version} built for ${productionUrl}`)
