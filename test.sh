#!/bin/bash

# Stop and remove existing container
docker stop postgres-test 2>/dev/null || true


docker rm postgres-test 2>/dev/null || true

export DB_HOST=127.0.0.1
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=testdb

# Start postgres container
docker run --name postgres-test \
    -e POSTGRES_PASSWORD=$DB_PASSWORD \
    -e POSTGRES_USER=$DB_USER \
    -e POSTGRES_DB=$DB_NAME \
    -p $DB_PORT:5432 \
    -d postgres:latest

# Wait for postgres to be ready
sleep 3

# Create schema
echo "Creating database schema..."
psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME" -f sql/schema.sql

# Insert test data
echo "Inserting test data..."
psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME" << EOF
-- Test Users
INSERT INTO users (name, password, email, is_admin) VALUES 
('admin', 'admin123', 'admin@test.com', true),
('user1', 'pass123', 'user1@test.com', false);

-- Test Products
INSERT INTO products (name, price, image_url, is_available) VALUES 
('Product 1', 19.99, 'http://example.com/img1.jpg', true),
('Product 2', 29.99, 'http://example.com/img2.jpg', true);

-- Test Blogs
INSERT INTO blogs (title, content, user_id, path) VALUES 
('First Blog', 'Content 1', 1, '/blog/first'),
('Second Blog', 'Content 2', 2, '/blog/second');

-- Test Orders
INSERT INTO orders (address, user_id) VALUES 
('123 Test St', 2);

-- Test Order Products
INSERT INTO order_products (order_id, product_id, quantity) VALUES 
(1, 1, 2),
(1, 2, 1);
EOF

# Verify container is running
if ! docker ps | grep -q postgres-test; then
    echo "Failed to start postgres container"
    exit 1
fi

echo "Test database is ready with schema and test data"
echo "Environment variables set:"
echo "DB_HOST=$DB_HOST"
echo "DB_PORT=$DB_PORT"
echo "DB_USER=$DB_USER"
echo "DB_NAME=$DB_NAME"

go build -o main ./cmd/main.go 
coproc CO { ./main; }

sleep 5
(curl -X GET http://localhost:8000/api/v1/products) || echo "Failed to start the server"

# Stop and remove the container
docker stop postgres-test
docker rm postgres-test

# Remove the binary
rm main