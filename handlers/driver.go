package handlers

import (
	"net/http"

	"food-delivery-api/config"
	"food-delivery-api/middleware"
	"food-delivery-api/models"
	"food-delivery-api/statemachine"

	"github.com/gin-gonic/gin"
)

// GetAvailableOrders shows orders READY_FOR_PICKUP that have no driver assigned
func GetAvailableOrders(c *gin.Context) {
	var orders []models.Order
	config.DB.Preload("Restaurant").Preload("Customer").
		Where("status = ? AND driver_id IS NULL", models.StatusReadyForPickup).
		Order("created_at asc").
		Find(&orders)
	c.JSON(http.StatusOK, gin.H{
		"count":  len(orders),
		"orders": orders,
	})
}

// GetMyDeliveries returns all orders assigned to the logged-in driver
func GetMyDeliveries(c *gin.Context) {
	driverID := middleware.GetUserID(c)
	var orders []models.Order
	config.DB.Preload("Items.MenuItem").Preload("Restaurant").Preload("Customer").
		Where("driver_id = ?", driverID).
		Order("updated_at desc").
		Find(&orders)
	c.JSON(http.StatusOK, gin.H{"count": len(orders), "orders": orders})
}

// PickupOrder assigns order to the driver and transitions READY_FOR_PICKUP → PICKED_UP
func PickupOrder(c *gin.Context) {
	driverID := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Prevent two drivers picking up same order
	if order.DriverID != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Order has already been picked up by another driver"})
		return
	}

	if err := statemachine.CanTransition(order.Status, models.StatusPickedUp, "driver"); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":             "Invalid state transition",
			"current_status":    order.Status,
			"reason":            err.Error(),
			"valid_next_states": statemachine.ValidTransitionsFrom(order.Status),
		})
		return
	}

	prevStatus := order.Status
	config.DB.Model(&order).Updates(map[string]interface{}{
		"status":    models.StatusPickedUp,
		"driver_id": driverID,
	})

	history := models.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: prevStatus,
		ToStatus:   models.StatusPickedUp,
		ChangedBy:  driverID,
		Note:       "Driver picked up the order",
	}
	config.DB.Create(&history)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Order picked up successfully",
		"order_id": order.ID,
		"status":   models.StatusPickedUp,
	})
}

// DeliverOrder transitions PICKED_UP → DELIVERED
func DeliverOrder(c *gin.Context) {
	driverID := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.DriverID == nil || *order.DriverID != driverID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the assigned driver for this order"})
		return
	}

	if err := statemachine.CanTransition(order.Status, models.StatusDelivered, "driver"); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":          "Invalid state transition",
			"current_status": order.Status,
			"reason":         err.Error(),
		})
		return
	}

	prevStatus := order.Status
	config.DB.Model(&order).Update("status", models.StatusDelivered)

	history := models.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: prevStatus,
		ToStatus:   models.StatusDelivered,
		ChangedBy:  driverID,
		Note:       "Order delivered to customer",
	}
	config.DB.Create(&history)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Order delivered successfully! 🎉",
		"order_id": order.ID,
		"status":   models.StatusDelivered,
	})
}
