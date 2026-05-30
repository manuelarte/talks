package main

import "embed"

//go:embed openapi.yaml
var OpenAPI []byte

//go:embed static/swagger-ui/*
var SwaggerUI embed.FS

//go:embed resources/*
var ResourcesFolder embed.FS
