package models

import "time"

// OrderStatus represents all possible states of a food delivery order
type OrderStatus string

const (
	StatusPlaced          OrderStatus = "PLACED"
	StatusConfirmed       OrderStatus = "CONFIRMED"
	StatusPreparing       OrderStatus = "PREPARING"
	StatusReadyForPickup  OrderStatus = "READY_FOR_PICKUP"
	StatusPickedUp        OrderStatus = "PICKED_UP"
	StatusDelivered       OrderStatus = "DELIVERED"
	StatusCancelled       OrderStatus = "CANCELLED"
)

type Order struct {
	ID              uint         `json:"id" gorm:"primaryKey"`
	CustomerID      uint         `json:"customer_id" gorm:"not null"`
	Customer        User         `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	RestaurantID    uint         `json:"restaurant_id" gorm:"not null"`
	Restaurant      Restaurant   `json:"restaurant,omitempty" gorm:"foreignKey:RestaurantID"`
	DriverID        *uint        `json:"driver_id"`
	Driver          *User        `json:"driver,omitempty" gorm:"foreignKey:DriverID"`
	Status          OrderStatus  `json:"status" gorm:"not null;default:'PLACED'"`
	TotalPrice      float64      `json:"total_price"`
	DeliveryAddress string       `json:"delivery_address" gorm:"not null"`
	Notes           string       `json:"notes"`
	EstimatedTime   int          `json:"estimated_time_minutes"` // novelty: ETA in minutes
	Items           []OrderItem  `json:"items,omitempty" gorm:"foreignKey:OrderID"`
	StatusHistory   []OrderStatusHistory `json:"status_history,omitempty" gorm:"foreignKey:OrderID"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type OrderItem struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	OrderID    uint     `json:"order_id" gorm:"not null"`
	MenuItemID uint     `json:"menu_item_id" gorm:"not null"`
	MenuItem   MenuItem `json:"menu_item,omitempty" gorm:"foreignKey:MenuItemID"`
	Quantity   int      `json:"quantity" gorm:"not null"`
	Price      float64  `json:"price" gorm:"not null"` // snapshot price at time of order
	Name       string   `json:"name"`                  // snapshot name
}

// OrderStatusHistory tracks every status change — audit trail novelty
type OrderStatusHistory struct {
	ID        uint        `json:"id" gorm:"primaryKey"`
	OrderID   uint        `json:"order_id" gorm:"not null"`
	FromStatus OrderStatus `json:"from_status"`
	ToStatus  OrderStatus `json:"to_status" gorm:"not null"`
	ChangedBy uint        `json:"changed_by"` // user ID who triggered the transition
	Note      string      `json:"note"`
	CreatedAt time.Time   `json:"created_at"`
}
