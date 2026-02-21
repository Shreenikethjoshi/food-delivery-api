package routes

import (
	"food-delivery-api/handlers"
	"food-delivery-api/middleware"
	"food-delivery-api/models"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// ── Public routes ──────────────────────────────────────────────
	public := r.Group("/api")
	{
		// Auth
		public.POST("/auth/register", handlers.Register)
		public.POST("/auth/login", handlers.Login)

		// Restaurants & menus (no auth needed)
		public.GET("/restaurants", handlers.ListRestaurants)
		public.GET("/restaurants/:id", handlers.GetRestaurant)
		public.GET("/restaurants/:id/menu", handlers.GetMenu)

		// State machine info (great for docs/Postman)
		public.GET("/state-machine", handlers.GetStateMachineInfo)
	}

	// ── Authenticated routes ───────────────────────────────────────
	auth := r.Group("/api")
	auth.Use(middleware.AuthRequired())
	{
		auth.GET("/profile", handlers.GetProfile)
	}

	// ── Customer routes ────────────────────────────────────────────
	customer := r.Group("/api/customer")
	customer.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleCustomer))
	{
		customer.POST("/orders", handlers.PlaceOrder)
		customer.GET("/orders", handlers.GetMyOrders)
		customer.GET("/orders/:id", handlers.GetOrderDetail)
		customer.PUT("/orders/:id/cancel", handlers.CancelOrder)
	}

	// ── Restaurant owner routes ────────────────────────────────────
	restaurant := r.Group("/api/restaurant")
	restaurant.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleRestaurant))
	{
		// Restaurant management
		restaurant.POST("/", handlers.CreateRestaurant)
		restaurant.GET("/", handlers.GetMyRestaurant)
		restaurant.PUT("/", handlers.UpdateRestaurant)

		// Menu management
		restaurant.POST("/menu", handlers.AddMenuItem)
		restaurant.PUT("/menu/:itemId", handlers.UpdateMenuItem)
		restaurant.DELETE("/menu/:itemId", handlers.DeleteMenuItem)

		// Order management
		restaurant.GET("/orders", handlers.GetRestaurantOrders)
		restaurant.PUT("/orders/:id/status", handlers.UpdateOrderStatus)
	}

	// ── Driver routes ──────────────────────────────────────────────
	driver := r.Group("/api/driver")
	driver.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleDriver))
	{
		driver.GET("/orders/available", handlers.GetAvailableOrders)
		driver.GET("/orders/my-deliveries", handlers.GetMyDeliveries)
		driver.PUT("/orders/:id/pickup", handlers.PickupOrder)
		driver.PUT("/orders/:id/deliver", handlers.DeliverOrder)
	}

	// ── Admin routes ───────────────────────────────────────────────
	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleAdmin))
	{
		admin.GET("/orders", handlers.AdminGetAllOrders)
		admin.PUT("/orders/:id/status", handlers.AdminForceOrderStatus)
		admin.GET("/users", handlers.AdminGetAllUsers)
		admin.GET("/restaurants", handlers.AdminGetAllRestaurants)
	}
}
