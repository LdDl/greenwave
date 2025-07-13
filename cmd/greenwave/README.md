```bash
go run ./cmd/greenwave/main.go --conf ./cmd/greenwave/conf.toml
```

How Swagger documentation has been prepared:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/greenwave/main.go --output ./app/rest/docs --outputTypes json
```