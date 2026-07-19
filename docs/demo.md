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
3. No painel da plataforma, abra `Tenants`.
4. Crie o tenant `Cliente Demonstração`.
5. Informe o administrador do tenant:
   - Nome: `Administrador Cliente`
   - E-mail: `cliente@demo.local`
   - Senha: `Cliente123456!`
6. Faça logout.
7. Faça login como `cliente@demo.local`.
8. O painel entra no tenant autorizado pelo JWT; se houver mais de um tenant, selecione apenas entre os tenants permitidos.
9. Cadastre um cliente/pagador com CPF/CNPJ.
10. Cadastre o provider `Mock` com config `{"delay_ms":0}`.
11. Crie um boleto para o cliente e provider.
12. Clique em `Emitir`.
13. Veja a operação em `Dashboard` e `Transações`.
14. Em `Compliance`, bloqueie o CPF/CNPJ do cliente.
15. Crie novo boleto para o mesmo cliente.
16. Tente emitir e confirme retorno `CUSTOMER_BLOCKED`.

## Conceitos

- Tenant: empresa que usa a plataforma.
- User: pessoa que acessa o painel.
- Customer: pagador/sacado do boleto.

`TENANT_ADMIN` não é customer. Ele administra recursos do tenant autorizado.
