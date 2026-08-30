package usecase

import (
	"context"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// InviteMemberInput はメンバー招待の入力。
type InviteMemberInput struct {
	Email    string
	Password string // 初期パスワード（発行側が本人へ連絡する）
}

// MemberUseCase は組織メンバーの管理ユースケース。
type MemberUseCase struct {
	auth  Authenticator
	users UserRepo
}

func NewMemberUseCase(auth Authenticator, users UserRepo) *MemberUseCase {
	return &MemberUseCase{auth: auth, users: users}
}

// Invite は admin が自組織にメンバーを登録する（Cognito 作成 + ローカル行の先行作成）。
//
// admin の組織に member として紐付ける。member が初回サインインすると
// auth.resolveUser がこの先行作成行を FindByID で当て、role/tenant を保持したまま返す。
func (uc *MemberUseCase) Invite(ctx context.Context, admin entity.User, in InviteMemberInput) (entity.User, error) {
	if !admin.IsAdmin() {
		return entity.User{}, entity.ErrTenantForbidden
	}
	email, err := entity.NewEmail(in.Email)
	if err != nil {
		return entity.User{}, err
	}

	sub, err := uc.auth.AdminCreateUser(ctx, email, in.Password)
	if err != nil {
		return entity.User{}, err
	}
	uid, err := entity.NewUserID(sub)
	if err != nil {
		return entity.User{}, err
	}
	member, err := entity.NewUserWithTenant(uid, email, admin.TenantID, entity.MemberRole())
	if err != nil {
		return entity.User{}, err
	}
	return uc.users.Save(ctx, member)
}

// List は自組織のメンバー一覧を返す。
func (uc *MemberUseCase) List(ctx context.Context, user entity.User) ([]entity.User, error) {
	return uc.users.ListByTenant(ctx, user.TenantID)
}
