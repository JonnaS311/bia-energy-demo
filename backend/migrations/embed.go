// Package migrations embebe los archivos SQL en el binario (spec backend RF-B-01).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
