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
SELECT * FROM tenants 
WHERE (name ILIKE '%' || $1 || '%' OR COALESCE(category, '') ILIKE '%' || $1 || '%')
AND (COALESCE(category, '') = $2 OR $2 = '')
ORDER BY name;

-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products 
WHERE tenant_id = $1 
AND (name ILIKE '%' || $2 || '%' OR COALESCE(category, '') ILIKE '%' || $2 || '%')
AND (COALESCE(category, '') = $3 OR $3 = '')
ORDER BY created_at DESC;

-- name: CreateProduct :one
INSERT INTO products (tenant_id, name, description, price, stock, image_url, category)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products 
SET name = $2, description = $3, price = $4, stock = $5, image_url = $6, category = $7
WHERE id = $1 AND tenant_id = $8
RETURNING *;

-- name: GetUserDiscount :one
SELECT ld.discount_percent 
FROM loyalty_discounts ld
JOIN users u ON u.loyalty_tier = ld.tier_name
WHERE u.id = $1;

-- name: UpdateUserLoyaltyTier :one
UPDATE users SET loyalty_tier = $2 WHERE id = $1 RETURNING *;

-- name: ListCategories :many
SELECT DISTINCT category FROM products WHERE tenant_id = $1;

-- name: ListTenantCategories :many
SELECT DISTINCT category FROM tenants;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1 LIMIT 1;

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

-- name: GetBlog :one
SELECT * FROM blogs WHERE id = $1 LIMIT 1;

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
SELECT o.*, u.full_name, u.email, u.street, u.zip_code, u.city
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.tenant_id = $1 
ORDER BY o.created_at DESC;

-- name: ListOrdersByUser :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetOrderItems :many
SELECT oi.*, p.name as product_name 
FROM order_items oi
JOIN products p ON oi.product_id = p.id
WHERE oi.order_id = $1;

-- name: GetRevenueByDay :many
SELECT date_trunc('day', created_at)::date as day, SUM(total_amount::numeric) as revenue
FROM orders
WHERE tenant_id = $1 AND status = 'completed'
GROUP BY day
ORDER BY day DESC
LIMIT 30;

-- name: GetTopSellingProducts :many
SELECT p.id, p.name, SUM(oi.quantity) as total_sold
FROM order_items oi
JOIN products p ON oi.product_id = p.id
JOIN orders o ON oi.order_id = o.id
WHERE o.tenant_id = $1 AND o.status = 'completed'
GROUP BY p.id, p.name
ORDER BY total_sold DESC
LIMIT 5;

-- name: CreateReview :one
INSERT INTO product_reviews (product_id, user_id, rating, comment)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListReviewsByProduct :many
SELECT pr.*, u.email as user_email
FROM product_reviews pr
JOIN users u ON pr.user_id = u.id
WHERE pr.product_id = $1
ORDER BY pr.created_at DESC;

-- name: GetAverageRating :one
SELECT AVG(rating)::float as avg_rating, COUNT(*)::int as review_count
FROM product_reviews
WHERE product_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders SET status = $2 WHERE id = $1 AND tenant_id = $3 RETURNING *;

-- name: SetTenantOwner :one
UPDATE tenants SET owner_id = $2 WHERE id = $1 RETURNING *;

-- name: AddTenantOwner :exec
INSERT INTO tenant_owners (tenant_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTenantOwner :exec
DELETE FROM tenant_owners WHERE tenant_id = $1 AND user_id = $2;

-- name: ListTenantOwners :many
SELECT u.id, u.email, u.role
FROM users u
JOIN tenant_owners t_o ON u.id = t_o.user_id
WHERE t_o.tenant_id = $1;

-- name: IsTenantOwner :one
SELECT EXISTS (
    SELECT 1 FROM tenant_owners 
    WHERE tenant_id = $1 AND user_id = $2
);

-- name: UpdateTenantAppearance :one
UPDATE tenants SET cover_url = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: UpdateUserRole :one
UPDATE users SET role = $2 WHERE id = $1 RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users SET full_name = $2, street = $3, zip_code = $4, city = $5 WHERE id = $1 RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
