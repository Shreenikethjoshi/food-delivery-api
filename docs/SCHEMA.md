# Database Schema

SQLite is used for zero-configuration simplicity. The database file (`food_delivery.db`) is created automatically on first run — no database server installation required.

---

## Tables

### users
```sql
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,              -- bcrypt hashed
    role          TEXT NOT NULL CHECK(role IN ('customer', 'restaurant', 'driver', 'admin')),
    phone         TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### restaurants
```sql
CREATE TABLE restaurants (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id    INTEGER NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    cuisine     TEXT,
    address     TEXT NOT NULL,
    description TEXT,
    is_open     BOOLEAN DEFAULT TRUE,
    rating      REAL DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### menu_items
```sql
CREATE TABLE menu_items (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id),
    name          TEXT NOT NULL,
    description   TEXT,
    price         REAL NOT NULL CHECK(price > 0),
    category      TEXT,
    is_available  BOOLEAN DEFAULT TRUE,
    is_veg        BOOLEAN DEFAULT FALSE,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### orders
```sql
CREATE TABLE orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id      INTEGER NOT NULL REFERENCES users(id),
    restaurant_id    INTEGER NOT NULL REFERENCES restaurants(id),
    driver_id        INTEGER REFERENCES users(id),           -- NULL until driver picks up
    status           TEXT NOT NULL DEFAULT 'PLACED'
                     CHECK(status IN (
                         'PLACED', 'CONFIRMED', 'PREPARING',
                         'READY_FOR_PICKUP', 'PICKED_UP',
                         'DELIVERED', 'CANCELLED'
                     )),
    total_price           REAL NOT NULL,
    delivery_address      TEXT NOT NULL,
    notes                 TEXT,
    estimated_time_minutes INTEGER DEFAULT 30,               -- novelty: auto-calculated ETA
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### order_items
```sql
CREATE TABLE order_items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id    INTEGER NOT NULL REFERENCES orders(id),
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id),
    quantity    INTEGER NOT NULL CHECK(quantity > 0),
    price       REAL NOT NULL,    -- snapshot of price at time of order
    name        TEXT NOT NULL     -- snapshot of item name at time of order
);
```

### order_status_histories
```sql
CREATE TABLE order_status_histories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id    INTEGER NOT NULL REFERENCES orders(id),
    from_status TEXT,             -- NULL on first entry (PLACED)
    to_status   TEXT NOT NULL,
    changed_by  INTEGER REFERENCES users(id),
    note        TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Entity Relationships

```
users ──< restaurants      (one user owns one restaurant)
users ──< orders           (as customer_id)
users ──< orders           (as driver_id, nullable)
restaurants ──< menu_items
restaurants ──< orders
orders ──< order_items
menu_items ──< order_items
orders ──< order_status_histories
```

---

## Order Status Values (State Machine)

| Status | Description |
|---|---|
| `PLACED` | Initial state when customer submits order |
| `CONFIRMED` | Restaurant accepted the order |
| `PREPARING` | Kitchen is cooking |
| `READY_FOR_PICKUP` | Food packaged, awaiting driver |
| `PICKED_UP` | Driver collected the order |
| `DELIVERED` | Terminal -- delivered to customer |
| `CANCELLED` | Terminal -- cancelled by restaurant or customer |

---

*Database is auto-migrated by GORM on every server start — schema is always in sync with the models.*
