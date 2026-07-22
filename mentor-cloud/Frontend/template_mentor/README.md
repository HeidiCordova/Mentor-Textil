# Mentor Monitor - Template Frontend

Sistema de monitoreo industrial construido con Vue.js 3 y Apache ECharts.

## Requisitos

- Node.js 18+
- npm o yarn

## Instalación

```bash
npm install
```

## Configuración

1. Copiar archivo de ejemplo de variables de entorno:
```bash
cp .env.example .env
```

2. Configurar URL del backend en `.env`:
```env
VITE_API_BASE_URL=http://localhost:8080/api
```

## Desarrollo

```bash
npm run dev
```

La aplicación estará disponible en `http://localhost:3000`

## Build para Producción

```bash
npm run build
```

Los archivos optimizados se generarán en la carpeta `/dist`

## Modo Mock

Para desarrollo sin backend, configurar en `.env`:
```env
VITE_USE_MOCKS=true
```

## Documentación

- Arquitectura: Ver `ARQUITECTURA_PROYECTO.md`
- Integración Backend: Ver `docs/BACKEND_INTEGRATION.md`
- Contratos API: Ver `docs/api-contracts/`

## Stack Tecnológico

- Vue.js 3 (Composition API)
- Vue Router 4
- Pinia
- Axios
- Apache ECharts
- TailwindCSS
- Vite
