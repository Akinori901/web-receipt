-- receipts テーブル。
-- 領収書は発行時点のスナップショットが正。宛名・住所は顧客マスタへの参照ではなく
-- 「値」として直接カラムに保存する（発行後に顧客名が変わっても領収書は不変であるべき／
-- 領収書サービスを顧客管理サービスに依存させず単体完結にするため）。
CREATE TABLE IF NOT EXISTS receipts (
    id                  VARCHAR(36)  NOT NULL,
    owner_id            VARCHAR(64)  NOT NULL,
    recipient           VARCHAR(200) NOT NULL,
    recipient_address   VARCHAR(500) NULL,
    amount              BIGINT       NOT NULL,          -- 最小通貨単位（円）。float は使わない
    tax_category        VARCHAR(20)  NOT NULL,          -- standard10 / reduced8 / exempt
    registration_number VARCHAR(14)  NULL,              -- インボイス登録番号 T + 13桁（任意）
    proviso             VARCHAR(200) NULL,              -- 但し書き（任意）
    issued_on           DATE         NOT NULL,
    created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_receipts_owner_issued (owner_id, issued_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
