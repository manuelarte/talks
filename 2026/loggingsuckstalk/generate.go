package main

//go:generate go tool oapi-codegen -config cfg.yaml openapi.yaml
//go:generate go tool gospecpaths --package rest --output ./internal/infrastructure/api/rest/paths.gen.go ./openapi.yaml
//go:generate sqlc generate
