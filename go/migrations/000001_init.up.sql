-- users テーブル（PostgreSQL）。
-- Cognito が認証の正であり、このテーブルは初回サインイン時に自前プロビジョニングする
-- ローカルの写し。id には Cognito の sub をそのまま入れる。
CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(64)  PRIMARY KEY,
    email        VARCHAR(254) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
