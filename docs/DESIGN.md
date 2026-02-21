# Food Delivery API — Design Document

## Overview

This document explains the architecture and design decisions behind the Food Delivery Order Management API — built for the Infosys Capstone Project using Go (Golang).

The system models a real-world food delivery platform (like Zomato/Uber Eats) with three distinct actor types coordinating through a strict **finite state machine**.

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Client (Postman)               │
└──────────────────────┬──────────────────────────┘
                       │ HTTP Request
┌──────────────────────▼──────────────────────────┐
│              Gin HTTP Router                    │
│         (routes/routes.go)                      │
├─────────────────────────────────────────────────┤
│           JWT Auth Middleware                   │
│    verifies token + enforces role access        │
├─────────────────────────────────────────────────┤
│              Handler Layer                      │
│  auth / customer / restaurant / driver / admin  │
├─────────────────────────────────────────────────┤
│           State Machine Validator               │
│   statemachine/order_state.go (O(1) lookup)     │
├─────────────────────────────────────────────────┤
│           GORM ORM (Data Layer)                 │
├─────────────────────────────────────────────────┤
│        SQLite Database (food_delivery.db)        │
└─────────────────────────────────────────────────┘
```

---

## Key Design Decisions

### 1. SQLite — Zero Config Database
SQLite was chosen over PostgreSQL/MySQL because:
- No server installation required
- Single file (`food_delivery.db`) — easy to deploy and demo
- Perfect for capstone/evaluation environments
- GORM AutoMigrate handles schema automatically

### 2. Gin Framework
- Industry-standard Go web framework
- Built-in middleware support (logger, recovery, custom auth)
- Significantly cleaner routing than raw `net/http`
- Automatic JSON binding and validation

### 3. GORM ORM
- Translates Go structs directly to SQL tables
- AutoMigrate keeps schema in sync with code
- Supports preloading relations (e.g., `Preload("Items.MenuItem")`)
- No raw SQL needed — reduces SQL injection risk

### 4. JWT Authentication (HS256)
- Stateless — no session store needed
- Token contains: `user_id`, `email`, `role`
- 24-hour expiry
- Role is extracted in middleware and passed through Gin context

### 5. Actor-Based State Machine (Core Innovation)
The state machine is not just "what transitions are valid" — it also tracks **who** can make each transition:

```go
type Transition struct {
    From  OrderStatus
    To    OrderStatus
    Actor string  // "restaurant", "driver", "customer"
}
```

This means a driver cannot confirm an order, and a customer cannot mark food as ready. Role violation returns `HTTP 403`, invalid state returns `HTTP 422`.

### 6. Price Snapshot in OrderItems
When an order is placed, each `OrderItem` stores the **current price and name** of the menu item. This means if the restaurant later changes the price, existing orders are unaffected — matching real-world billing behavior.

### 7. Audit Trail (Novelty)
Every status change creates a row in `order_status_histories`:
- Who changed it (`changed_by` = user ID)
- What it changed from/to
- Optional note/reason
- Exact timestamp

This enables full order traceability — important for dispute resolution in real systems.

### 8. Estimated Delivery Time (Novelty)
At order placement, ETA is auto-calculated:
```
ETA = 30 minutes (base) + 5 minutes × number of item types
```

---

## Role System

| Role | Can Do |
|---|---|
| `customer` | Browse, order, track, cancel (PLACED/CONFIRMED only) |
| `restaurant` | Manage restaurant + menu, confirm/prepare/ready |
| `driver` | View available orders, pickup, deliver |
| `admin` | View all data, force-override any order state |

Routes are **grouped by role** in `routes/routes.go`. Each group has its own middleware chain:
```go
customer.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleCustomer))
```

---

## State Machine — Full Transition Table

| From | → To | Actor | How |
|---|---|---|---|
| PLACED | CONFIRMED | restaurant | `PUT /api/restaurant/orders/:id/status` |
| PLACED | CANCELLED | restaurant / customer | Status endpoint / cancel endpoint |
| CONFIRMED | PREPARING | restaurant | Status endpoint |
| CONFIRMED | CANCELLED | restaurant / customer | Status endpoint / cancel endpoint |
| PREPARING | READY_FOR_PICKUP | restaurant | Status endpoint |
| READY_FOR_PICKUP | PICKED_UP | driver | `PUT /api/driver/orders/:id/pickup` |
| PICKED_UP | DELIVERED | driver | `PUT /api/driver/orders/:id/deliver` |

**Terminal states:** `DELIVERED`, `CANCELLED` — no further transitions possible by any actor (admin override excluded).

---

## API Example — Register & Login

### Register
**Request:**
```json
POST /api/auth/register
{
  "name": "Shreeniketh Joshi",
  "email": "shreenikethjoshi0605@gmail.com",
  "password": "123456",
  "role": "customer"
}
```
**Response (201 Created):**
```json
{
  "message": "Account created successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Shreeniketh Joshi",
    "email": "shreenikethjoshi0605@gmail.com",
    "role": "customer"
  }
}
```

### Register Restaurant Owner
**Request:**
```json
POST /api/auth/register
{
  "name": "Travel INN",
  "email": "travelinn@hotel.com",
  "password": "123456",
  "role": "restaurant"
}
```
**Response (201 Created):**
```json
{
  "message": "Account created successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 2,
    "name": "Travel INN",
    "email": "travelinn@hotel.com",
    "role": "restaurant"
  }
}
```

### Place Order
**Request:**
```json
POST /api/customer/orders
Authorization: Bearer <customer_token>
{
  "restaurant_id": 1,
  "delivery_address": "45 Brigade Road, Bangalore",
  "notes": "Please pack separately",
  "items": [
    { "menu_item_id": 1, "quantity": 1 },
    { "menu_item_id": 2, "quantity": 2 },
    { "menu_item_id": 3, "quantity": 3 }
  ]
}
```
**Response (201 Created):**
```json
{
  "estimated_time": 45,
  "message": "Order placed successfully",
  "order": {
    "id": 1,
    "status": "PLACED",
    "total_price": 819.94,
    "estimated_time_minutes": 45,
    "delivery_address": "45 Brigade Road, Bangalore"
  }
}
```
> **total_price** = 269.99×1 + 199.99×2 + 49.99×3 = ₹819.94 (auto-calculated)
> **estimated_time** = 30 min base + 5×3 item types = 45 mins (auto-calculated)

---

### State Machine — Complete Flow (Real Test Results)

| Step | Request | Response |
|---|---|---|
| 1 | Restaurant: `CONFIRMED` | `PLACED -> CONFIRMED` |
| 2 | Restaurant: `PREPARING` | `CONFIRMED -> PREPARING` |
| 3 | Restaurant: `READY_FOR_PICKUP` | `PREPARING -> READY_FOR_PICKUP` |
| 4 | Driver: pickup | `READY_FOR_PICKUP -> PICKED_UP` |
| 5 | Driver: deliver | `PICKED_UP -> DELIVERED` |

### Invalid State Transition (Real Test Result)
**Request:**
```
PUT /api/customer/orders/1/cancel
Authorization: Bearer <customer_token>
(Order status is DELIVERED)
```
**Response (HTTP 422 Unprocessable Entity):**
```json
{
  "current_state": "DELIVERED",
  "error": "Cannot cancel order",
  "reason": "invalid transition: DELIVERED → CANCELLED is not allowed for actor 'customer'. Valid transitions from DELIVERED are: none (terminal state)"
}
```
> The state machine correctly identifies `DELIVERED` as a **terminal state** and blocks the cancellation with a descriptive error.

---

## Project Structure

```
food-delivery-api/
├── main.go                    # Server entry point + CORS
├── config/config.go           # DB connection (pure-Go SQLite)
├── models/
│   ├── user.go                # User{} + UserRole type
│   ├── restaurant.go          # Restaurant{} + MenuItem{}
│   └── order.go               # Order{} + OrderItem{} + OrderStatusHistory{}
├── statemachine/
│   └── order_state.go         # FSM: transitions, validator, O(1) lookup
├── middleware/
│   └── auth.go                # JWT generate, AuthRequired(), RoleRequired()
├── handlers/
│   ├── auth.go                # Register, Login, GetProfile
│   ├── public.go              # ListRestaurants, GetMenu, GetStateMachineInfo
│   ├── restaurant.go          # CreateRestaurant, AddMenuItem, UpdateMenuItem
│   ├── restaurant_order.go    # GetRestaurantOrders, UpdateOrderStatus
│   ├── customer.go            # PlaceOrder, GetMyOrders, CancelOrder
│   ├── driver.go              # GetAvailableOrders, PickupOrder, DeliverOrder
│   └── admin.go               # AdminGetAllOrders, AdminForceOrderStatus
├── routes/routes.go           # All route groups with middleware chains
└── docs/
    ├── SCHEMA.md              # SQL table definitions
    ├── STATE_MACHINE.md       # FSM diagram + transition table
    ├── INVALID_TRANSITIONS.md # Prevention approach
    └── PROMPTS.md             # AI tool transparency
```

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/gin-gonic/gin` | HTTP framework |
| `gorm.io/gorm` | ORM |
| `github.com/glebarez/sqlite` | Pure-Go SQLite driver (no C compiler needed) |
| `github.com/golang-jwt/jwt/v5` | JWT token generation + validation |
| `golang.org/x/crypto/bcrypt` | Password hashing |

---

