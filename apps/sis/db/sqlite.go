package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLiteDB(path string) (*sql.DB, error) {
	return OpenSQLiteDB(path)
}

func OpenSQLiteDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL",
		path,
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// SQLite works best with a single writer
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	if err := configureSQLite(db); err != nil {
		return nil, err
	}

	return db, nil
}

func configureSQLite(db *sql.DB) error {
	pragmas := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`PRAGMA temp_store = MEMORY;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
		`PRAGMA auto_vacuum = INCREMENTAL;`,
		// `PRAGMA mmap_size = 30000000000;`, // 30GB for large DBs
		`PRAGMA mmap_size = 2147483648;`, // 2GB for smaller DBs
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}

	return nil
}
