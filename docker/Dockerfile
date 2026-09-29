# Stage 1: Build UI (SvelteKit)
FROM docker.io/node:22-alpine AS ui-builder

WORKDIR /build_ui

# Corepack uses the pnpm version pinned in package.json.
RUN corepack enable

# Copy package files
COPY web/greenwave-ui/package.json web/greenwave-ui/pnpm-lock.yaml web/greenwave-ui/pnpm-workspace.yaml web/greenwave-ui/.npmrc ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY web/greenwave-ui/ ./

# Build static files
RUN pnpm build

# Stage 2: Build Go application
FROM docker.io/golang:1.23-alpine AS go-builder

WORKDIR /build_app

COPY go.mod go.sum ./

RUN go mod download

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY ./*.go ./
COPY ./color ./color
COPY ./geom ./geom
COPY ./greeninterval ./greeninterval
COPY ./junction ./junction
COPY ./maxpressure ./maxpressure
COPY ./app ./app
COPY ./cmd/greenwave ./cmd/greenwave

# Regenerate Swagger documentation just in case
RUN swag init -g cmd/greenwave/main.go --output ./app/rest/docs --outputTypes json

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o greenwave-server-app ./cmd/greenwave/main.go

# Stage 3: Final image
FROM scratch

WORKDIR /app

# Copy Go binary
COPY --from=go-builder /build_app/greenwave-server-app ./

# Copy Swagger docs
COPY --from=go-builder /build_app/app/rest/docs ./app/rest/docs

# Copy UI static files
COPY --from=ui-builder /build_ui/build ./static

# Environment variables
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=36000
ENV SERVER_MAIN_PATH=api
ENV SERVER_STARTUP_MESSAGE=true
ENV USE_CORS=true
ENV DOCS_PATH=/app/app/rest/docs
ENV STATIC_PATH=/app/static

EXPOSE 36000

ENTRYPOINT ["./greenwave-server-app"]
