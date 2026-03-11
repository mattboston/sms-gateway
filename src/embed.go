package smsgateway

import "embed"

// WebFS embeds the built web UI assets from web/dist/.
//
//go:embed web/dist/*
var WebFS embed.FS

// MigrationsFS embeds the SQL migration files.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
