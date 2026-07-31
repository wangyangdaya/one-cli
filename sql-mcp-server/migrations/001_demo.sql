CREATE SCHEMA IF NOT EXISTS mcp_demo;

CREATE TABLE IF NOT EXISTS mcp_demo.customers (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    joined_on DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS mcp_demo.orders (
    id BIGINT PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES mcp_demo.customers(id),
    ordered_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    total NUMERIC(12, 2) NOT NULL
);

INSERT INTO mcp_demo.customers (id, name, email, joined_on)
VALUES
    (1, 'Ada Lovelace', 'ada@example.test', DATE '2026-01-10'),
    (2, 'Grace Hopper', 'grace@example.test', DATE '2026-02-14'),
    (3, 'Linus Torvalds', 'linus@example.test', DATE '2026-03-20')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    joined_on = EXCLUDED.joined_on;

INSERT INTO mcp_demo.orders (id, customer_id, ordered_at, status, total)
VALUES
    (101, 1, TIMESTAMPTZ '2026-07-01 09:00:00+00', 'paid', 125.50),
    (102, 1, TIMESTAMPTZ '2026-07-10 10:30:00+00', 'shipped', 89.00),
    (103, 2, TIMESTAMPTZ '2026-07-15 14:15:00+00', 'paid', 240.25),
    (104, 3, TIMESTAMPTZ '2026-07-20 08:45:00+00', 'cancelled', 42.00)
ON CONFLICT (id) DO UPDATE SET
    customer_id = EXCLUDED.customer_id,
    ordered_at = EXCLUDED.ordered_at,
    status = EXCLUDED.status,
    total = EXCLUDED.total;

