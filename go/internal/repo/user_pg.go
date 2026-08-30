// Package repo は永続化の実装。usecase が定義した契約を満たす。
//
// **SQL / ドライバを知ってよいのはこの層だけ。**
// 返す直前に必ず DB の行 → entity へ変換する。
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// UserRepo は usecase.UserRepo の PostgreSQL 実装。
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo は依存を注入して組み立てる。
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const selectUserColumns = `SELECT id, email, display_name, is_active, tenant_id, role FROM users`

// FindByID は ID でユーザーを取得する。
//
// 見つからない場合は entity.ErrUserNotFound を返す。
// sql.ErrNoRows をそのまま上へ返さないこと。database/sql の型が漏れる。
func (r *UserRepo) FindByID(ctx context.Context, id entity.UserID) (entity.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserColumns+` WHERE id = $1`, id.String())
	return scanUserRow(row)
}

// FindByEmail は Email でユーザーを取得する。
func (r *UserRepo) FindByEmail(ctx context.Context, email entity.Email) (entity.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserColumns+` WHERE email = $1`, email.String())
	return scanUserRow(row)
}

// ListByTenant は組織に所属するユーザー一覧を返す。
func (r *UserRepo) ListByTenant(ctx context.Context, tenantID entity.TenantID) ([]entity.User, error) {
	rows, err := r.db.QueryContext(ctx, selectUserColumns+` WHERE tenant_id = $1 ORDER BY email`, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("ユーザー一覧取得に失敗: %w", err)
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ユーザー一覧取得に失敗: %w", err)
	}
	return users, nil
}

// Save はユーザーを永続化する（新規・更新の両方）。
func (r *UserRepo) Save(ctx context.Context, user entity.User) (entity.User, error) {
	const q = `
		INSERT INTO users (id, email, display_name, is_active, tenant_id, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			display_name = EXCLUDED.display_name,
			is_active = EXCLUDED.is_active,
			tenant_id = EXCLUDED.tenant_id,
			role = EXCLUDED.role`

	if _, err := r.db.ExecContext(ctx, q,
		user.ID.String(), user.Email.String(), user.DisplayName.String(), user.IsActive,
		user.TenantID.String(), user.Role.String(),
	); err != nil {
		return entity.User{}, fmt.Errorf("ユーザー保存に失敗: %w", err)
	}
	return user, nil
}

func scanUserRow(row *sql.Row) (entity.User, error) {
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, err
}

// scanUser は DB の行を entity へ変換する。この境界で永続化の都合を断ち切る。
func scanUser(s rowScanner) (entity.User, error) {
	var (
		rawID, rawEmail, rawName, rawTenant, rawRole string
		isActive                                     bool
	)
	if err := s.Scan(&rawID, &rawEmail, &rawName, &isActive, &rawTenant, &rawRole); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.User{}, entity.ErrUserNotFound
		}
		return entity.User{}, fmt.Errorf("ユーザー取得に失敗: %w", err)
	}

	id, err := entity.NewUserID(rawID)
	if err != nil {
		return entity.User{}, err
	}
	email, err := entity.NewEmail(rawEmail)
	if err != nil {
		return entity.User{}, err
	}
	name, err := entity.NewDisplayName(rawName)
	if err != nil {
		return entity.User{}, err
	}
	tenantID, err := entity.NewTenantID(rawTenant)
	if err != nil {
		return entity.User{}, err
	}
	role, err := entity.NewRole(rawRole)
	if err != nil {
		return entity.User{}, err
	}

	return entity.User{
		ID: id, Email: email, DisplayName: name, IsActive: isActive,
		TenantID: tenantID, Role: role,
	}, nil
}
