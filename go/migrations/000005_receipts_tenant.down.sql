ALTER TABLE receipts DROP CONSTRAINT IF EXISTS fk_receipts_tenant;
DROP INDEX IF EXISTS idx_receipts_tenant_issued;
ALTER TABLE receipts DROP COLUMN IF EXISTS tenant_id;
