CREATE TABLE IF NOT EXISTS orders (
    order_uid UUID PRIMARY KEY,
    track_number TEXT NOT NULL,
    entry TEXT NOT NULL,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT NOT NULL,
    delivery_service TEXT,
    shardkey TEXT,
    sm_id INTEGER,
    date_created TIMESTAMP DEFAULT now(),
    oof_shard TEXT
);

CREATE TABLE IF NOT EXISTS deliveries (
    id SERIAL PRIMARY KEY,
    order_uid UUID NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    zip TEXT,
    city TEXT NOT NULL,
    address TEXT NOT NULL,
    region TEXT,
    email TEXT
);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    order_uid UUID NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction TEXT NOT NULL,
    request_id TEXT,
    currency CHAR(3) NOT NULL,
    provider TEXT,
    amount NUMERIC NOT NULL,
    payment_dt TIMESTAMP,
    bank TEXT,
    delivery_cost NUMERIC,
    goods_total NUMERIC,
    custom_fee NUMERIC
);

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    order_uid UUID NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id INTEGER,
    track_number TEXT,
    price NUMERIC,
    rid TEXT,
    name TEXT,
    sale INTEGER,
    size TEXT,
    total_price NUMERIC,
    nm_id INTEGER,
    brand TEXT,
    status INTEGER
);

ALTER TABLE deliveries ADD CONSTRAINT deliveries_order_uid_unique UNIQUE(order_uid);
ALTER TABLE payments ADD CONSTRAINT payments_order_uid_unique UNIQUE(order_uid);