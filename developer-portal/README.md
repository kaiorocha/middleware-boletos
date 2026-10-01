# Developer Portal

Portal estático da API pública, renderizado com Scalar empacotado localmente. O build lê `docs/openapi/middleware-boletos-public.openapi.json`, injeta os servers somente na cópia de `dist/openapi.json` e não modifica o contrato-fonte.

## Desenvolvimento local

```bash
npm ci
PUBLIC_API_PRODUCTION_URL=https://api.example.com npm run build
python3 -m http.server 8000 --directory dist
```

Abra `http://localhost:8000`. `PUBLIC_API_HML_URL` é opcional; não o configure se homologação não deve ser exposta. `PUBLIC_DOCS_URL` é usado pelo workflow para o smoke test.

O contrato interno completo continua em `docs/openapi/middleware-boletos-etapa-5.openapi.json`; somente `middleware-boletos-public.openapi.json` é copiado para o portal e publicado. O botão de cliente interativo do Scalar fica oculto porque CORS para o domínio de documentação não foi ampliado nesta entrega.

Para rollback, execute manualmente o workflow de publicação informando um commit/tag anterior em `ref`. O versionamento do S3 preserva o histórico, mas a republicação de uma referência conhecida é o procedimento operacional preferido.
