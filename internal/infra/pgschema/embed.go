package pgschema

import "embed"

//go:embed postgres_platform.sql postgres_sessions.sql postgres_memory.sql
var Files embed.FS
