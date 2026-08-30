// Package http は外部との接点。HTTP の語彙を扱ってよい唯一の層。
//
// controller がやってよいのは 3 つだけ:
//  1. 入力の検証
//  2. usecase の呼び出し
//  3. 応答の組み立て（ドメインエラー → HTTP ステータスの変換）
//
// ビジネスロジックを書かないこと。
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/Akinori901/web-receipt/go/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// Router は HTTP ルーティングを組み立てる。
type Router struct {
	auth           *usecase.AuthUseCase
	health         *usecase.HealthUseCase
	receipt        *usecase.ReceiptUseCase
	tenant         *usecase.TenantUseCase
	member         *usecase.MemberUseCase
	handover       *usecase.HandoverUseCase
	allowedOrigins []string // CORS で許可するフロントのオリジン
}

// NewRouter は依存を注入して組み立てる。
func NewRouter(
	auth *usecase.AuthUseCase,
	health *usecase.HealthUseCase,
	receipt *usecase.ReceiptUseCase,
	tenant *usecase.TenantUseCase,
	member *usecase.MemberUseCase,
	handover *usecase.HandoverUseCase,
	allowedOrigins []string,
) *Router {
	return &Router{
		auth: auth, health: health, receipt: receipt, tenant: tenant,
		member: member, handover: handover, allowedOrigins: allowedOrigins,
	}
}

// Handler はルータを返す。
//
// *chi.Mux をそのまま返すのは、Lambda アダプタ (aws-lambda-go-api-proxy) が
// 具象型を要求するため。http.Handler で包むと Lambda 側で使えない。
func (rt *Router) Handler() *chi.Mux {
	r := chi.NewRouter()
	// フロント（別オリジン）からのブラウザ fetch を許可する。
	// 許可オリジンは環境変数から注入する（ローカル=localhost:5173 / 本番=CloudFront）。
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   rt.allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", rt.getHealth)
		r.Get("/health/live", rt.getLiveness)
		r.Post("/auth/sign-in", rt.postSignIn)
		r.Get("/auth/me", rt.getCurrentUser)

		r.Get("/tenant", rt.getTenant)
		r.Put("/tenant", rt.updateTenant)

		r.Post("/members", rt.inviteMember)
		r.Get("/members", rt.listMembers)

		r.Route("/receipts", func(r chi.Router) {
			r.Post("/", rt.postReceipt)
			r.Get("/", rt.listReceipts)
			r.Get("/{id}", rt.getReceipt)
			r.Post("/{id}/reissue", rt.reissueReceipt)
			r.Get("/{id}/pdf", rt.getReceiptPDF)
			// 発行側（認証・カウント非加算）: 閲覧ID/PW発行・アクセス状況。
			r.Post("/{id}/handover", rt.issueHandover)
			r.Get("/{id}/handover", rt.getHandoverStatus)
		})

		// 受領側（非認証・カウント加算）: viewID/PW で閲覧・PDF出力・宛名確定。
		// authUser を通さないため、発行側の閲覧とカウント経路が分離される。
		r.Route("/handover", func(r chi.Router) {
			r.Post("/{viewID}/view", rt.publicView)
			r.Post("/{viewID}/pdf", rt.publicPDF)
			r.Post("/{viewID}/confirm", rt.publicConfirm)
		})
	})
	return r
}

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	IsActive    bool   `json:"is_active"`
	TenantID    string `json:"tenant_id"`
	Role        string `json:"role"`
}

type signInResponse struct {
	AccessToken  string       `json:"access_token"`
	IDToken      string       `json:"id_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int32        `json:"expires_in"`
	User         userResponse `json:"user"`
}

type componentResponse struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

type healthResponse struct {
	Healthy    bool                `json:"healthy"`
	Components []componentResponse `json:"components"`
}

func (rt *Router) postSignIn(w http.ResponseWriter, r *http.Request) {
	var req signInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が不正です")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "メールアドレスとパスワード(8文字以上)は必須です")
		return
	}

	out, err := rt.auth.SignIn(r.Context(), req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, signInResponse{
		AccessToken:  out.Tokens.AccessToken,
		IDToken:      out.Tokens.IDToken,
		RefreshToken: out.Tokens.RefreshToken,
		ExpiresIn:    out.Tokens.ExpiresIn,
		User:         toUserResponse(out.User),
	})
}

func (rt *Router) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "Authorization ヘッダがありません")
		return
	}

	user, err := rt.auth.CurrentUser(r.Context(), token)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (rt *Router) getHealth(w http.ResponseWriter, r *http.Request) {
	status := rt.health.Check(r.Context())

	components := make([]componentResponse, 0, len(status.Components))
	for _, c := range status.Components {
		components = append(components, componentResponse{
			Name: c.Name, State: string(c.State), Detail: c.Detail,
		})
	}

	// 依存が落ちていれば 503 を返す。ALB はステータスコードで判定するため、
	// 本文が返せていても 200 にしないこと。
	code := http.StatusOK
	if !status.IsHealthy() {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, healthResponse{Healthy: status.IsHealthy(), Components: components})
}

// getLiveness はプロセスの生存のみを見る（依存を確認しない）。
func (rt *Router) getLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── 領収書 ────────────────────────────────────────────────

type createReceiptRequest struct {
	Recipient          string `json:"recipient"`
	RecipientAddress   string `json:"recipient_address"`
	Amount             int64  `json:"amount"`
	TaxCategory        string `json:"tax_category"`
	RegistrationNumber string `json:"registration_number"`
	Proviso            string `json:"proviso"`
	IssuedOn           string `json:"issued_on"`
}

type receiptResponse struct {
	ID                 string `json:"id"`
	Recipient          string `json:"recipient"`
	RecipientAddress   string `json:"recipient_address"`
	Amount             int64  `json:"amount"`
	TaxCategory        string `json:"tax_category"`
	TaxAmount          int64  `json:"tax_amount"`
	NetAmount          int64  `json:"net_amount"`
	RegistrationNumber string `json:"registration_number"`
	Proviso            string `json:"proviso"`
	IssuedOn           string `json:"issued_on"`
}

func (rt *Router) postReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	var req createReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が不正です")
		return
	}
	receipt, err := rt.receipt.Create(r.Context(), user, usecase.CreateReceiptInput{
		Recipient:          req.Recipient,
		RecipientAddress:   req.RecipientAddress,
		Amount:             req.Amount,
		TaxCategory:        req.TaxCategory,
		RegistrationNumber: req.RegistrationNumber,
		Proviso:            req.Proviso,
		IssuedOn:           req.IssuedOn,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toReceiptResponse(receipt))
}

func (rt *Router) listReceipts(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	receipts, err := rt.receipt.List(r.Context(), user, entity.ReceiptFilter{
		Recipient: q.Get("recipient"),
		IssuedGTE: q.Get("issued_from"),
		IssuedLTE: q.Get("issued_to"),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]receiptResponse, 0, len(receipts))
	for _, rc := range receipts {
		out = append(out, toReceiptResponse(rc))
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) getReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	id, err := entity.NewReceiptID(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	receipt, err := rt.receipt.Get(r.Context(), user, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReceiptResponse(receipt))
}

func (rt *Router) reissueReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	id, err := entity.NewReceiptID(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	receipt, err := rt.receipt.Reissue(r.Context(), user, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReceiptResponse(receipt))
}

func (rt *Router) getReceiptPDF(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	id, err := entity.NewReceiptID(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	pdf, err := rt.receipt.GeneratePDF(r.Context(), user, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="receipt-`+id.String()+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

// ── 組織（発行者情報）────────────────────────────────────

type tenantResponse struct {
	ID                 string `json:"id"`
	CorporateName      string `json:"corporate_name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registration_number"`
}

type updateTenantRequest struct {
	CorporateName      string `json:"corporate_name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registration_number"`
}

func (rt *Router) getTenant(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	tenant, err := rt.tenant.Get(r.Context(), user)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTenantResponse(tenant))
}

func (rt *Router) updateTenant(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	var req updateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が不正です")
		return
	}
	tenant, err := rt.tenant.Update(r.Context(), user, usecase.UpdateTenantInput{
		CorporateName:      req.CorporateName,
		Address:            req.Address,
		Phone:              req.Phone,
		Email:              req.Email,
		RegistrationNumber: req.RegistrationNumber,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTenantResponse(tenant))
}

func toTenantResponse(t entity.Tenant) tenantResponse {
	return tenantResponse{
		ID:                 t.ID.String(),
		CorporateName:      t.CorporateName.String(),
		Address:            t.Address.String(),
		Phone:              t.Phone.String(),
		Email:              t.Email.String(),
		RegistrationNumber: t.RegistrationNumber.String(),
	}
}

// ── メンバー（組織の member 管理）────────────────────────

type inviteMemberRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (rt *Router) inviteMember(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が不正です")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "メールアドレスとパスワード(8文字以上)は必須です")
		return
	}
	member, err := rt.member.Invite(r.Context(), user, usecase.InviteMemberInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(member))
}

func (rt *Router) listMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	members, err := rt.member.List(r.Context(), user)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]userResponse, 0, len(members))
	for _, m := range members {
		out = append(out, toUserResponse(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// ── 受け渡し（発行側=認証・カウント非加算）────────────────

type handoverStatusResponse struct {
	ViewID             string `json:"view_id"`
	ViewPassword       string `json:"view_password"`
	URLViewCount       int    `json:"url_view_count"`
	PDFCount           int    `json:"pdf_count"`
	RecipientConfirmed bool   `json:"recipient_confirmed"`
}

func toHandoverStatusResponse(s usecase.HandoverStatus) handoverStatusResponse {
	return handoverStatusResponse{
		ViewID: s.ViewID, ViewPassword: s.ViewPassword,
		URLViewCount: s.URLViewCount, PDFCount: s.PDFCount,
		RecipientConfirmed: s.RecipientConfirmed,
	}
}

func (rt *Router) issueHandover(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	id, err := entity.NewReceiptID(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status, err := rt.handover.Issue(r.Context(), user, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toHandoverStatusResponse(status))
}

func (rt *Router) getHandoverStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.authUser(w, r)
	if !ok {
		return
	}
	id, err := entity.NewReceiptID(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status, err := rt.handover.Status(r.Context(), user, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toHandoverStatusResponse(status))
}

// ── 受け渡し（受領側=非認証・カウント加算）────────────────

type publicHandoverRequest struct {
	Password  string `json:"password"`
	Recipient string `json:"recipient"` // confirm でのみ使う
}

type receiptViewResponse struct {
	Recipient          string `json:"recipient"`
	Amount             int64  `json:"amount"`
	TaxCategory        string `json:"tax_category"`
	IssuedOn           string `json:"issued_on"`
	Proviso            string `json:"proviso"`
	RecipientConfirmed bool   `json:"recipient_confirmed"`
}

func (rt *Router) publicView(w http.ResponseWriter, r *http.Request) {
	viewID, req, ok := rt.publicInput(w, r)
	if !ok {
		return
	}
	view, err := rt.handover.View(r.Context(), viewID, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receiptViewResponse{
		Recipient: view.Recipient, Amount: view.Amount, TaxCategory: view.TaxCategory,
		IssuedOn: view.IssuedOn, Proviso: view.Proviso, RecipientConfirmed: view.RecipientConfirmed,
	})
}

func (rt *Router) publicPDF(w http.ResponseWriter, r *http.Request) {
	viewID, req, ok := rt.publicInput(w, r)
	if !ok {
		return
	}
	pdf, err := rt.handover.OutputPDF(r.Context(), viewID, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="receipt.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (rt *Router) publicConfirm(w http.ResponseWriter, r *http.Request) {
	viewID, req, ok := rt.publicInput(w, r)
	if !ok {
		return
	}
	if err := rt.handover.ConfirmRecipient(r.Context(), viewID, req.Password, req.Recipient); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

// publicInput は受領側リクエストの viewID(パス) と body を取り出す。認証は通さない。
func (rt *Router) publicInput(w http.ResponseWriter, r *http.Request) (entity.ViewID, publicHandoverRequest, bool) {
	viewID, err := entity.NewViewID(chi.URLParam(r, "viewID"))
	if err != nil {
		writeDomainError(w, err)
		return entity.ViewID{}, publicHandoverRequest{}, false
	}
	var req publicHandoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が不正です")
		return entity.ViewID{}, publicHandoverRequest{}, false
	}
	return viewID, req, true
}

// authUser は Bearer トークンから本人を確定する。失敗時は応答を書いて false を返す。
func (rt *Router) authUser(w http.ResponseWriter, r *http.Request) (entity.User, bool) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "Authorization ヘッダがありません")
		return entity.User{}, false
	}
	user, err := rt.auth.CurrentUser(r.Context(), token)
	if err != nil {
		writeDomainError(w, err)
		return entity.User{}, false
	}
	return user, true
}

func toReceiptResponse(rc entity.Receipt) receiptResponse {
	return receiptResponse{
		ID:                 rc.ID.String(),
		Recipient:          rc.Recipient.String(),
		RecipientAddress:   rc.RecipientAddress.String(),
		Amount:             rc.Amount.Int64(),
		TaxCategory:        rc.TaxCategory.String(),
		TaxAmount:          rc.TaxAmount().Int64(),
		NetAmount:          rc.NetAmount().Int64(),
		RegistrationNumber: rc.RegistrationNumber.String(),
		Proviso:            rc.Proviso.String(),
		IssuedOn:           rc.IssuedOn.String(),
	}
}

func toUserResponse(u entity.User) userResponse {
	return userResponse{
		UserID:      u.ID.String(),
		Email:       u.Email.String(),
		DisplayName: u.DisplayName.String(),
		IsActive:    u.IsActive,
		TenantID:    u.TenantID.String(),
		Role:        u.Role.String(),
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

// writeDomainError はドメインのエラーを HTTP ステータスへ翻訳する。
//
// **この変換を行ってよいのは controller だけ。**
// entity や usecase が 401 を知る必要はない。
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entity.ErrAuthFailed),
		errors.Is(err, entity.ErrUserDeactivated),
		errors.Is(err, entity.ErrHandoverAuthFailed):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, entity.ErrUserNotFound),
		errors.Is(err, entity.ErrReceiptNotFound),
		errors.Is(err, entity.ErrTenantNotFound),
		errors.Is(err, entity.ErrHandoverNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, entity.ErrRecipientAlreadyConfirmed):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrReceiptForbidden),
		errors.Is(err, entity.ErrTenantForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, entity.ErrInvalidEmail),
		errors.Is(err, entity.ErrInvalidUserID),
		errors.Is(err, entity.ErrInvalidDisplayName),
		errors.Is(err, entity.ErrInvalidReceiptID),
		errors.Is(err, entity.ErrInvalidAmount),
		errors.Is(err, entity.ErrInvalidRecipient),
		errors.Is(err, entity.ErrInvalidRecipientAddress),
		errors.Is(err, entity.ErrInvalidTaxCategory),
		errors.Is(err, entity.ErrInvalidRegistrationNumber),
		errors.Is(err, entity.ErrInvalidProviso),
		errors.Is(err, entity.ErrInvalidIssuedOn),
		errors.Is(err, entity.ErrInvalidTenantID),
		errors.Is(err, entity.ErrInvalidCorporateName),
		errors.Is(err, entity.ErrInvalidAddress),
		errors.Is(err, entity.ErrInvalidPhone),
		errors.Is(err, entity.ErrInvalidRole),
		errors.Is(err, entity.ErrInvalidViewID),
		errors.Is(err, entity.ErrInvalidViewPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		// 想定外は 500。**レスポンスには詳細を出さないが、ログには必ず残す。**
		// 両方伏せると、本番で原因が追えなくなる。
		slog.Error("想定外のエラー", "error", err)
		writeError(w, http.StatusInternalServerError, "内部エラーが発生しました")
	}
}

func writeError(w http.ResponseWriter, code int, detail string) {
	writeJSON(w, code, map[string]string{"detail": detail})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
