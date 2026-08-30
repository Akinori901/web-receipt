-- handovers テーブル（領収書の受け渡し管理）。
-- 領収書ごとに1つ。発行側が閲覧ID/PWを発行して受領側へ連絡し、
-- 受領側は非認証でアクセスして PDF 出力・宛名確定を行う。
-- アクセス状況（URL閲覧数/PDF出力数）を記録する。
CREATE TABLE IF NOT EXISTS handovers (
    receipt_id          VARCHAR(36)  PRIMARY KEY REFERENCES receipts (id) ON DELETE CASCADE,
    view_id             VARCHAR(64)  NOT NULL UNIQUE,
    view_password       VARCHAR(64)  NOT NULL,          -- 平文保存（要件どおり・発行側が連絡）
    url_view_count      INTEGER      NOT NULL DEFAULT 0,
    pdf_count           INTEGER      NOT NULL DEFAULT 0,
    recipient_confirmed BOOLEAN      NOT NULL DEFAULT FALSE,
    recipient           VARCHAR(200),                   -- 受領側が確定した宛名（未確定は NULL）
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);
