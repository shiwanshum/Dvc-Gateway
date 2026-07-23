package models

import "time"

type Role struct {
	ID          string       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string       `json:"name" gorm:"uniqueIndex;size:50"`
	Label       string       `json:"label" gorm:"size:100"`
	Description string       `json:"description" gorm:"size:255"`
	Level       int          `json:"level"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Users       []User       `gorm:"foreignKey:RoleID" json:"-"`
}

type User struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email     string    `json:"email" gorm:"uniqueIndex;size:255"`
	Password  string    `json:"-" gorm:"size:255"`
	Name      string    `json:"name" gorm:"size:100"`
	RoleID    string    `json:"role_id" gorm:"type:uuid"`
	Role      *Role     `json:"role" gorm:"foreignKey:RoleID"`
	Active    bool      `json:"active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
