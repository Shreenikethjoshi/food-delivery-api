package models

import "time"

type Restaurant struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	OwnerID     uint       `json:"owner_id" gorm:"not null"`
	Owner       User       `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Name        string     `json:"name" gorm:"not null"`
	Cuisine     string     `json:"cuisine"`
	Address     string     `json:"address"`
	Description string     `json:"description"`
	IsOpen      bool       `json:"is_open" gorm:"default:true"`
	Rating      float64    `json:"rating" gorm:"default:0"`
	MenuItems   []MenuItem `json:"menu_items,omitempty" gorm:"foreignKey:RestaurantID"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type MenuItem struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	RestaurantID uint       `json:"restaurant_id" gorm:"not null"`
	Name         string     `json:"name" gorm:"not null"`
	Description  string     `json:"description"`
	Price        float64    `json:"price" gorm:"not null"`
	Category     string     `json:"category"`
	IsAvailable  bool       `json:"is_available" gorm:"default:true"`
	IsVeg        bool       `json:"is_veg" gorm:"default:false"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
