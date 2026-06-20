package mockdb

import (
	"database/sql"
	"testing"
	"time"
)

func TestMockDBDriverErrors(t *testing.T) {
	db, err := sql.Open("mockdb", "")
	if err != nil {
		t.Fatalf("failed to open mockdb: %v", err)
	}
	defer db.Close()

	// 1. SimError
	SimError = sql.ErrConnDone
	if _, err := db.Begin(); err == nil {
		t.Error("expected Begin error, got nil")
	}
	if _, err := db.Prepare("SELECT 1"); err == nil {
		t.Error("expected Prepare error, got nil")
	}
	if _, err := db.Exec("DELETE FROM table"); err == nil {
		t.Error("expected Exec error, got nil")
	}
	if _, err := db.Query("SELECT 1"); err == nil {
		t.Error("expected Query error, got nil")
	}
	SimError = nil

	// 2. ExecError
	ExecError = sql.ErrTxDone
	if _, err := db.Exec("DELETE FROM table"); err == nil {
		t.Error("expected Exec error via ExecError, got nil")
	}
	ExecError = nil

	// 3. NextError
	NextError = sql.ErrNoRows
	rows, err := db.Query("SELECT 1")
	if err == nil {
		if rows.Next() {
			t.Error("expected Next to return false when error happens")
		}
		if err := rows.Err(); err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows, got %v", err)
		}
		rows.Close()
	}
	NextError = nil

	// 4. SimError on Stmt.Exec and Stmt.Query
	stmt, err := db.Prepare("INSERT INTO table (id) VALUES (1)")
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	SimError = sql.ErrConnDone
	if _, err := stmt.Exec(nil); err == nil {
		t.Error("expected Exec error, got nil")
	}
	if _, err := stmt.Query(nil); err == nil {
		t.Error("expected Query error, got nil")
	}
	SimError = nil

	// 5. TriggerScanError
	TriggerScanError = true
	rowsScan, err := db.Query("SELECT * FROM events")
	if err == nil {
		for rowsScan.Next() {
			var ev, uid, et, ip, ua, reason string
			var pay []byte
			var tm, scr time.Time
			var gdpr, fraud bool
			var score float64
			_ = rowsScan.Scan(&ev, &uid, &et, &ip, &ua, &pay, &tm, &gdpr, &fraud, &score, &reason, &scr)
		}
		rowsScan.Close()
	}
	rowsScanMetric, err := db.Query("SELECT * FROM metrics_counters")
	if err == nil {
		for rowsScanMetric.Next() {
			var name string
			var val int64
			_ = rowsScanMetric.Scan(&name, &val)
		}
		rowsScanMetric.Close()
	}
	TriggerScanError = false

	// 6. FailCRMDelete and InvalidJSONPayload
	FailCRMDelete = true
	stmtCRM, _ := db.Prepare("DELETE FROM crm_contacts WHERE user_id = 1")
	if _, err := stmtCRM.Exec(nil); err == nil {
		t.Error("expected CRM delete failure, got nil")
	}
	FailCRMDelete = false

	InvalidJSONPayload = true
	rowsJSON, err := db.Query("SELECT * FROM events")
	if err == nil {
		if rowsJSON.Next() {
			var ev, uid, et, ip, ua, reason string
			var pay []byte
			var tm, scr time.Time
			var gdpr, fraud bool
			var score float64
			_ = rowsJSON.Scan(&ev, &uid, &et, &ip, &ua, &pay, &tm, &gdpr, &fraud, &score, &reason, &scr)
		}
		rowsJSON.Close()
	}
	InvalidJSONPayload = false
}
