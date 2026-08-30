-- users に組織所属と役割を追加。
-- 既存ユーザーは既定組織の admin として扱う（開発中データの移行）。
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64) NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    ADD COLUMN IF NOT EXISTS role      VARCHAR(16) NOT NULL DEFAULT 'admin';

-- 組織への外部キー。
ALTER TABLE users
    ADD CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id);

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);
