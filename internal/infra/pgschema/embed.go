package pgschema

import "embed"

//go:embed postgres_platform.sql postgres_sessions.sql postgres_memory.sql postgres_eval.sql
var Files embed.FS
