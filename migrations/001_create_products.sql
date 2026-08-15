CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL CHECK (price >= 0),
    currency CHAR(3) NOT NULL,
    stock INTEGER NOT NULL CHECK (stock >= 0)
);

INSERT INTO products (
    name,
    description,
    price,
    currency,
    stock
)
VALUES
    (
        'MacBook Pro',
        'Apple laptop',
        150000,
        'EUR',
        10
    ),
    (
        'Mechanical Keyboard',
        'Mechanical keyboard',
        12000,
        'EUR',
        50
    );