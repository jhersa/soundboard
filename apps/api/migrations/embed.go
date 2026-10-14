// Package migrations embebe los .sql del esquema. Un cambio de esquema siempre es un archivo nuevo.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
