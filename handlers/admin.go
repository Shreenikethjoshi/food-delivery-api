package handlers

import (
	"net/http"

	"food-delivery-api/config"
	"food-delivery-api/models"

	"github.com/gin-gonic/gin"
)

// AdminGetAllOrders returns all orders with full detail — admin only
func AdminGetAllOrders(c *gin.Context) {
	var orders []models.Order
	query := config.DB.Preload("Items.MenuItem").
		Preload("Customer").Preload("Restaurant").Preload("Driver").Preload("StatusHistory")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if customerID := c.Query("customer_id"); customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if restaurantID := c.Query("restaurant_id"); restaurantID != "" {
		query = query.Where("restaurant_id = ?", restaurantID)
	}

	query.Order("created_at desc").Find(&orders)

	// Admin dashboard: aggregate by status
	summary := map[string]int{}
	var totalRevenue float64
	for _, o := range orders {
		summary[string(o.Status)]++
		if o.Status == models.StatusDelivered {
			totalRevenue += o.TotalPrice
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"order_summary": summary,
		"total_revenue": totalRevenue,
		"count":         len(orders),
		"orders":        orders,
	})
}

// AdminGetAllUsers returns all users — admin only
func AdminGetAllUsers(c *gin.Context) {
	var users []models.User
	query := config.DB
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	query.Find(&users)
	c.JSON(http.StatusOK, gin.H{"count": len(users), "users": users})
}

// AdminGetAllRestaurants returns all restaurants — admin only
func AdminGetAllRestaurants(c *gin.Context) {
	var restaurants []models.Restaurant
	config.DB.Preload("Owner").Preload("MenuItems").Find(&restaurants)
	c.JSON(http.StatusOK, gin.H{"count": len(restaurants), "restaurants": restaurants})
}

// AdminForceOrderStatus lets admin override any order state (emergency use)
func AdminForceOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	var req struct {
		Status models.OrderStatus `json:"status" binding:"required"`
		Reason string             `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	prevStatus := order.Status
	config.DB.Model(&order).Update("status", req.Status)

	history := models.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: prevStatus,
		ToStatus:   req.Status,
		Note:       "[ADMIN OVERRIDE] " + req.Reason,
	}
	config.DB.Create(&history)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Order status force-updated by admin",
		"order_id":        order.ID,
		"previous_status": prevStatus,
		"new_status":      req.Status,
	})
}
