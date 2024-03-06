CREATE TABLE orders (
                        id SERIAL PRIMARY KEY,
                        data jsonb UNIQUE
);