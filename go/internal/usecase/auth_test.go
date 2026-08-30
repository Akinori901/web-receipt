package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/Akinori901/web-receipt/go/internal/usecase"
)

// 契約が interface になっていることの実利。
// Cognito も MySQL も無しでユースケースを完全に検証できる。

type fakeAuth struct {
	subject  string
	password string
}

func (f fakeAuth) SignIn(_ context.Context, _ entity.Email, password string) (entity.AuthTokens, error) {
	if password != f.password {
		return entity.AuthTokens{}, entity.ErrAuthFailed
	}
	return entity.AuthTokens{AccessToken: "access-" + f.subject, ExpiresIn: 3600}, nil
}

func (f fakeAuth) VerifyAccessToken(_ context.Context, token string) (entity.VerifiedIdentity, error) {
	if token != "access-"+f.subject {
		return entity.VerifiedIdentity{}, entity.ErrAuthFailed
	}
	return entity.VerifiedIdentity{Subject: f.subject, Email: "taro@example.com"}, nil
}

type inMemoryUsers struct {
	store map[string]entity.User
}

func newInMemoryUsers(seed ...entity.User) *inMemoryUsers {
	m := &inMemoryUsers{store: map[string]entity.User{}}
	for _, u := range seed {
		m.store[u.ID.String()] = u
	}
	return m
}

func (r *inMemoryUsers) FindByID(_ context.Context, id entity.UserID) (entity.User, error) {
	u, ok := r.store[id.String()]
	if !ok {
		return entity.User{}, entity.ErrUserNotFound
	}
	return u, nil
}

func (r *inMemoryUsers) FindByEmail(_ context.Context, email entity.Email) (entity.User, error) {
	for _, u := range r.store {
		if u.Email == email {
			return u, nil
		}
	}
	return entity.User{}, entity.ErrUserNotFound
}

func (r *inMemoryUsers) Save(_ context.Context, u entity.User) (entity.User, error) {
	r.store[u.ID.String()] = u
	return u, nil
}

func mustUser(t *testing.T, id, email string, active bool) entity.User {
	t.Helper()
	uid, err := entity.NewUserID(id)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	em, err := entity.NewEmail(email)
	if err != nil {
		t.Fatalf("Email: %v", err)
	}
	tid, _ := entity.NewTenantID("tenant-1")
	u, err := entity.NewUserWithTenant(uid, em, tid, entity.AdminRole())
	if err != nil {
		t.Fatalf("User: %v", err)
	}
	u.IsActive = active
	return u
}

// fakeTenants は usecase.TenantRepo の in-memory fake。
type fakeTenants struct {
	store map[string]entity.Tenant
}

func newFakeTenants() *fakeTenants { return &fakeTenants{store: map[string]entity.Tenant{}} }

func (f *fakeTenants) FindByID(_ context.Context, id entity.TenantID) (entity.Tenant, error) {
	t, ok := f.store[id.String()]
	if !ok {
		return entity.Tenant{}, entity.ErrTenantNotFound
	}
	return t, nil
}

func (f *fakeTenants) Save(_ context.Context, t entity.Tenant) (entity.Tenant, error) {
	f.store[t.ID.String()] = t
	return t, nil
}

// fixedID は usecase.IDGenerator の固定実装。
type fixedIDAuth struct{ id string }

func (f fixedIDAuth) NewID() string { return f.id }

// newAuthUC はテスト用に AuthUseCase を組み立てる。
func newAuthUC(auth usecase.Authenticator, users usecase.UserRepo) *usecase.AuthUseCase {
	return usecase.NewAuthUseCase(auth, users, newFakeTenants(), fixedIDAuth{id: "tenant-new"})
}

func TestSignInProvisionsLocalUserOnFirstSignIn(t *testing.T) {
	t.Parallel()

	users := newInMemoryUsers()
	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "correct-pass"}, users)

	out, err := uc.SignIn(context.Background(), "taro@example.com", "correct-pass")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if out.User.ID.String() != "sub-1" {
		t.Errorf("UserID = %q", out.User.ID.String())
	}
	if out.User.DisplayName.String() != "taro" {
		t.Errorf("DisplayName = %q", out.User.DisplayName.String())
	}
	// Cognito 側にしか居なかったユーザーがローカルにも作られている
	if _, ok := users.store["sub-1"]; !ok {
		t.Error("ローカルにユーザーが作られていない")
	}
}

func TestSignInRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "correct-pass"}, newInMemoryUsers())

	_, err := uc.SignIn(context.Background(), "taro@example.com", "wrong-pass")
	if !errors.Is(err, entity.ErrAuthFailed) {
		t.Errorf("err = %v, want ErrAuthFailed", err)
	}
}

func TestSignInRejectsDeactivatedUserEvenIfCognitoAccepts(t *testing.T) {
	t.Parallel()

	// 認証基盤の状態と、業務上の有効/無効は別の関心事である
	users := newInMemoryUsers(mustUser(t, "sub-1", "taro@example.com", false))
	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "correct-pass"}, users)

	_, err := uc.SignIn(context.Background(), "taro@example.com", "correct-pass")
	if !errors.Is(err, entity.ErrUserDeactivated) {
		t.Errorf("err = %v, want ErrUserDeactivated", err)
	}
}

func TestSignInRejectsInvalidEmailBeforeCallingAuth(t *testing.T) {
	t.Parallel()

	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "correct-pass"}, newInMemoryUsers())

	_, err := uc.SignIn(context.Background(), "not-an-email", "correct-pass")
	if !errors.Is(err, entity.ErrInvalidEmail) {
		t.Errorf("err = %v, want ErrInvalidEmail", err)
	}
}

func TestCurrentUserReturnsNotFoundWhenAbsent(t *testing.T) {
	t.Parallel()

	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "p"}, newInMemoryUsers())

	_, err := uc.CurrentUser(context.Background(), "access-sub-1")
	if !errors.Is(err, entity.ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

func TestCurrentUserRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	uc := newAuthUC(fakeAuth{subject: "sub-1", password: "p"}, newInMemoryUsers())

	_, err := uc.CurrentUser(context.Background(), "garbage")
	if !errors.Is(err, entity.ErrAuthFailed) {
		t.Errorf("err = %v, want ErrAuthFailed", err)
	}
}

func (r *inMemoryUsers) ListByTenant(_ context.Context, tid entity.TenantID) ([]entity.User, error) {
	var out []entity.User
	for _, u := range r.store {
		if u.TenantID == tid {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f fakeAuth) AdminCreateUser(_ context.Context, email entity.Email, _ string) (string, error) {
	// テストでは email をそのまま sub 代わりに返す（決定的）。
	return "sub-" + email.String(), nil
}
