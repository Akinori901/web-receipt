-- users テーブル。
-- Cognito が認証の正であり、このテーブルは初回サインイン時に自前プロビジョニングする
-- ローカルの写し。id には Cognito の sub をそのまま入れる。
CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(64)  NOT NULL,
    email        VARCHAR(254) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
