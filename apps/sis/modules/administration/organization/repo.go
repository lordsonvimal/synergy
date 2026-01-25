package organization

import "database/sql"

type OrgStat struct {
	ID           int
	Name         string
	Type         string
	StudentCount int
}

func GetOrganizationStats(db *sql.DB) ([]OrgStat, error) {
	rows, err := db.Query(`
		SELECT
			o.id,
			o.name,
			o.type,
			COUNT(DISTINCT e.user_id) as student_count
		FROM organizations o
		LEFT JOIN courses c ON c.organization_id = o.id
		LEFT JOIN sections s ON s.course_id = c.id
		LEFT JOIN enrollments e ON e.section_id = s.id
		GROUP BY o.id
		ORDER BY o.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []OrgStat
	for rows.Next() {
		var o OrgStat
		rows.Scan(&o.ID, &o.Name, &o.Type, &o.StudentCount)
		stats = append(stats, o)
	}
	return stats, nil
}
