package main

import (
	"database/sql"
	"pajak/internal/pengetahuan"
	"testing"
	"time"
)

func TestPostgresPengetahuanUnchangedRowsNotRewritten(t *testing.T) {
	pgTest(t)
	srv, _ := serverMaster(t)
	st := pengetahuan.NewState()
	var err error
	for _, title := range []string{"A", "B"} {
		_, err = st.TambahSumber(pengetahuan.Sumber{Judul: title, Isi: "Isi pengetahuan yang diuji", Jenis: "manual"}, "Tester", time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = savePengetahuanDB(srv.db, st); err != nil {
		t.Fatal(err)
	}
	var before, after string
	if err = srv.db.QueryRow(`SELECT ctid::text FROM sumber_pengetahuan WHERE id=2`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	old := st.Clone()
	st.Sumber[0].Judul = "A baru"
	if err = savePengetahuanDB(srv.db, st, old); err != nil {
		t.Fatal(err)
	}
	if err = srv.db.QueryRow(`SELECT ctid::text FROM sumber_pengetahuan WHERE id=2`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("unchanged large source was rewritten")
	}
	old = st.Clone()
	st.Sumber = st.Sumber[:1]
	if err = savePengetahuanDB(srv.db, st, old); err != nil {
		t.Fatal(err)
	}
	if err = srv.db.QueryRow(`SELECT judul FROM sumber_pengetahuan WHERE id=2`).Scan(&after); err != sql.ErrNoRows {
		t.Fatal("deleted source remains", err)
	}
}
