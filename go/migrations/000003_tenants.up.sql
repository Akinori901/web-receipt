-- tenants テーブル（組織 = 発行者情報の器）。
-- 領収書の発行者情報（法人名・住所・電話・メール・インボイス登録番号）を組織が1つ持つ。
CREATE TABLE IF NOT EXISTS tenants (
    id                  VARCHAR(64)  PRIMARY KEY,
    corporate_name      VARCHAR(200) NOT NULL,
    address             VARCHAR(500),
    phone               VARCHAR(30),
    email               VARCHAR(254),
    registration_number VARCHAR(14),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 既定テナント。既存の users / receipts を紐付ける移行先。
INSERT INTO tenants (id, corporate_name)
VALUES ('00000000-0000-0000-0000-000000000000', '既定組織')
ON CONFLICT (id) DO NOTHING;
