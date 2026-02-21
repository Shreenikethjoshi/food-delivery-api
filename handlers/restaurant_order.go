package handlers

import (
	"net/http"

	"food-delivery-api/config"
	"food-delivery-api/middleware"
	"food-delivery-api/models"
	"food-delivery-api/statemachine"

	"github.com/gin-gonic/gin"
)

// GetRestaurantOrders returns all orders for the restaurant owner
func GetRestaurantOrders(c *gin.Context) {
	ownerID := middleware.GetUserID(c)

	var restaurant models.Restaurant
	if err := config.DB.Where("owner_id = ?", ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No restaurant found for your account"})
		return
	}

	var orders []models.Order
	query := config.DB.Preload("Items.MenuItem").Preload("Customer").Preload("Driver").
		Where("restaurant_id = ?", restaurant.ID)

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("created_at desc").Find(&orders)

	// Group counts by status — novelty: dashboard summary
	summary := map[string]int{}
	for _, o := range orders {
		summary[string(o.Status)]++
	}

	c.JSON(http.StatusOK, gin.H{
		"restaurant":    restaurant.Name,
		"order_summary": summary,
		"count":         len(orders),
		"orders":        orders,
	})
}

type UpdateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status" binding:"required"`
	Note   string             `json:"note"`
}

// UpdateOrderStatus handles restaurant's state transitions
func UpdateOrderStatus(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	orderID := c.Param("id")

	var restaurant models.Restaurant
	if err := config.DB.Where("owner_id = ?", ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No restaurant found for your account"})
		return
	}

	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.RestaurantID != restaurant.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "This order does not belong to your restaurant"})
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := statemachine.CanTransition(order.Status, req.Status, "restaurant"); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":             "Invalid state transition",
			"current_status":    order.Status,
			"requested":         req.Status,
			"reason":            err.Error(),
			"valid_next_states": statemachine.ValidTransitionsFrom(order.Status),
		})
		return
	}

	prevStatus := order.Status
	config.DB.Model(&order).Update("status", req.Status)

	// Auto-set estimated time when preparing
	if req.Status == models.StatusPreparing {
		config.DB.Model(&order).Update("estimated_time", 20)
	}

	history := models.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: prevStatus,
		ToStatus:   req.Status,
		ChangedBy:  ownerID,
		Note:       req.Note,
	}
	config.DB.Create(&history)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Order status updated",
		"order_id":        order.ID,
		"previous_status": string(prevStatus),
		"current_status":  string(req.Status),
	})
}
