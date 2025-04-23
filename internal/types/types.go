package types

import "time"

// User represents a user based on task_types.py User dataclass
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// LoginRequest represents the data needed for a login attempt based on types.py
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ResUser represents the user data returned after login based on types.py
type ResUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

// AuthService interface defines the authentication operations
type AuthService interface {
	Login(email, password string) (*ResUser, error)
}

// Project represents a project based on task_types.py Project dataclass
type Project struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedBy   *User      `json:"created_by,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	Members     []User     `json:"members,omitempty"`
	FavoriteBy  []User     `json:"favorite_by,omitempty"`
	Status      *string    `json:"status,omitempty"`
}

// Task represents a task based on task_types.py Task dataclass
type Task struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Project     Project `json:"project"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// WorkReport represents a work report based on task_types.py WorkReport dataclass
type WorkReport struct {
	ID          int        `json:"id"`
	User        User       `json:"user"`
	Project     Project    `json:"project"`
	Task        Task       `json:"task"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}
