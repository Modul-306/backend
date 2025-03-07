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
INSERT INTO users (name, password, email, is_admin)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $1, password = $2, email = $3, is_admin = $4
WHERE id = $5 AND $6 = $5
RETURNING *;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = $1 AND $2 = $1
RETURNING *;

-- Blog queries
-- name: GetBlog :one
SELECT * FROM blogs
WHERE id = $1 LIMIT 1;

-- name: GetBlogs :many
SELECT * FROM blogs;

-- name: CreateBlog :one
INSERT INTO blogs (title, content, user_id, path)
SELECT $1, $2, $3, $4
WHERE $5 IN (SELECT id FROM users WHERE is_admin = true)
RETURNING *;

-- name: UpdateBlog :one
UPDATE blogs
SET title = $1, 
    content = $2, 
    user_id = $3,
    path = $4,
    modified_at = CURRENT_TIMESTAMP
WHERE blogs.id = $5 AND ($5 IN (SELECT id FROM users WHERE is_admin = true) OR $5 = $3)
RETURNING *;

-- name: DeleteBlog :one
DELETE FROM blogs
WHERE blogs.id = $1
AND ($2 IN (SELECT id FROM users WHERE is_admin = true) OR $2 = blogs.user_id)
RETURNING *;

-- Product Queries
-- -----------------------------------------------------------------------------

-- Get a single product by ID
-- name: GetProduct :one
SELECT * 
FROM   products 
WHERE  id = $1 
LIMIT  1;

-- Get all products
-- name: GetProducts :many
SELECT * 
FROM   products;

-- Create new product (admin only)
-- name: CreateProduct :one
INSERT INTO products (
    name,
    price,
    image_url,
    is_available
)
SELECT $1, $2, $3, $4
WHERE  $5 IN (SELECT id FROM users WHERE is_admin = true)
RETURNING *;

-- Update existing product (admin only)
-- name: UpdateProduct :one
UPDATE products
SET    name = $1,
       price = $2,
       image_url = $3,
       is_available = $4,
       modified_at = CURRENT_TIMESTAMP
WHERE products.id = $5
  AND $6 IN (SELECT id FROM users WHERE is_admin = true)
RETURNING *;

-- Delete product (admin only)
-- name: DeleteProduct :one
DELETE FROM products
WHERE  products.id = $1
  AND  $2 IN (SELECT id FROM users WHERE is_admin = true)
RETURNING *;

-- Order queries
-- name: GetOrder :one
SELECT * FROM orders
WHERE orders.id = $1
AND (
    orders.user_id = $2
    OR
    $2 IN (SELECT id FROM users WHERE is_admin = true)
)
LIMIT 1;

-- name: GetOrders :many
SELECT * FROM orders
WHERE $1 = orders.user_id 
OR $1 IN
(SELECT id FROM users WHERE is_admin = true);

-- name: CreateOrder :one
INSERT INTO orders (address, user_id, is_completed)
VALUES ($1, $2, false)
RETURNING *;

-- name: UpdateOrder :one
UPDATE orders
SET address = $1, 
    user_id = $2,
    is_completed = $3
WHERE orders.id = $4 AND ($5 = orders.user_id OR $5 IN
(SELECT id FROM users WHERE is_admin = true))
RETURNING *;

-- name: DeleteOrder :one
DELETE FROM orders
WHERE orders.id = $1 AND $2 = orders.user_id 
OR $2 IN
(SELECT id FROM users WHERE is_admin = true)
RETURNING *;

-- Order Product queries
-- name: GetOrderProduct :one
SELECT * FROM order_products
WHERE order_products.id = $1 AND (
    $2 IN (
        SELECT id FROM users 
        WHERE is_admin = true
    ) 
    OR $2 = (
        SELECT user_id FROM orders 
        WHERE id = (
            SELECT order_id FROM order_products 
            WHERE id = $1
        )
    )
)
LIMIT 1;

-- name: GetOrderProducts :many
SELECT * FROM order_products
WHERE $1 IN (
    SELECT id FROM users 
    WHERE is_admin = true
) OR $1 = (
    SELECT user_id FROM orders 
    WHERE id = order_products.order_id
);

-- name: CreateOrderProduct :one
INSERT INTO order_products (order_id, product_id, quantity)
SELECT $1, $2, $3
WHERE $4 IN (
    SELECT id FROM users 
    WHERE is_admin = true
) OR $4 = (
    SELECT user_id FROM orders 
    WHERE id = $1
)
RETURNING *;

-- name: UpdateOrderProduct :one
UPDATE order_products
SET order_id = $1, product_id = $2, quantity = $3
WHERE order_products.id = $4 AND (
    $5 IN (
        SELECT id FROM users 
        WHERE is_admin = true
    ) 
    OR $5 = (
        SELECT user_id FROM orders 
        WHERE id = (
            SELECT order_id FROM order_products 
            WHERE id = $4
        )
    )
)
RETURNING *;

-- name: DeleteOrderProduct :one
DELETE FROM order_products
WHERE order_products.id = $1 AND (
    $2 IN (
        SELECT id FROM users 
        WHERE is_admin = true
    ) 
    OR $2 = (
        SELECT user_id FROM orders 
        WHERE id = (
            SELECT order_id FROM order_products 
            WHERE id = $1
        )
    )
)
RETURNING *;

