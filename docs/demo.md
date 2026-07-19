# Demo Etapa 4

## Subir ambiente

```bash
cp .env.example .env
docker compose up --build
```

URLs locais:

- Frontend: `http://localhost:3000/login`
- Backend: `http://localhost:8080`

## Bootstrap Platform Admin

Em desenvolvimento, configure no `.env`:

```env
APP_ENV=development
JWT_SECRET=local-development-secret-change-me
JWT_ISSUER=middleware-boletos-local
JWT_AUDIENCE=middleware-boletos-api
ENABLE_ADMIN_BOOTSTRAP=false
BOOTSTRAP_ADMIN_EMAIL=admin@middleware.local
BOOTSTRAP_ADMIN_PASSWORD=ChangeMe123456!
BOOTSTRAP_ADMIN_NAME=Administrador
NEXT_PUBLIC_APP_ENV=development
NEXT_PUBLIC_DEMO_ADMIN_EMAIL=admin@middleware.local
NEXT_PUBLIC_DEMO_ADMIN_PASSWORD=ChangeMe123456!
```

O backend cria o primeiro `PLATFORM_ADMIN` somente quando ainda não existe usuário com essa role e quando as variáveis estão explicitamente preenchidas. A operação é idempotente e a senha não é registrada em logs.

Credenciais demo:

- E-mail: `admin@middleware.local`
- Senha: `ChangeMe123456!`

Use essas credenciais apenas em desenvolvimento.

`NEXT_PUBLIC_DEMO_ADMIN_PASSWORD` serve somente para preencher a tela de login em desenvolvimento/demo local. Nunca configure essa variável com senha real em produção.

## Bootstrap em produção

Por padrão, production não executa bootstrap automático mesmo que `BOOTSTRAP_ADMIN_EMAIL`, `BOOTSTRAP_ADMIN_PASSWORD` e `BOOTSTRAP_ADMIN_NAME` estejam configurados.

Procedimento controlado:

1. Configure temporariamente `ENABLE_ADMIN_BOOTSTRAP=true`.
2. Configure `BOOTSTRAP_ADMIN_EMAIL`, `BOOTSTRAP_ADMIN_PASSWORD` e `BOOTSTRAP_ADMIN_NAME`.
3. Inicie a aplicação.
4. Confirme a criação do `PLATFORM_ADMIN`.
5. Altere `ENABLE_ADMIN_BOOTSTRAP=false`.
6. Remova `BOOTSTRAP_ADMIN_PASSWORD` do ambiente.

O bootstrap continua idempotente: se já existir `PLATFORM_ADMIN`, outro usuário não é criado.

## Fluxo de apresentação

1. Acesse `http://localhost:3000/login`.
2. Faça login como `admin@middleware.local`.
3. No painel da plataforma, abra `Providers`.
4. Crie o provider `Mock` no catálogo, com config `{"delay_ms":0}`.
5. Abra `Tenants`.
6. Crie o tenant `Cliente Demonstração`.
7. Informe o administrador do tenant:
   - Nome: `Administrador Cliente`
   - E-mail: `cliente@demo.local`
   - Senha: `Cliente123456!`
8. Selecione o provider `Mock` para habilitá-lo no onboarding.
9. Faça logout.
10. Faça login como `cliente@demo.local`.
11. O painel entra no tenant autorizado pelo JWT; se houver mais de um tenant, selecione apenas entre os tenants permitidos.
12. Confira `Dashboard`, `Transações`, `Boletos`, `Clientes` e `Providers` em modo consultivo/operacional.
13. Use a collection Postman ou chamadas REST para criar customer, criar boleto e solicitar emissão via API.
14. Veja a operação refletida em `Dashboard` e `Transações` do tenant.
15. Em `Compliance`, bloqueie o CPF/CNPJ do cliente.
16. Pela API, crie novo boleto para o mesmo cliente e tente emitir.
17. Confirme retorno `CUSTOMER_BLOCKED`.

## Conceitos

- Tenant: empresa que usa a plataforma.
- User: pessoa que acessa o painel.
- Customer: pagador/sacado do boleto.
- Provider: integração bancária criada no catálogo global e habilitada por tenant.

`TENANT_ADMIN` não é customer. Ele administra recursos do tenant autorizado.
