package appctx

import "database/sql"

// AppContext holds request-scoped application dependencies
type AppContext struct {
	DB *sql.DB
	// Tx *sql.Tx        // add later if needed
	// OrgID int         // multi-tenant future
	// UserID int        // auth future
}
