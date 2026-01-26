package course

import (
	"database/sql"
	"strings"
)

type CourseInfo struct {
	CourseID       int64
	Name           string
	OrganizationID int64
	Sections       []string
	StudentCount   int
}

type CourseRepository struct {
	db *sql.DB
}

func NewCourseRepository(db *sql.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) GetAllCoursesAcrossOrganizations() ([]CourseInfo, error) {
	rows, err := r.db.Query(`
		SELECT
			c.id,
			c.name,
			c.organization_id,
			GROUP_CONCAT(DISTINCT s.name) AS sections,
			COUNT(DISTINCT e.user_id) AS student_count
		FROM courses c
		LEFT JOIN sections s
			ON s.course_id = c.id
		LEFT JOIN enrollments e
			ON e.section_id = s.id
		GROUP BY c.id
		ORDER BY c.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CourseInfo

	for rows.Next() {
		var row CourseInfo
		var sections sql.NullString

		err := rows.Scan(
			&row.CourseID,
			&row.Name,
			&row.OrganizationID,
			&sections,
			&row.StudentCount,
		)
		if err != nil {
			return nil, err
		}

		if sections.Valid {
			row.Sections = strings.Split(sections.String, ",")
		} else {
			row.Sections = []string{}
		}

		result = append(result, row)
	}

	return result, rows.Err()
}
