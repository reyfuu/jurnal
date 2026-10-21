package main

import (
	"strings"
	"testing"
)

// Tabel Primbon wajib ada di schema: server berhenti (log.Fatalf) saat boot bila tabelnya
// tidak ada. Pernah hilang saat menyelesaikan konflik merge schema.sql.
func TestSchema_TabelPrimbonAda(t *testing.T) {
	for _, tabel := range []string{"primbon_bookmark", "primbon_catatan"} {
		if !strings.Contains(schemaSQL, "CREATE TABLE IF NOT EXISTS "+tabel+" (") {
			t.Errorf("schema.sql tidak membuat tabel %s", tabel)
		}
	}
}
