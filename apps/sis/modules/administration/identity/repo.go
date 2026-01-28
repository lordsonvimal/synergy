package identity

import (
	"database/sql"
	"fmt"
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

func (r *UserRepository) DeleteUser(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}

func (r *UserRepository) GetAllUsersAcrossOrganizations() ([]UserInfo, error) {
	rows, err := r.db.Query(`
		SELECT
			u.id,
			u.name,
			GROUP_CONCAT(DISTINCT r.name) AS roles,
			u.is_active,
			COUNT(DISTINCT e.section_id) AS course_count,
			MIN(p.name) AS parent_name,
			MIN(p.email) AS parent_email
		FROM users u
		LEFT JOIN memberships m ON m.user_id = u.id
		LEFT JOIN roles r ON r.id = m.role_id
		LEFT JOIN enrollments e ON e.user_id = u.id
		LEFT JOIN user_relationships ur ON ur.user_id = u.id
		LEFT JOIN users p ON p.id = ur.related_user_id
		GROUP BY u.id
		ORDER BY u.id ASC;
	`)
	if err != nil {
		fmt.Println("Error fetching users:", err)
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

func (r *UserRepository) CreateUser(form *UserForm) (err error) {
	fmt.Println("Creating user:", form.Name, form.Email, form.Phone, form.IsActive, form.Roles)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	res, err := tx.Exec(`
		INSERT INTO users (name, email, phone, is_active)
		VALUES (?, ?, ?, ?)
	`, form.Name, form.Email, form.Phone, form.IsActive)
	if err != nil {
		return err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	fmt.Println("Created userId:", userID)

	for _, role := range form.Roles {
		_, err = tx.Exec(`
			INSERT INTO memberships (user_id, role_id, organization_id)
			VALUES (?, ?, ?)
		`, userID, role.RoleID, role.OrganizationID)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}
