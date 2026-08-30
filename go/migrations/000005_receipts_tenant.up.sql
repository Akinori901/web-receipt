-- receipts に組織スコープを追加。
-- 領収書は組織に属し、組織メンバー全員が履歴を見られる（認可は tenant_id で判定）。
-- 既存行は既定組織に紐付ける（開発中データの移行）。
ALTER TABLE receipts
    ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64) NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';

ALTER TABLE receipts
    ADD CONSTRAINT fk_receipts_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id);

CREATE INDEX IF NOT EXISTS idx_receipts_tenant_issued ON receipts (tenant_id, issued_on);
