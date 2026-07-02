# Arquitetura — Etapa 1

Visão geral

O projeto "middleware-boletos" é dividido em três camadas iniciais:

- backend/ (Go): API REST mínima — responsável pela lógica do middleware e integrações com serviços bancários futuramente.
- frontend/ (Next.js): interface inicial para testes e demos.
- infra/ (Docker Compose): composição para desenvolvimento local com PostgreSQL.

Escopo da Etapa 1

- Criar scaffolds para backend e frontend
- Fornecer um docker-compose para subir Postgres, backend e frontend localmente
- Documentação inicial explicando como executar o ambiente

Decisões técnicas iniciais

- Linguagem backend: Go (binaries simples, bom para serviços)
- Frontend: Next.js (React) para facilitar evolução para páginas isomórficas
- Banco: PostgreSQL em container para desenvolvimento local
- Comunicação inicial: HTTP REST. Avaliar gRPC no futuro se necessário

Próximos passos

- Validar contratos entre frontend/backend (endpoints mínimos)
- Modelagem do domínio e design de banco para Etapa 2
- Adicionar scripts de desenvolvimento e Makefile

