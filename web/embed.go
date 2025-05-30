package web

import "embed"

//go:embed admin.html view.html assets/*
var templates embed.FS
