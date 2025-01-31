-- User queries
-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE name = $1 LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users;

-- name: CreateUser :one
INSERT INTO users (name, password, email)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $1, password = $2, email = $3
WHERE id = $4
RETURNING *;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = $1
RETURNING *;

-- Blog queries
-- name: GetBlog :one
SELECT * FROM blogs
WHERE id = $1 LIMIT 1;

-- name: GetBlogs :many
SELECT * FROM blogs;

-- name: CreateBlog :one
INSERT INTO blogs (title, content, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateBlog :one
UPDATE blogs
SET title = $1, content = $2, user_id = $3
WHERE id = $4
RETURNING *;

-- name: DeleteBlog :one
DELETE FROM blogs
WHERE id = $1
RETURNING *;

-- Product queries
-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1 LIMIT 1;

-- name: GetProducts :many
SELECT * FROM products;

-- name: CreateProduct :one
INSERT INTO products (name, price, image_url)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = $1, price = $2, image_url = $3
WHERE id = $4
RETURNING *;

-- name: DeleteProduct :one
DELETE FROM products
WHERE id = $1
RETURNING *;

-- Order queries
-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: GetOrders :many
SELECT * FROM orders;

-- name: CreateOrder :one
INSERT INTO orders (address, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateOrder :one
UPDATE orders
SET address = $1, user_id = $2
WHERE id = $3
RETURNING *;

-- name: DeleteOrder :one
DELETE FROM orders
WHERE id = $1
RETURNING *;

-- Order Product queries
-- name: GetOrderProduct :one
SELECT * FROM order_products
WHERE id = $1 LIMIT 1;

-- name: GetOrderProducts :many
SELECT * FROM order_products;

-- name: CreateOrderProduct :one
INSERT INTO order_products (order_id, product_id, quantity)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateOrderProduct :one
UPDATE order_products
SET order_id = $1, product_id = $2, quantity = $3
WHERE id = $4
RETURNING *;

-- name: DeleteOrderProduct :one
DELETE FROM order_products
WHERE id = $1
RETURNING *;

