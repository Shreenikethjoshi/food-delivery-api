package handlers

import (
	"net/http"
	"time"

	"food-delivery-api/config"
	"food-delivery-api/middleware"
	"food-delivery-api/models"
	"food-delivery-api/statemachine"

	"github.com/gin-gonic/gin"
)

type PlaceOrderRequest struct {
	RestaurantID    uint   `json:"restaurant_id" binding:"required"`
	DeliveryAddress string `json:"delivery_address" binding:"required"`
	Notes           string `json:"notes"`
	Items           []struct {
		MenuItemID uint `json:"menu_item_id" binding:"required"`
		Quantity   int  `json:"quantity" binding:"required,min=1"`
	} `json:"items" binding:"required,min=1"`
}

// PlaceOrder creates a new order (customer only)
func PlaceOrder(c *gin.Context) {
	customerID := middleware.GetUserID(c)

	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate restaurant exists and is open
	var restaurant models.Restaurant
	if err := config.DB.First(&restaurant, req.RestaurantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}
	if !restaurant.IsOpen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Restaurant is currently closed"})
		return
	}

	// Build order items and calculate total
	var orderItems []models.OrderItem
	var total float64

	for _, reqItem := range req.Items {
		var menuItem models.MenuItem
		if err := config.DB.First(&menuItem, reqItem.MenuItemID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Menu item not found: " + string(rune(reqItem.MenuItemID))})
			return
		}
		if menuItem.RestaurantID != req.RestaurantID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Menu item does not belong to this restaurant"})
			return
		}
		if !menuItem.IsAvailable {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Menu item '" + menuItem.Name + "' is not available"})
			return
		}
		lineTotal := menuItem.Price * float64(reqItem.Quantity)
		total += lineTotal
		orderItems = append(orderItems, models.OrderItem{
			MenuItemID: menuItem.ID,
			Quantity:   reqItem.Quantity,
			Price:      menuItem.Price,
			Name:       menuItem.Name,
		})
	}

	// Novelty: calculate estimated delivery time (base 30 min + 5 per item)
	estimatedTime := 30 + (5 * len(req.Items))

	order := models.Order{
		CustomerID:      customerID,
		RestaurantID:    req.RestaurantID,
		Status:          models.StatusPlaced,
		TotalPrice:      total,
		DeliveryAddress: req.DeliveryAddress,
		Notes:           req.Notes,
		EstimatedTime:   estimatedTime,
		Items:           orderItems,
	}

	if err := config.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to place order"})
		return
	}

	// Record initial status history
	history := models.OrderStatusHistory{
		OrderID:   order.ID,
		ToStatus:  models.StatusPlaced,
		ChangedBy: customerID,
		Note:      "Order placed by customer",
	}
	config.DB.Create(&history)

	config.DB.Preload("Items.MenuItem").Preload("Restaurant").First(&order, order.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Order placed successfully",
		"order":          order,
		"estimated_time": estimatedTime,
	})
}

// GetMyOrders returns all orders for the logged-in customer
func GetMyOrders(c *gin.Context) {
	customerID := middleware.GetUserID(c)
	var orders []models.Order
	config.DB.Preload("Items.MenuItem").Preload("Restaurant").
		Where("customer_id = ?", customerID).
		Order("created_at desc").
		Find(&orders)
	c.JSON(http.StatusOK, gin.H{"count": len(orders), "orders": orders})
}

// GetOrderDetail returns a single order's full detail with history
func GetOrderDetail(c *gin.Context) {
	customerID := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := config.DB.
		Preload("Items.MenuItem").
		Preload("Restaurant").
		Preload("StatusHistory").
		Preload("Driver").
		First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "This order does not belong to you"})
		return
	}

	// Novelty: compute time elapsed
	elapsed := time.Since(order.CreatedAt).Minutes()
	c.JSON(http.StatusOK, gin.H{
		"order":           order,
		"minutes_elapsed": int(elapsed),
	})
}

// CancelOrder cancels an order (customer can cancel PLACED or CONFIRMED)
func CancelOrder(c *gin.Context) {
	customerID := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "This order does not belong to you"})
		return
	}

	if err := statemachine.CanTransition(order.Status, models.StatusCancelled, "customer"); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":         "Cannot cancel order",
			"reason":        err.Error(),
			"current_state": order.Status,
		})
		return
	}

	prevStatus := order.Status
	config.DB.Model(&order).Update("status", models.StatusCancelled)

	history := models.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: prevStatus,
		ToStatus:   models.StatusCancelled,
		ChangedBy:  customerID,
		Note:       "Order cancelled by customer",
	}
	config.DB.Create(&history)

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully", "order_id": order.ID})
}
