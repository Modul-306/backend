-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1 LIMIT 1;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug)
VALUES ($1, $2)
RETURNING *;

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
