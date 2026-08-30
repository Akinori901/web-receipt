package usecase

import (
	"context"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// UpdateTenantInput は発行者情報の更新入力（プリミティブ）。
type UpdateTenantInput struct {
	CorporateName      string
	Address            string
	Phone              string
	Email              string
	RegistrationNumber string
}

// TenantUseCase は組織（発行者情報）のユースケース。
type TenantUseCase struct {
	tenants TenantRepo
}

func NewTenantUseCase(tenants TenantRepo) *TenantUseCase {
	return &TenantUseCase{tenants: tenants}
}

// Get は自組織の発行者情報を返す。
func (uc *TenantUseCase) Get(ctx context.Context, user entity.User) (entity.Tenant, error) {
	return uc.tenants.FindByID(ctx, user.TenantID)
}

// Update は自組織の発行者情報を更新する。admin のみ許可。
func (uc *TenantUseCase) Update(ctx context.Context, user entity.User, in UpdateTenantInput) (entity.Tenant, error) {
	if !user.IsAdmin() {
		return entity.Tenant{}, entity.ErrTenantForbidden
	}

	name, err := entity.NewCorporateName(in.CorporateName)
	if err != nil {
		return entity.Tenant{}, err
	}
	address, err := entity.NewAddress(in.Address)
	if err != nil {
		return entity.Tenant{}, err
	}
	phone, err := entity.NewPhone(in.Phone)
	if err != nil {
		return entity.Tenant{}, err
	}
	email, err := entity.NewOptionalEmail(in.Email)
	if err != nil {
		return entity.Tenant{}, err
	}
	registration, err := entity.NewRegistrationNumber(in.RegistrationNumber)
	if err != nil {
		return entity.Tenant{}, err
	}

	tenant := entity.NewTenant(user.TenantID, name, address, phone, email, registration)
	return uc.tenants.Save(ctx, tenant)
}
