package domain

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleStaff Role = "staff"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}
