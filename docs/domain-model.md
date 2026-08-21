# Domain Model

Este documento descreve as entidades primárias da Etapa 1, seus propósitos, atributos principais e relacionamentos esperados.

## Tenant
- Propósito: Representa uma organização/cliente que utiliza a plataforma (multi-tenant).
- Principais atributos: ID, Name, OwnerID, CreatedAt, UpdatedAt.
- Relacionamentos: possui Users, Customers, Providers, Boletos e AuditLogs pertencentes ao Tenant.

## User
- Propósito: Representa um usuário humano ou sistema que opera dentro de um Tenant.
- Principais atributos: ID, TenantID, Email, Name, CreatedAt, UpdatedAt.
- Relacionamentos: pertence a um Tenant; pode gerar AuditLogs; pode criar/gerenciar Customers e Boletos.

## Customer
- Propósito: Representa o cedente/beneficiário associado a um Tenant (entidade que emite ou recebe pagamentos).
- Principais atributos: ID, TenantID, Name, Document, CreatedAt, UpdatedAt.
- Relacionamentos: pertence a um Tenant; está associado a Boletos (um Customer pode ter muitos Boletos).

## Provider
- Propósito: Representa um provedor bancário ou parceiro de integração (ex.: banco, gateway).
- Principais atributos: ID, TenantID, Name, Config, CreatedAt, UpdatedAt.
- Relacionamentos: pertence a um Tenant; pode ser referenciado por Boletos; pode enviar WebhookEvents.

## Boleto
- Propósito: Representa um título de cobrança (boleto) gerenciado pela plataforma.
- Principais atributos: ID, TenantID, CustomerID, ProviderID, AmountCents, DueDate, Barcode, OurNumber, CreatedAt, UpdatedAt.
- Relacionamentos: pertence a um Tenant; refere-se a um Customer e opcionalmente a um Provider; pode gerar WebhookEvents e AuditLogs.

## WebhookEvent
- Propósito: Representa eventos recebidos de provedores ou enviados a clientes (atualizações de status, conciliações etc.).
- Principais atributos: ID, TenantID, Type, Payload, CreatedAt.
- Relacionamentos: vinculado a um Tenant; referenciará entidades como Boleto ou Provider via payload quando aplicável.

## AuditLog
- Propósito: Registrar ações audíveis na plataforma para compliance e rastreabilidade.
- Principais atributos: ID, TenantID, UserID, Action, Metadata, CreatedAt.
- Relacionamentos: pertence a um Tenant; opcionalmente ligado a um User; pode referenciar Boleto/Customer/Provider via metadata.
