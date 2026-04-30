package domain

import (
	"context"
	"time"
)

type Role string

const (
	Admin   Role = "admin"
	Analyst Role = "analyst"
)

type User struct {
    ID          string 
    GitHubID    string    `json:"github_id"`
    Username    string    `json:"username"`
    Email       string    `json:"email"`
    AvatarURL   string    `json:"avatar_url"`
    Role        string    `json:"role"`
    IsActive    bool      `json:"is_active"`
    LastLoginAt time.Time `json:"last_login_at"`
    CreatedAt   time.Time `json:"created_at"`
}

type UserRepository interface {
	FindByID(id string) (*User, error)
	FindByGitHubID(githubID string) (*User, error)
     FindByUsername(ctx context.Context, username string) (*User, error)
	Upsert(ctx context.Context, user *User) error
}
