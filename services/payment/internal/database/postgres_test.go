package database

import "testing"

func TestInvalidDatabaseURL(t *testing.T) {
	_, err := Connect("postgres://test:private-fixture@localhost/payment?sslmode=invalid")
	if err == nil || err.Error() != "could not open PostgreSQL with the supplied DATABASE_URL" {
		t.Fatalf("invalid DSN must fail without exposing its contents: %v", err)
	}
}

func TestConnectDoesNotRequireAvailableDatabase(t *testing.T) {
	db, err := Connect("postgres://localhost:1/payment-test?sslmode=disable")
	if err != nil {
		t.Fatalf("valid DSN must allow startup without a reachable database: %v", err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
}
