package backend

import "embed"

//go:embed migrations/*.sql seed/*.json web/*
var Assets embed.FS
