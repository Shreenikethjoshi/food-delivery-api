package handlers

import (
	"net/http"

	"food-delivery-api/config"
	"food-delivery-api/models"

	"github.com/gin-gonic/gin"
)

// ListRestaurants returns all open restaurants (public)
func ListRestaurants(c *gin.Context) {
	var restaurants []models.Restaurant
	query := config.DB.Preload("Owner")

	// Novelty: filter by cuisine or search by name
	if cuisine := c.Query("cuisine"); cuisine != "" {
		query = query.Where("cuisine LIKE ?", "%"+cuisine+"%")
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if open := c.Query("open"); open == "true" {
		query = query.Where("is_open = ?", true)
	}

	query.Find(&restaurants)
	c.JSON(http.StatusOK, gin.H{
		"count":       len(restaurants),
		"restaurants": restaurants,
	})
}

// GetRestaurant returns a single restaurant
func GetRestaurant(c *gin.Context) {
	var restaurant models.Restaurant
	if err := config.DB.Preload("MenuItems").First(&restaurant, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restaurant": restaurant})
}

// GetMenu returns the menu for a specific restaurant (public)
func GetMenu(c *gin.Context) {
	restaurantID := c.Param("id")
	var restaurant models.Restaurant
	if err := config.DB.First(&restaurant, restaurantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var items []models.MenuItem
	query := config.DB.Where("restaurant_id = ?", restaurantID)

	// Novelty: filter by category or veg
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if isVeg := c.Query("is_veg"); isVeg == "true" {
		query = query.Where("is_veg = ?", true)
	}
	query.Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"restaurant": restaurant.Name,
		"count":      len(items),
		"menu":       items,
	})
}

// GetStateMachineInfo returns the full state machine for informational purposes
func GetStateMachineInfo(c *gin.Context) {
	info := []gin.H{
		{"from": "PLACED", "to": "CONFIRMED", "actor": "restaurant"},
		{"from": "PLACED", "to": "CANCELLED", "actor": "restaurant or customer"},
		{"from": "CONFIRMED", "to": "PREPARING", "actor": "restaurant"},
		{"from": "CONFIRMED", "to": "CANCELLED", "actor": "restaurant or customer"},
		{"from": "PREPARING", "to": "READY_FOR_PICKUP", "actor": "restaurant"},
		{"from": "READY_FOR_PICKUP", "to": "PICKED_UP", "actor": "driver"},
		{"from": "PICKED_UP", "to": "DELIVERED", "actor": "driver"},
	}
	c.JSON(http.StatusOK, gin.H{
		"state_machine": info,
		"terminal_states": []string{"DELIVERED", "CANCELLED"},
		"description": "Food Delivery Order Lifecycle State Machine",
	})
}
