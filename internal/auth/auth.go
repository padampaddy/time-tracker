package auth

// Service defines the authentication operations
type Service interface {
	Login(email, password string) (*User, error)
}

// User represents authenticated user data
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

// Credentials contains login request data
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
