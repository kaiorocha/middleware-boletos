# Etapa 2

## Unicidade por tenant

A Etapa 2 aplica regras de unicidade por tenant para evitar duplicidade entre registros ativos sem quebrar o isolamento multi-tenant. A mesma informação pode existir em tenants diferentes.

Regras implementadas:

- Usuários: não permite repetir e-mail dentro do mesmo tenant. A comparação é case-insensitive.
- Clientes: não permite repetir documento dentro do mesmo tenant. O documento é normalizado sem máscara; documentos vazios ou nulos não bloqueiam duplicidade.
- Provedores: não permite repetir nome dentro do mesmo tenant. A comparação é case-insensitive.
- Boletos: não permite repetir `external_id` ou `our_number` dentro do mesmo tenant quando informados. Valores vazios são tratados como nulos.

As violações de unicidade retornam HTTP `409 Conflict`:

```json
{
  "error": {
    "code": "DUPLICATE_RESOURCE",
    "message": "Já existe um recurso com estes dados neste tenant."
  }
}
```

No banco de dados, as garantias são feitas por índices únicos parciais criados pela migration `009_add_unique_constraints_per_tenant.sql`.
