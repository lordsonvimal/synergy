package dashboard

import "database/sql"

type Metrics struct {
	TotalStudents int
	TotalCourses  int
	FeesCollected float64
	FeesPending   float64
}

func GetMetrics(db *sql.DB) (Metrics, error) {
	var m Metrics

	db.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM enrollments
	`).Scan(&m.TotalStudents)

	db.QueryRow(`
		SELECT COUNT(*) FROM courses
	`).Scan(&m.TotalCourses)

	db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM invoices
		WHERE status = 'paid'
	`).Scan(&m.FeesCollected)

	db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM invoices
		WHERE status = 'pending'
	`).Scan(&m.FeesPending)

	return m, nil
}
