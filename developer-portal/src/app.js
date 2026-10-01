import { createApiReference } from '@scalar/api-reference'

createApiReference('#api-reference', {
  url: '/openapi.json',
  theme: 'default',
  layout: 'modern',
  hideModels: false,
  hideClientButton: true,
  telemetry: false,
})
