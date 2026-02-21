# Order State Machine

The order lifecycle is implemented as a finite state machine (FSM) with 7 states and 9 defined transitions. Every state change is validated through a central `CanTransition()` function before being persisted — no handler can bypass this check.

---

## State Diagram

```
                 +---------------------------------------------+
                 |   (Restaurant or Customer cancels)          |
                 v                                             |
  [PLACED] --> [CONFIRMED] --> [PREPARING] --> [READY_FOR_PICKUP]
      |              |
      |  (cancel)    |
      +--------------+-------------------------------------> [CANCELLED]  <- terminal

  [READY_FOR_PICKUP] --> [PICKED_UP] --> [DELIVERED]  <- terminal
         (Driver)             (Driver)
```

---

## State Definitions

| State | Meaning |
|---|---|
| `PLACED` | Customer placed an order. Awaiting restaurant response. |
| `CONFIRMED` | Restaurant accepted the order. |
| `PREPARING` | Kitchen is actively preparing the food. |
| `READY_FOR_PICKUP` | Food is ready. Waiting for a driver. |
| `PICKED_UP` | Driver has collected the food and is en route. |
| `DELIVERED` | Terminal. Customer received their order. |
| `CANCELLED` | Terminal. Order was cancelled by customer or restaurant. |

---

## Valid Transitions

| From | To | Actor | Trigger |
|---|---|---|---|
| PLACED | CONFIRMED | restaurant | Restaurant accepts the order |
| PLACED | CANCELLED | restaurant | Restaurant rejects |
| PLACED | CANCELLED | customer | Customer changes mind |
| CONFIRMED | PREPARING | restaurant | Kitchen starts cooking |
| CONFIRMED | CANCELLED | restaurant | Restaurant cancels after confirmation |
| CONFIRMED | CANCELLED | customer | Customer cancels before cooking |
| PREPARING | READY_FOR_PICKUP | restaurant | Food is packaged and ready |
| READY_FOR_PICKUP | PICKED_UP | driver | Driver picks up the order |
| PICKED_UP | DELIVERED | driver | Driver reaches the customer |

---

## Example: Restaurant Progressing an Order

### Step 1 — Confirm

```bash
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "CONFIRMED", "note": "Order received!" }'
```

**Response:**
```json
{
  "message": "Order status updated",
  "order_id": 1,
  "previous_status": "PLACED",
  "current_status": "CONFIRMED"
}
```

### Step 2 — Start Preparing

```bash
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "PREPARING" }'
```

**Response:**
```json
{
  "message": "Order status updated",
  "order_id": 1,
  "previous_status": "CONFIRMED",
  "current_status": "PREPARING"
}
```

### Step 3 — Mark Ready for Pickup

```bash
curl -X PUT http://localhost:8080/api/restaurant/orders/1/status \
  -H "Authorization: Bearer <restaurant_token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "READY_FOR_PICKUP", "note": "Food is packed!" }'
```

**Response:**
```json
{
  "message": "Order status updated",
  "order_id": 1,
  "previous_status": "PREPARING",
  "current_status": "READY_FOR_PICKUP"
}
```

---

## Implementation: O(1) Hash Map Lookup

Instead of nested if-else chains, all valid transitions are stored in a Go map keyed by `(from, to, actor)`:

```go
type transitionKey struct {
    From  OrderStatus
    To    OrderStatus
    Actor string
}

var transitionMap = map[transitionKey]bool{ ... }
```

`CanTransition()` does a single map lookup — O(1), no iteration, no branching.

---

## Audit Trail

Every successful transition automatically creates an `order_status_histories` record:

```sql
id | order_id | from_status | to_status | changed_by | note | created_at
1  |    1     |   PLACED    | CONFIRMED |     2      | "Order received!" | 2026-02-21 ...
2  |    1     |  CONFIRMED  | PREPARING |     2      |       NULL        | 2026-02-21 ...
```

---

## Terminal States

`DELIVERED` and `CANCELLED` have no outgoing transitions. Any attempt returns:

```json
{
  "error": "Invalid state transition",
  "current_state": "DELIVERED",
  "reason": "... Valid transitions from DELIVERED are: none (terminal state)"
}
```

The only exception is the admin emergency override endpoint (`PUT /api/admin/orders/:id/status`).
