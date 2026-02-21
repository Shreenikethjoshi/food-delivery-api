package handlers

import (
	"net/http"

	"food-delivery-api/config"
	"food-delivery-api/middleware"
	"food-delivery-api/models"

	"github.com/gin-gonic/gin"
)

// ── Restaurant Management ────────────────────────────────────────────────────

type CreateRestaurantRequest struct {
	Name        string `json:"name" binding:"required"`
	Cuisine     string `json:"cuisine"`
	Address     string `json:"address" binding:"required"`
	Description string `json:"description"`
}

// CreateRestaurant lets a restaurant-role user create their restaurant
func CreateRestaurant(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	var req CreateRestaurantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restaurant := models.Restaurant{
		OwnerID:     ownerID,
		Name:        req.Name,
		Cuisine:     req.Cuisine,
		Address:     req.Address,
		Description: req.Description,
		IsOpen:      true,
	}
	if err := config.DB.Create(&restaurant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create restaurant"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Restaurant created", "restaurant": restaurant})
}

// GetMyRestaurant fetches the restaurant owned by the logged-in user
func GetMyRestaurant(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	var restaurant models.Restaurant
	if err := config.DB.Preload("MenuItems").Where("owner_id = ?", ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No restaurant found for your account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restaurant": restaurant})
}

// UpdateRestaurant updates restaurant details
func UpdateRestaurant(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	var restaurant models.Restaurant
	if err := config.DB.Where("owner_id = ?", ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Only allow safe fields
	allowed := map[string]bool{"name": true, "cuisine": true, "address": true, "description": true, "is_open": true}
	update := map[string]interface{}{}
	for k, v := range req {
		if allowed[k] {
			update[k] = v
		}
	}
	config.DB.Model(&restaurant).Updates(update)
	c.JSON(http.StatusOK, gin.H{"message": "Restaurant updated", "restaurant": restaurant})
}

// ── Menu Management ─────────────────────────────────────────────────────────

type CreateMenuItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Category    string  `json:"category"`
	IsVeg       bool    `json:"is_veg"`
}

// AddMenuItem adds a new item to the restaurant's menu
func AddMenuItem(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	var restaurant models.Restaurant
	if err := config.DB.Where("owner_id = ?", ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Create a restaurant first before adding menu items"})
		return
	}

	var req CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := models.MenuItem{
		RestaurantID: restaurant.ID,
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.Price,
		Category:     req.Category,
		IsVeg:        req.IsVeg,
		IsAvailable:  true,
	}
	if err := config.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add menu item"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Menu item added", "item": item})
}

// UpdateMenuItem updates a menu item (only by the owner)
func UpdateMenuItem(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var item models.MenuItem
	if err := config.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	// Verify ownership
	var restaurant models.Restaurant
	if err := config.DB.Where("id = ? AND owner_id = ?", item.RestaurantID, ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this menu item"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Model(&item).Updates(req)
	c.JSON(http.StatusOK, gin.H{"message": "Menu item updated", "item": item})
}

// DeleteMenuItem removes a menu item
func DeleteMenuItem(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var item models.MenuItem
	if err := config.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}
	var restaurant models.Restaurant
	if err := config.DB.Where("id = ? AND owner_id = ?", item.RestaurantID, ownerID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this menu item"})
		return
	}
	config.DB.Delete(&item)
	c.JSON(http.StatusOK, gin.H{"message": "Menu item deleted"})
}
