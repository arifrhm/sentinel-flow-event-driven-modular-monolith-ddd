package mockdb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"time"
)

// SimError can be set by tests to simulate a database query/connection error.
var (
	SimError           error
	NextError          error
	CommitError        error
	FailCRMDelete      bool
	ExecError          error
	InvalidJSONPayload bool
)

func init() {
	sql.Register("mockdb", &mockDriver{})
}

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	if SimError != nil {
		return nil, SimError
	}
	return &mockStmt{query: query}, nil
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	if SimError != nil {
		return nil, SimError
	}
	return &mockTx{}, nil
}

type mockStmt struct {
	query string
}

func (s *mockStmt) Close() error { return nil }

func (s *mockStmt) NumInput() int { return -1 }

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	if SimError != nil {
		return nil, SimError
	}
	if ExecError != nil {
		return nil, ExecError
	}
	if FailCRMDelete && strings.Contains(strings.ToLower(s.query), "crm_contacts") {
		return nil, errors.New("mock crm contacts delete error")
	}
	return &mockResult{}, nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	if SimError != nil {
		return nil, SimError
	}
	return &mockRows{query: s.query}, nil
}

type mockTx struct{}

func (t *mockTx) Commit() error {
	return CommitError
}

func (t *mockTx) Rollback() error { return nil }

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }

func (r *mockResult) RowsAffected() (int64, error) { return 1, nil }

type mockRows struct {
	query string
	index int
}

func (r *mockRows) Columns() []string {
	q := strings.ToLower(r.query)
	if strings.Contains(q, "metrics_counters") {
		return []string{"metric_name", "counter_value"}
	}
	if strings.Contains(q, "crm_contacts") {
		return []string{"user_id", "email", "workflow_triggered", "sync_status", "synced_at", "retry_count"}
	}
	if strings.Contains(q, "privacy_logs") {
		return []string{"log_id", "user_id", "action", "timestamp"}
	}
	return []string{
		"event_id", "user_id", "event_type", "ip_address", "user_agent", "payload",
		"timestamp", "gdpr_consent", "is_fraudulent", "fraud_score", "fraud_reason", "screened_at",
	}
}

var TriggerScanError bool

func (r *mockRows) Close() error { return nil }

func (r *mockRows) Next(dest []driver.Value) error {
	if NextError != nil {
		return NextError
	}
	q := strings.ToLower(r.query)
	if strings.Contains(q, "metrics_counters") {
		if TriggerScanError {
			if r.index > 0 {
				return io.EOF
			}
			dest[0] = "total_received"
			dest[1] = "bad-scan-value"
			r.index++
			return nil
		}
		names := []string{"total_received", "legitimate", "fraudulent", "crm_attempts", "crm_successes", "crm_failures"}
		if r.index >= len(names) {
			return io.EOF
		}
		dest[0] = names[r.index]
		dest[1] = int64(10)
		r.index++
		return nil
	}

	if r.index > 0 {
		return io.EOF
	}
	r.index++
	if TriggerScanError {
		for i := range dest {
			dest[i] = "bad-scan-value"
		}
		return nil
	}
	if strings.Contains(q, "crm_contacts") {
		dest[0] = "usr_test"
		dest[1] = "test@example.com"
		dest[2] = "onboarding"
		dest[3] = "synced"
		dest[4] = time.Now()
		dest[5] = int64(0)
	} else if strings.Contains(q, "privacy_logs") {
		dest[0] = "log_1"
		dest[1] = "usr_test"
		dest[2] = "GDPR_PURGE"
		dest[3] = time.Now()
	} else {
		dest[0] = "ev_1"
		dest[1] = "usr_test"
		dest[2] = "signup"
		dest[3] = "1.2.3.4"
		dest[4] = "Mozilla"
		if InvalidJSONPayload {
			dest[5] = []byte("{invalid-json")
		} else {
			dest[5] = []byte(`{"email":"test@example.com"}`)
		}
		dest[6] = time.Now()
		dest[7] = true
		dest[8] = false
		dest[9] = float64(0.0)
		dest[10] = ""
		dest[11] = time.Now()
	}
	return nil
}
