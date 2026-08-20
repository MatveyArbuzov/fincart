CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_one_draft_per_user
ON orders (user_id)
WHERE status = 'draft';