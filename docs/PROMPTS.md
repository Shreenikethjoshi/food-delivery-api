#  Prompts Used — Transparency Document

This document lists the prompts used while building the Food Delivery Order Management API. Submitted as required by Infosys guidelines for transparency in tool-assisted development.

---

## 1. System Architecture Design

> Design the architecture for a Food Delivery Order Management REST API in Go. The system should support three user roles: customer, restaurant owner, and delivery driver. Include JWT-based authentication, role-based access control, and a SQLite database. Suggest the best Go packages to use and explain the project folder structure.

**Used for:** Designing the overall architecture, choosing Gin + GORM + SQLite + JWT stack, and planning the folder structure.

---

## 2. Order State Machine Design

> Design a finite state machine for a food delivery order lifecycle. The order should go through states: PLACED, CONFIRMED, PREPARING, READY_FOR_PICKUP, PICKED_UP, DELIVERED, and CANCELLED. Define which actor (customer, restaurant, driver) can trigger each transition. How do I implement this efficiently in Go so that invalid transitions are rejected at runtime?

**Used for:** Designing the 7-state FSM, the actor-aware transition table, and implementing it using an O(1) hash map lookup in Go.

---

## 3. Preventing Invalid State Transitions

> How do I enforce state machine rules strictly in a Go REST API so that no handler can bypass validation? I want the system to reject invalid transitions with a clear HTTP error and also tell the caller what the valid next states are. Implement this in Go with GORM and Gin.

**Used for:** Implementing the `CanTransition()` validator, `ValidTransitionsFrom()` helper, and the multi-layer defense strategy documented in `INVALID_TRANSITIONS.md`.

---

## 4. JWT Authentication and Role Middleware

> Implement JWT-based authentication in Go using `golang-jwt/jwt`. Create a middleware that extracts the user's role from the token and enforces role-based access. Show how to group routes by role in Gin so that customers cannot access restaurant or driver endpoints.

**Used for:** Writing `middleware/auth.go` — token generation, `AuthRequired()`, and `RoleRequired()` middleware.

---

## 5. Database Models with GORM

> Design GORM models in Go for a food delivery system. Include: User (with role), Restaurant, MenuItem, Order (with status history), and OrderItem. Make sure the Order model supports price snapshots per item and an audit trail of status changes. Use SQLite as the database.

**Used for:** Writing all models in the `models/` directory, including the `OrderStatusHistory` audit trail model.

---

## 6. Price Snapshot and ETA Calculation

> When a customer places an order, how do I store a snapshot of the item price at the time of order (not the current price), and auto-calculate an estimated delivery time based on the number of items ordered? Implement this in Go.

**Used for:** The `PlaceOrder` handler — price snapshot logic per `OrderItem`, and the ETA formula (30 + 5 × item types).

---

## 7. Switching to a Pure-Go SQLite Driver

> The `gorm.io/driver/sqlite` package requires CGO and a C compiler to work on Windows. What is a pure-Go alternative that works without installing MinGW or GCC, and how do I switch to it in a GORM-based project?

**Used for:** Switching from `gorm.io/driver/sqlite` to `github.com/glebarez/sqlite` to make the project work on Windows without any C compiler.

---

## 8. Admin Dashboard and Order Analytics

> Add an admin-only endpoint in Go that returns all orders with a summary grouped by status and total revenue from delivered orders. Also add the ability for admin to force-override any order's status in emergency situations, with the change logged in the audit trail.

**Used for:** Writing `handlers/admin.go` — order analytics, revenue totals, user/restaurant management, and emergency override endpoint.

---

*All implementation decisions were reviewed, tested, and verified manually using Postman.*

