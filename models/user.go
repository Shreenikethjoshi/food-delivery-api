package models

import (
	"time"
)

// UserRole defines allowed roles in the system
type UserRole string

const (
	RoleCustomer    UserRole = "customer"
	RoleRestaurant  UserRole = "restaurant"
	RoleDriver      UserRole = "driver"
	RoleAdmin       UserRole = "admin"
)

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Role         UserRole  `json:"role" gorm:"not null;default:'customer'"`
	Phone        string    `json:"phone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
