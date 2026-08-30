package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/Akinori901/web-receipt/go/internal/usecase"
)

func TestMemberUseCase_Invite_ByAdmin(t *testing.T) {
	t.Parallel()
	users := newInMemoryUsers()
	uc := usecase.NewMemberUseCase(fakeAuth{}, users)
	admin := mustUser(t, "admin-1", "admin@example.com", true) // AdminRole

	member, err := uc.Invite(context.Background(), admin, usecase.InviteMemberInput{
		Email: "member@example.com", Password: "Passw0rd!",
	})
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}
	// 招待された member は admin と同じテナントに、member ロールで作られる。
	if member.TenantID != admin.TenantID {
		t.Errorf("member のテナントが admin と異なる")
	}
	if member.IsAdmin() {
		t.Error("招待された member が admin になっている")
	}
	if member.Email.String() != "member@example.com" {
		t.Errorf("member email = %q", member.Email.String())
	}
}

func TestMemberUseCase_Invite_RejectedForMember(t *testing.T) {
	t.Parallel()
	users := newInMemoryUsers()
	uc := usecase.NewMemberUseCase(fakeAuth{}, users)

	// member ロールのユーザーは招待できない。
	uid, _ := entity.NewUserID("member-x")
	em, _ := entity.NewEmail("mx@example.com")
	tid, _ := entity.NewTenantID("tenant-1")
	member, _ := entity.NewUserWithTenant(uid, em, tid, entity.MemberRole())

	_, err := uc.Invite(context.Background(), member, usecase.InviteMemberInput{
		Email: "another@example.com", Password: "Passw0rd!",
	})
	if !errors.Is(err, entity.ErrTenantForbidden) {
		t.Errorf("member の招待で err=%v, want ErrTenantForbidden", err)
	}
}

func TestMemberUseCase_List(t *testing.T) {
	t.Parallel()
	admin := mustUser(t, "admin-1", "admin@example.com", true)
	users := newInMemoryUsers(admin)
	uc := usecase.NewMemberUseCase(fakeAuth{}, users)

	_, err := uc.Invite(context.Background(), admin, usecase.InviteMemberInput{
		Email: "member@example.com", Password: "Passw0rd!",
	})
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}

	list, err := uc.List(context.Background(), admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// admin 自身 + 招待した member の2名。
	if len(list) != 2 {
		t.Errorf("メンバー数 = %d, want 2", len(list))
	}
}
