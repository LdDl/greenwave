# Greenwave UI

Network editing, green-wave coordination and Max pressure in one Svelte app.

- [Network editor](NETWORK_EDITOR.md): networks, junction programs and corridors.
- [Max pressure](MAX_PRESSURE.md): traffic scenarios, control settings and results.
- [Shared project model](EDITOR_MODEL.md): data format and editor internals.

## Run locally

Use Node.js 22 (22.12 or newer) and pnpm 10.13.1, pinned in `package.json`. The API requires Go 1.23.4 or newer.

Start the API from the repository root:

```bash
go run ./cmd/greenwave --conf ./cmd/greenwave/conf.toml
```

In another terminal, also from the repository root:

```bash
cd web/greenwave-ui
corepack enable
pnpm install --frozen-lockfile
pnpm dev
```

Open <http://localhost:5173>. The UI calls the API at `http://localhost:36000`. The supplied API configuration enables CORS.

To use another API origin, run `VITE_API_URL=http://localhost:35000 pnpm dev`. This setting only affects development.

## Check and build

Run from `web/greenwave-ui`:

```bash
pnpm test
pnpm check
pnpm build
```

The static site is written to `build/`. Production requests use `/api/greenwave` and `/api/maxpressure` on the same origin. The [Docker image](../../README.md#docker) serves both the UI and API.
