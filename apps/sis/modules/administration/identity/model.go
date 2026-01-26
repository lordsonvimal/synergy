package identity

import "time"

type User struct {
	ID        int64
	Email     string
	Name      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time

	Memberships   []Membership
	Relationships []UserRelationship
}

type Membership struct {
	ID             int64
	OrganizationID int64
	Role           Role
	CreatedAt      time.Time
}

type Role struct {
	ID          int64
	Name        string
	Description string
}

type UserRelationship struct {
	ID               int64
	RelatedUserID    int64
	RelatedUserName  string
	RelationshipType string
	CreatedAt        time.Time
}
