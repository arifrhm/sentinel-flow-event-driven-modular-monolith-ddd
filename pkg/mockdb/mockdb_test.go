package mockdb

import (
	"database/sql"
	"testing"
	"time"
)

func TestMockDBDriver(t *testing.T) {
	db, err := sql.Open("mockdb", "")
	if err != nil {
		t.Fatalf("failed to open mockdb: %v", err)
	}
	defer db.Close()

	// Test transaction interface
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Error("failed to commit transaction")
	}
	txRollback, _ := db.Begin()
	_ = txRollback.Rollback()

	// Test Exec / Result interface
	res, err := db.Exec("INSERT INTO table (id) VALUES (1)")
	if err != nil {
		t.Fatalf("failed to exec insert: %v", err)
	}
	if affected, _ := res.RowsAffected(); affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}
	if id, _ := res.LastInsertId(); id != 0 {
		t.Errorf("expected 0 last insert id, got %d", id)
	}

	// Test Query interface (metrics_counters)
	rows, err := db.Query("SELECT metric_name, counter_value FROM metrics_counters")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	cols, _ := rows.Columns()
	if len(cols) != 2 || cols[0] != "metric_name" {
		t.Errorf("unexpected columns returned: %v", cols)
	}
	for rows.Next() {
		var name string
		var val int64
		if err := rows.Scan(&name, &val); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
	}
	rows.Close()

	// Query crm_contacts
	rowsCRM, err := db.Query("SELECT * FROM crm_contacts")
	if err == nil {
		_, _ = rowsCRM.Columns()
		for rowsCRM.Next() {
			var u, em, w, s string
			var tm time.Time
			var r int64
			_ = rowsCRM.Scan(&u, &em, &w, &s, &tm, &r)
		}
		rowsCRM.Close()
	}

	// Query privacy_logs
	rowsPriv, err := db.Query("SELECT * FROM privacy_logs")
	if err == nil {
		_, _ = rowsPriv.Columns()
		for rowsPriv.Next() {
			var lid, uid, act string
			var tm time.Time
			_ = rowsPriv.Scan(&lid, &uid, &act, &tm)
		}
		rowsPriv.Close()
	}

	// Query events
	rowsEv, err := db.Query("SELECT * FROM events")
	if err == nil {
		_, _ = rowsEv.Columns()
		for rowsEv.Next() {
			var ev, uid, et, ip, ua, reason string
			var pay []byte
			var tm, scr time.Time
			var gdpr, fraud bool
			var score float64
			_ = rowsEv.Scan(&ev, &uid, &et, &ip, &ua, &pay, &tm, &gdpr, &fraud, &score, &reason, &scr)
		}
		rowsEv.Close()
	}
}
