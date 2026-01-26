package identity

import (
	"database/sql"
	"strings"
)

// OrganizationOption represents an organization dropdown item
type OrganizationOption struct {
	ID   int64
	Name string
}

// RoleOption represents a role dropdown item
type RoleOption struct {
	ID   int64
	Name string
}

type UserInfo struct {
	UserID      int64
	Name        string
	Roles       []string
	CourseCount int
	IsActive    bool
	ParentName  *string
	ParentEmail *string
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetAllUsersAcrossOrganizations() ([]UserInfo, error) {
	rows, err := r.db.Query(`
		SELECT
			u.id,
			u.name,
			GROUP_CONCAT(DISTINCT r.name) AS roles,
			u.is_active,
			COUNT(DISTINCT e.course_id) AS course_count,
			MIN(p.name) AS parent_name,
			MIN(p.email) AS parent_email
		FROM users u
		JOIN memberships m ON m.user_id = u.id
		JOIN roles r ON r.id = m.role_id
		LEFT JOIN enrollments e ON e.user_id = u.id
		LEFT JOIN user_relationships ur
			ON ur.user_id = u.id
			AND ur.relationship_type IN ('parent', 'guardian')
		LEFT JOIN users p ON p.id = ur.related_user_id
		GROUP BY u.id
		ORDER BY u.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserInfo

	for rows.Next() {
		var row UserInfo
		var roles sql.NullString

		err := rows.Scan(
			&row.UserID,
			&row.Name,
			&roles,
			&row.IsActive,
			&row.CourseCount,
			&row.ParentName,
			&row.ParentEmail,
		)
		if err != nil {
			return nil, err
		}

		if roles.Valid {
			row.Roles = strings.Split(roles.String, ",")
		} else {
			row.Roles = []string{}
		}

		result = append(result, row)
	}

	return result, rows.Err()
}

func (r *UserRepository) GetAllOrganizations() ([]OrganizationOption, error) {
	rows, err := r.db.Query(`
		SELECT id, name
		FROM organizations
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []OrganizationOption
	for rows.Next() {
		var o OrganizationOption
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}

	return orgs, rows.Err()
}

// GetAllRoles fetches all roles
func (r *UserRepository) GetAllRoles() ([]RoleOption, error) {
	rows, err := r.db.Query(`
		SELECT id, name
		FROM roles
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []RoleOption
	for rows.Next() {
		var ro RoleOption
		if err := rows.Scan(&ro.ID, &ro.Name); err != nil {
			return nil, err
		}
		roles = append(roles, ro)
	}

	return roles, rows.Err()
}
