-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1 LIMIT 1;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateTenantIcon :one
UPDATE tenants SET icon_url = $2 WHERE id = $1 RETURNING *;

-- name: UpdateTenant :one
UPDATE tenants SET name = $2, slug = $3 WHERE id = $1 RETURNING *;

-- name: DeleteTenant :exec
DELETE FROM tenants WHERE id = $1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY name;

-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: CreateProduct :one
INSERT INTO products (tenant_id, name, description, price, stock, image_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListProducts :many
SELECT * FROM products WHERE tenant_id = $1 ORDER BY created_at DESC;

-- name: UpdateProduct :one
UPDATE products 
SET name = $2, description = $3, price = $4, stock = $5, image_url = $6
WHERE id = $1 AND tenant_id = $7
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1 AND tenant_id = $2;

-- name: UpdateProductStock :one
UPDATE products SET stock = stock - $2
WHERE id = $1 AND tenant_id = $3 AND stock >= $2
RETURNING *;

-- name: CreateBlog :one
INSERT INTO blogs (tenant_id, title, content_md)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListBlogs :many
SELECT * FROM blogs WHERE tenant_id = $1 ORDER BY published_at DESC;

-- name: UpdateBlog :one
UPDATE blogs
SET title = $2, content_md = $3
WHERE id = $1 AND tenant_id = $4
RETURNING *;

-- name: DeleteBlog :exec
DELETE FROM blogs WHERE id = $1 AND tenant_id = $2;

-- name: CreateOrder :one
INSERT INTO orders (tenant_id, user_id, status, total_amount)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, price_at_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListOrdersByTenant :many
SELECT * FROM orders WHERE tenant_id = $1 ORDER BY created_at DESC;

-- name: ListOrdersByUser :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetOrderItems :many
SELECT oi.*, p.name as product_name 
FROM order_items oi
JOIN products p ON oi.product_id = p.id
WHERE oi.order_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders SET status = $2 WHERE id = $1 AND tenant_id = $3 RETURNING *;

-- name: SetTenantOwner :one
UPDATE tenants SET owner_id = $2 WHERE id = $1 RETURNING *;

-- name: UpdateTenantAppearance :one
UPDATE tenants SET cover_url = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: UpdateUserRole :one
UPDATE users SET role = $2 WHERE id = $1 RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
