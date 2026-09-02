package main

import "embed"

//go:embed openapi.yaml
var OpenAPI []byte

//go:embed resources/*
var ResourcesFolder embed.FS
