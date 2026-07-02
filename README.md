# middleware-boletos

Visão geral

Middleware para emissão e gerenciamento de boletos (scaffold inicial — Etapa 1). Backend em Go, frontend em Next.js.

Stack

- Backend: Go 1.21
- Frontend: Next.js (React)
- Database: PostgreSQL
- Cache: Redis
- Containerization: Docker / Docker Compose

Estrutura de pastas

backend/
  cmd/api/main.go - Entrypoint do servidor
  internal/ - pacotes internos (config, domain, tenant, ...)
  pkg/ - pacotes reutilizáveis
  go.mod
frontend/ - Next.js app
infra/ - infra helpers

Pré-requisitos

- Docker e Docker Compose
- Go 1.21 (para desenvolvimento local)
- Node 18+ (para desenvolvimento local frontend)

Rodando localmente com Docker Compose

1. Copiar .env.example para .env e ajustar variáveis se necessário
2. docker-compose up --build
3. Backend disponível em http://localhost:8080
4. Frontend disponível em http://localhost:3000

Endpoints disponíveis

- GET /health — health check (retorna {"status":"ok"})

Próximos passos

- Implementar persistência e migrações
- Criar endpoints CRUD para entidades
- Adicionar autenticação e autorização (Etapa 2)

