CREATE TABLE orders (
    id UUID PRIMARY KEY,
    item TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_created_at ON orders(created_at);