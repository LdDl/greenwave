FROM docker.io/golang:alpine AS builder

WORKDIR /build_app

COPY go.mod go.sum ./

RUN go mod download

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY ./*.go ./
COPY ./color ./color
COPY ./app ./app
COPY ./cmd/greenwave ./cmd/greenwave

# Regenerate Swagger documentation just in case
RUN swag init -g cmd/greenwave/main.go --output ./docs --outputTypes json

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o greenwave-server-app ./cmd/greenwave/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /build_app/greenwave-server-app ./
COPY --from=builder /build_app/app/rest/docs ./app/rest/docs

ENTRYPOINT ["./greenwave-server-app"]