# Food Delivery Order Management API

A production-quality REST API built in Go (Golang) for managing food delivery orders — similar to Zomato or Uber Eats. Customers place orders, restaurants accept and prepare them, and drivers deliver them.

---

## Problem Statement

Build a REST API for managing food delivery orders. The system must handle:

- Customers placing orders from restaurants
- Restaurants accepting and preparing those orders
- Drivers picking up and delivering completed orders
- A robust **order lifecycle state machine** that enforces valid transitions
- Role-based access so each user type can only perform their allowed actions
- Coordination between three independent user roles without conflicts

The core challenge is implementing a strict state machine where invalid transitions (e.g., delivering an order before it is confirmed, or cancelling an already-delivered order) are structurally impossible — not just discouraged.

---

## Approach

### 1. Actor-Aware Finite State Machine

The order lifecycle has 7 states and 9 transitions. Every transition is defined with three fields: `from state`, `to state`, and `actor` (who can trigger it). This means the system rejects not just invalid states but also valid transitions attempted by the wrong role.

```
PLACED -> CONFIRMED        (restaurant only)
CONFIRMED -> PREPARING     (restaurant only)
PREPARING -> READY_FOR_PICKUP  (restaurant only)
READY_FOR_PICKUP -> PICKED_UP  (driver only)
PICKED_UP -> DELIVERED     (driver only)
PLACED/CONFIRMED -> CANCELLED  (restaurant or customer)
```

### 2. O(1) Hash Map Validation

All valid transitions are stored at startup in a Go hash map keyed by `(from, to, actor)`. Validation is a single map lookup — no branching, no iteration.

### 3. JWT Role-Based Middleware

Each endpoint group is protected by a middleware chain that verifies the JWT token and enforces the role. A customer cannot reach the restaurant or driver endpoints even if they guess the URL.

### 4. Full Audit Trail

Every state change is logged in `order_status_histories` with the actor ID, timestamp, and optional note. This enables full traceability of every order lifecycle.

### 5. Price Snapshot

Item prices are frozen at the time of order placement. Restaurant price changes do not affect existing orders — matching real-world billing behavior.

---


## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| Web Framework | Gin |
| ORM | GORM |
| Database | SQLite (file-based, zero config) |
| Authentication | JWT (HS256) |
| Password Hashing | bcrypt |

---

## Getting Started

### Prerequisites
- Go 1.20 or higher
- No database installation required — SQLite file is created automatically

### 1. Clone and Run

```bash
git clone https://github.com/Shreenikethjoshi/food-delivery-api.git
cd food-delivery-api
go mod tidy
go run main.go
```

You will see this output when the server starts:

```
[GIN-debug] Listening and serving HTTP on :8080
Database connected and migrated successfully
Food Delivery API running on http://localhost:8080
```

### 2. Verify the Server is Running

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "service": "Food Delivery Order Management API",
  "status": "healthy",
  "version": "1.0.0"
}
```

### 3. Build as a Single Binary (optional)

```bash
go build -o food-delivery-api.exe .
./food-delivery-api.exe
```

### Environment Variables (optional)

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `JWT_SECRET` | `food_delivery_super_secret_2024` | JWT signing key |
| `GIN_MODE` | `debug` | Set to `release` in production |

---

## Project Structure

```
food-delivery-api/
├── main.go                    # Entry point
├── config/config.go           # DB init, JWT secret
├── models/
│   ├── user.go                # User + Role types
│   ├── restaurant.go          # Restaurant + MenuItem
│   └── order.go               # Order + OrderItem + StatusHistory
├── statemachine/
│   └── order_state.go         # State machine with O(1) transition lookup
├── middleware/
│   └── auth.go                # JWT generation + auth + role middleware
├── handlers/
│   ├── auth.go                # Register, Login, Profile
│   ├── public.go              # Public restaurant/menu browsing
│   ├── restaurant.go          # Restaurant + menu management
│   ├── restaurant_order.go    # Restaurant order state transitions
│   ├── customer.go            # Customer order flow
│   ├── driver.go              # Driver pickup + delivery flow
│   └── admin.go               # Admin dashboard
├── routes/routes.go           # All route registrations
└── docs/
    ├── DESIGN.md
    ├── SCHEMA.md
    ├── STATE_MACHINE.md
    ├── INVALID_TRANSITIONS.md
    └── PROMPTS.md
```

---

## User Roles

| Role | Description |
|---|---|
| `customer` | Browses restaurants, places orders, tracks and cancels orders |
| `restaurant` | Manages their restaurant, menu, and processes orders |
| `driver` | Picks up ready orders and delivers them |
| `admin` | Full visibility + emergency order override |

---

## Example API Calls

### 1. Register a Customer

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Shreeniketh Joshi",
    "email": "shreenikethjoshi0605@gmail.com",
    "password": "123456",
    "role": "customer"
  }'
```

**Response:**
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

---

### 2. Register a Restaurant Owner

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Travel INN",
    "email": "travelinn@hotel.com",
    "password": "123456",
    "role": "restaurant"
  }'
```

**Response:**
```json
{
  "message": "Account created successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": { "id": 2, "name": "Travel INN", "role": "restaurant" }
}
```

---

### 3. Create a Restaurant and Add Menu Items

```bash
curl -X POST http://localhost:8080/api/restaurant/ \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Travel INN Restaurant",
    "cuisine": "North Indian",
    "address": "12 MG Road, Bangalore",
    "description": "Authentic North Indian cuisine since 1995"
  }'
```

```bash
curl -X POST http://localhost:8080/api/restaurant/menu \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paneer Butter Masala",
    "description": "Rich creamy paneer curry",
    "price": 269.99,
    "category": "Main Course",
    "is_veg": true
  }'
```

**Response:**
```json
{
  "message": "Menu item added",
  "item": { "id": 1, "name": "Paneer Butter Masala", "price": 269.99, "is_veg": true }
}
```

---

### 4. Browse Restaurants and Menu

```bash
curl http://localhost:8080/api/restaurants
```

```bash
curl http://localhost:8080/api/restaurants/1/menu
```

```bash
# Filter veg items only
curl http://localhost:8080/api/restaurants/1/menu?is_veg=true
```

---

### 5. Customer Places an Order

```bash
curl -X POST http://localhost:8080/api/customer/orders \
  -H "Authorization: Bearer <customer_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "restaurant_id": 1,
    "delivery_address": "45 Brigade Road, Bangalore",
    "notes": "Please pack separately",
    "items": [
      { "menu_item_id": 1, "quantity": 1 },
      { "menu_item_id": 2, "quantity": 2 },
      { "menu_item_id": 3, "quantity": 3 }
    ]
  }'
```

**Response:**
```json
{
  "message": "Order placed successfully",
  "estimated_time": 45,
  "order": {
    "id": 1,
    "status": "PLACED",
    "total_price": 819.94,
    "estimated_time_minutes": 45,
    "delivery_address": "45 Brigade Road, Bangalore"
  }
}
```

> **total_price** is auto-calculated from all items. **estimated_time** = 30 min base + 5 min per item type.

---

### 6. Restaurant Processes the Order (State Machine)

```bash
# PLACED -> CONFIRMED
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "CONFIRMED", "note": "Order received!" }'

# CONFIRMED -> PREPARING
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "PREPARING" }'

# PREPARING -> READY_FOR_PICKUP
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "READY_FOR_PICKUP", "note": "Food is packed!" }'
```

**Response (each step):**
```json
{
  "message": "Order status updated",
  "order_id": 1,
  "previous_status": "PLACED",
  "current_status": "CONFIRMED"
}
```

---

### 7. Driver Picks Up and Delivers

```bash
# See available orders
curl http://localhost:8080/api/driver/orders/available \
  -H "Authorization: Bearer <driver_token>"

# Pick up
curl -X PUT http://localhost:8080/api/driver/orders/1/pickup \
  -H "Authorization: Bearer <driver_token>"

# Deliver
curl -X PUT http://localhost:8080/api/driver/orders/1/deliver \
  -H "Authorization: Bearer <driver_token>"
```

**Response (deliver):**
```json
{
  "message": "Order delivered successfully!",
  "order_id": 1,
  "status": "DELIVERED"
}
```

---

### 8. Invalid State Transition (State Machine Guard)

```bash
# Try to cancel an already-delivered order
curl -X PUT http://localhost:8080/api/customer/orders/1/cancel \
  -H "Authorization: Bearer <customer_token>"
```

**Response (HTTP 422 Unprocessable Entity):**
```json
{
  "current_state": "DELIVERED",
  "error": "Cannot cancel order",
  "reason": "invalid transition: DELIVERED -> CANCELLED is not allowed for actor 'customer'. Valid transitions from DELIVERED are: none (terminal state)"
}
```

---

## State Machine

```
PLACED --------> CONFIRMED --------> PREPARING --------> READY_FOR_PICKUP --------> PICKED_UP --------> DELIVERED
  |                  |
  |                  |
  +------------------+-------------------------------------------------------------> CANCELLED
  (customer/restaurant)
```

| From | To | Actor |
|---|---|---|
| PLACED | CONFIRMED | restaurant |
| PLACED | CANCELLED | restaurant / customer |
| CONFIRMED | PREPARING | restaurant |
| CONFIRMED | CANCELLED | restaurant / customer |
| PREPARING | READY_FOR_PICKUP | restaurant |
| READY_FOR_PICKUP | PICKED_UP | driver |
| PICKED_UP | DELIVERED | driver |

Terminal states: `DELIVERED`, `CANCELLED` — no further transitions allowed by any actor.

---

## Novelty Features

1. **Order Status Audit Trail** — Every status change is logged in `order_status_histories` with actor ID, timestamp, and optional note
2. **Estimated Delivery Time** — Auto-calculated at order placement: 30 min base + 5 min per unique item type
3. **Admin Revenue Dashboard** — Returns total revenue from DELIVERED orders grouped by status
4. **Concurrent Pickup Protection** — Driver pickup checks for existing `driver_id` to prevent race conditions
5. **Rich Error Messages** — Invalid transitions return current state AND all valid next states
6. **Menu Filtering** — Filter by `?is_veg=true`, `?category=Starters`, `?cuisine=North Indian`
7. **Price Snapshot** — Item prices are frozen at order time; restaurant price changes don't affect existing orders
8. **Emergency Admin Override** — Admin can force any order to any state (with full audit logging)

---

## API Endpoints

### Public (No Auth)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `POST` | `/api/auth/register` | Register new user |
| `POST` | `/api/auth/login` | Login and get JWT |
| `GET` | `/api/restaurants` | List all restaurants |
| `GET` | `/api/restaurants/:id/menu` | Restaurant menu |

### Customer
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/customer/orders` | Place a new order |
| `GET` | `/api/customer/orders` | My order history |
| `PUT` | `/api/customer/orders/:id/cancel` | Cancel order |

### Restaurant
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/restaurant/` | Create restaurant |
| `POST` | `/api/restaurant/menu` | Add menu item |
| `GET` | `/api/restaurant/orders` | View incoming orders |
| `PUT` | `/api/restaurant/orders/:id/status` | Update order status |

### Driver
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/driver/orders/available` | Available orders |
| `PUT` | `/api/driver/orders/:id/pickup` | Pick up an order |
| `PUT` | `/api/driver/orders/:id/deliver` | Mark as delivered |

### Admin
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/admin/orders` | All orders + revenue |
| `PUT` | `/api/admin/orders/:id/status` | Force-override status |
| `GET` | `/api/admin/users` | All users |

---

## Database

SQLite file: `food_delivery.db` — auto-created on first run, no installation needed. View it with [DB Browser for SQLite](https://sqlitebrowser.org/).

See [docs/SCHEMA.md](docs/SCHEMA.md) for the full table definitions.

---

## Documentation

| File | Contents |
|---|---|
| [docs/DESIGN.md](docs/DESIGN.md) | Architecture, design decisions, real API examples |
| [docs/SCHEMA.md](docs/SCHEMA.md) | Full SQL schema for all 6 tables |
| [docs/STATE_MACHINE.md](docs/STATE_MACHINE.md) | State diagram and transition table |
| [docs/INVALID_TRANSITIONS.md](docs/INVALID_TRANSITIONS.md) | How invalid transitions are prevented |
| [docs/PROMPTS.md](docs/PROMPTS.md) | Prompts used (Infosys transparency requirement) |

---

