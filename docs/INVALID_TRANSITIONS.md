# Preventing Invalid State Transitions

This document explains how the Food Delivery API makes invalid order state transitions structurally impossible — not just discouraged.

---

## The Problem

In a food delivery system, invalid state transitions cause real-world issues:

- A driver marks an order as delivered before the restaurant confirmed it
- A customer cancels an order that is already out for delivery
- A restaurant skips "preparing" and marks food ready immediately

These are not just data bugs — they break trust between all three parties. The system must make them impossible at every level.

---

## Our Approach: 5-Layer Defense

### Layer 1 — Compile-Time Transition Table

All valid transitions are defined once in `statemachine/order_state.go` as a Go slice:

```go
var validTransitions = []Transition{
    {From: StatusPlaced,         To: StatusConfirmed,      Actor: "restaurant"},
    {From: StatusPlaced,         To: StatusCancelled,      Actor: "restaurant"},
    {From: StatusPlaced,         To: StatusCancelled,      Actor: "customer"},
    {From: StatusConfirmed,      To: StatusPreparing,      Actor: "restaurant"},
    {From: StatusConfirmed,      To: StatusCancelled,      Actor: "restaurant"},
    {From: StatusConfirmed,      To: StatusCancelled,      Actor: "customer"},
    {From: StatusPreparing,      To: StatusReadyForPickup, Actor: "restaurant"},
    {From: StatusReadyForPickup, To: StatusPickedUp,       Actor: "driver"},
    {From: StatusPickedUp,       To: StatusDelivered,      Actor: "driver"},
}
```

This is the single source of truth. Nothing outside this list is possible.

---

### Layer 2 — O(1) Hash Map Validation

At startup, the slice is compiled into a hash map for O(1) lookup:

```go
type transitionKey struct { From, To OrderStatus; Actor string }
var transitionMap = map[transitionKey]bool{ ... }

func CanTransition(from, to OrderStatus, actor string) error {
    key := transitionKey{from, to, actor}
    if !transitionMap[key] {
        return fmt.Errorf("invalid transition: %s -> %s is not allowed for actor '%s'", from, to, actor)
    }
    return nil
}
```

No iteration. No branching. One map lookup.

---

### Layer 3 — JWT Role Middleware

The middleware extracts the user's role from the JWT and verifies it before any handler runs:

```go
restaurant.Use(middleware.AuthRequired(), middleware.RoleRequired(models.RoleRestaurant))
```

A customer cannot reach restaurant endpoints. A driver cannot reach customer endpoints. The `actor` string passed to `CanTransition()` comes directly from the verified JWT — it cannot be forged.

---

### Layer 4 — Ownership Verification

Each handler checks that the caller owns the resource:

```go
// Restaurant handler
if order.RestaurantID != myRestaurant.ID {
    c.JSON(403, gin.H{"error": "Not your order"})
    return
}

// Driver handler — prevents two drivers picking the same order
if order.DriverID != nil {
    c.JSON(409, gin.H{"error": "Order already picked up by another driver"})
    return
}
```

---

### Layer 5 — Descriptive Error Responses

When a transition is rejected, the API explains why:

```bash
# Try to cancel a DELIVERED order
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

## What Cannot Happen

| Attempted Action | Blocked By |
|---|---|
| Skip PLACED to PREPARING directly | State machine: no such transition |
| Customer cancels a PREPARING order | State machine: PREPARING -> CANCELLED not defined |
| Driver delivers without picking up | State machine: no CONFIRMED -> DELIVERED transition |
| Cancel a DELIVERED order | State machine: terminal state, no outgoing edges |
| Two drivers pick the same order | Handler: `if order.DriverID != nil` check |
| Customer cancel another customer's order | Handler: `if order.CustomerID != myID` check |
| Restaurant manage another restaurant's orders | Handler: `if order.RestaurantID != myRestaurant.ID` check |
| Customer reach restaurant endpoints | JWT role middleware: HTTP 403 |

---

## Audit Trail

Every successful state change is recorded in `order_status_histories`:

```sql
id | order_id | from_status | to_status  | changed_by | note           | created_at
1  |    1     |    PLACED   | CONFIRMED  |     2      | "Order received!" | 2026-02-21 05:58:07
2  |    1     |  CONFIRMED  | PREPARING  |     2      |      NULL         | 2026-02-21 05:58:09
3  |    1     |  PREPARING  | READY_...  |     2      | "Food is packed!" | 2026-02-21 05:59:10
```

Even if a bug bypassed all checks, there is a full audit trail to trace every state change back to the exact user who triggered it.

---

## Summary

The state machine is not just a validation check — it is the core contract of the system. Valid transitions are defined at compile time. Role middleware ensures only the right actor type can call each endpoint. Ownership checks prevent cross-user access. The result is a layered defense where an invalid state change is structurally impossible — not just discouraged.
