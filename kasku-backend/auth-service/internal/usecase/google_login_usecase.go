package usecase

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	googleTokenInfoURL    = "https://oauth2.googleapis.com/tokeninfo"
	googleTokenURL        = "https://oauth2.googleapis.com/token"
	oauthPasswordSentinel = "OAUTH_NO_PASSWORD" // tidak valid sebagai Argon2id hash
)

// GoogleLoginInput merupakan data yang dikirim saat login via Google ID token (popup flow).
type GoogleLoginInput struct {
	IDToken   string
	UserAgent string
	IPAddress string
	IsDev     bool
}

// GoogleCodeInput merupakan data untuk authorization code flow (redirect flow).
type GoogleCodeInput struct {
	Code        string
	RedirectURI string
	UserAgent   string
	IPAddress   string
	IsDev       bool
}

// googleTokenResponse adalah response dari Google token endpoint.
type googleTokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// GoogleTokenInfo adalah subset dari response Google tokeninfo endpoint.
type GoogleTokenInfo struct {
	Sub           string `json:"sub"` // Google user ID (unik per Google account)
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"` // "true" / "false"
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	Aud           string `json:"aud"` // client ID audience
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

// GoogleLoginUseCase mendefinisikan kontrak untuk alur login via Google OAuth.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_google_login_usecase.go -package=mocks
type GoogleLoginUseCase interface {
	Execute(ctx context.Context, input GoogleLoginInput) (*LoginOutput, error)
	ExchangeCode(ctx context.Context, input GoogleCodeInput) (*LoginOutput, error)
}

type googleLoginUseCase struct {
	pool               *pgxpool.Pool
	userRepo           repository.UserRepository
	refreshTokenRepo   repository.RefreshTokenRepository
	publisher          messaging.EventPublisher
	jwtPrivateKey      *rsa.PrivateKey
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
	googleClientID     string // kosong = skip audience check (development mode)
	googleClientSecret string // wajib untuk authorization code flow
	httpClient         *http.Client
	billingQuerier     SubscriptionQuerier
}

// NewGoogleLoginUseCase membuat instance GoogleLoginUseCase.
func NewGoogleLoginUseCase(
	pool *pgxpool.Pool,
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	publisher messaging.EventPublisher,
	jwtPrivateKey *rsa.PrivateKey,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	googleClientID string,
	googleClientSecret string,
	billingQuerier SubscriptionQuerier,
) GoogleLoginUseCase {
	return &googleLoginUseCase{
		pool:               pool,
		userRepo:           userRepo,
		refreshTokenRepo:   refreshTokenRepo,
		publisher:          publisher,
		jwtPrivateKey:      jwtPrivateKey,
		accessTokenTTL:     accessTokenTTL,
		refreshTokenTTL:    refreshTokenTTL,
		googleClientID:     googleClientID,
		googleClientSecret: googleClientSecret,
		httpClient:         &http.Client{Timeout: 10 * time.Second},
		billingQuerier:     billingQuerier,
	}
}

// ExchangeCode menukar authorization code (redirect flow) menjadi LoginOutput.
// Code ditukar ke Google token endpoint menggunakan client secret (server-side),
// lalu id_token yang didapat diproses lewat alur Execute yang sudah ada.
func (uc *googleLoginUseCase) ExchangeCode(ctx context.Context, input GoogleCodeInput) (*LoginOutput, error) {
	if uc.googleClientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_SECRET tidak dikonfigurasi di server")
	}

	idToken, err := uc.exchangeAuthCode(input.Code, input.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrGoogleAuth, err)
	}

	return uc.Execute(ctx, GoogleLoginInput{
		IDToken:   idToken,
		UserAgent: input.UserAgent,
		IPAddress: input.IPAddress,
		IsDev:     input.IsDev,
	})
}

// exchangeAuthCode menukar OAuth2 authorization code ke id_token via Google token endpoint.
func (uc *googleLoginUseCase) exchangeAuthCode(code, redirectURI string) (string, error) {
	resp, err := uc.httpClient.PostForm(googleTokenURL, url.Values{
		"code":          {code},
		"client_id":     {uc.googleClientID},
		"client_secret": {uc.googleClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", fmt.Errorf("gagal hubungi Google token endpoint: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp googleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("gagal decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("google token error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.IDToken == "" {
		return "", fmt.Errorf("id_token tidak ada dalam response Google token endpoint")
	}

	return tokenResp.IDToken, nil
}

// Execute menjalankan alur login / auto-register via Google ID token:
//  1. Verifikasi ID token ke Google tokeninfo endpoint
//  2. Cari user berdasarkan google_id → jika ada, generate JWT
//  3. Jika tidak ada → cari berdasarkan email
//     a. Jika email sudah terdaftar → link google_id, generate JWT
//     b. Jika email belum terdaftar → auto-register user baru, generate JWT
func (uc *googleLoginUseCase) Execute(ctx context.Context, input GoogleLoginInput) (*LoginOutput, error) {
	tokenInfo, err := uc.verifyGoogleToken(input.IDToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrGoogleAuth, err)
	}

	if tokenInfo.EmailVerified != "true" {
		return nil, fmt.Errorf("%w: email Google belum diverifikasi", domainerrors.ErrGoogleAuth)
	}

	// Cari user berdasarkan google_id
	user, err := uc.userRepo.FindByGoogleID(ctx, tokenInfo.Sub)
	if err != nil {
		return nil, fmt.Errorf("gagal lookup user by google_id: %w", err)
	}

	if user == nil {
		// Cari berdasarkan email (akun lokal yang belum di-link)
		user, err = uc.userRepo.FindByEmail(ctx, tokenInfo.Email)
		if err != nil {
			return nil, fmt.Errorf("gagal lookup user by email: %w", err)
		}

		if user != nil {
			// Link google_id ke akun yang sudah ada
			if err := uc.userRepo.SaveGoogleID(ctx, user.ID, tokenInfo.Sub); err != nil {
				return nil, fmt.Errorf("gagal link google_id: %w", err)
			}
			user.GoogleID = &tokenInfo.Sub
			// Aktifkan akun jika belum aktif (mungkin daftar tapi belum verifikasi email)
			if !user.IsActive {
				if err := uc.userRepo.VerifyEmail(ctx, user.ID); err != nil {
					return nil, fmt.Errorf("gagal aktifkan akun via google: %w", err)
				}
				user.IsActive = true
				user.EmailVerified = true
			}
		} else {
			// Auto-register user baru
			user, err = uc.autoRegister(ctx, tokenInfo)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := uc.userRepo.UpdateLoginSuccess(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("gagal update login success: %w", err)
	}

	return uc.generateTokenPair(ctx, user, input)
}

// verifyGoogleToken memverifikasi ID token ke Google tokeninfo endpoint.
func (uc *googleLoginUseCase) verifyGoogleToken(idToken string) (*GoogleTokenInfo, error) {
	resp, err := uc.httpClient.Get(googleTokenInfoURL + "?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("gagal hubungi Google tokeninfo: %w", err)
	}
	defer resp.Body.Close()

	var info GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("gagal decode tokeninfo response: %w", err)
	}

	if info.Error != "" {
		return nil, fmt.Errorf("google token tidak valid: %s", info.ErrorDesc)
	}

	if info.Sub == "" || info.Email == "" {
		return nil, fmt.Errorf("tokeninfo tidak lengkap: sub atau email kosong")
	}

	// Verifikasi audience jika Client ID dikonfigurasi
	if uc.googleClientID != "" && info.Aud != uc.googleClientID {
		return nil, fmt.Errorf("token audience tidak cocok dengan client ID yang dikonfigurasi")
	}

	return &info, nil
}

// autoRegister membuat user baru dari data Google.
func (uc *googleLoginUseCase) autoRegister(ctx context.Context, info *GoogleTokenInfo) (*entity.User, error) {
	username := deriveUsername(info)

	// Pastikan username unik
	username, err := uc.ensureUniqueUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	googleID := info.Sub
	user := &entity.User{
		ID:               uuid.New(),
		Email:            info.Email,
		Username:         username,
		PasswordHash:     oauthPasswordSentinel, // tidak bisa dipakai untuk login biasa
		IsActive:         true,                  // Google sudah verifikasi email
		EmailVerified:    true,
		FailedLoginCount: 0,
		GoogleID:         &googleID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mulai transaksi registrasi OAuth: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO public.users
			(id, email, username, password_hash, is_active, email_verified,
			 failed_login_count, google_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		user.ID, user.Email, user.Username, user.PasswordHash, user.IsActive, user.EmailVerified,
		user.FailedLoginCount, user.GoogleID, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal insert oauth user: %w", err)
	}

	event := messaging.UserRegisteredEvent{
		UserID:            user.ID.String(),
		Email:             user.Email,
		Username:          user.Username,
		VerificationToken: "", // OAuth user — tidak perlu verifikasi email
	}
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal outbox event: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5)
	`,
		uuid.New(),
		"user.registered",
		messaging.RoutingKeyUserRegistered,
		string(eventPayload),
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi registrasi OAuth: %w", err)
	}

	return user, nil
}

// generateTokenPair membuat access token + refresh token untuk user.
func (uc *googleLoginUseCase) generateTokenPair(ctx context.Context, user *entity.User, input GoogleLoginInput) (*LoginOutput, error) {
	// Pakai helper yang sama dengan loginUseCase
	luc := &loginUseCase{
		userRepo:         uc.userRepo,
		refreshTokenRepo: uc.refreshTokenRepo,
		jwtPrivateKey:    uc.jwtPrivateKey,
		accessTokenTTL:   uc.accessTokenTTL,
		refreshTokenTTL:  uc.refreshTokenTTL,
	}

	tier := uc.billingQuerier.GetTierName(ctx, user.ID.String())
	accessToken, err := luc.generateAccessToken(user, tier)
	if err != nil {
		return nil, fmt.Errorf("gagal generate access token: %w", err)
	}

	rawToken, tokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("gagal generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	userAgent := &input.UserAgent
	ipAddress := &input.IPAddress
	if input.UserAgent == "" {
		userAgent = nil
	}
	if input.IPAddress == "" {
		ipAddress = nil
	}

	refreshToken := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		CreatedAt: now,
	}
	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("gagal simpan refresh token: %w", err)
	}

	return &LoginOutput{
		AccessToken: accessToken,
		TokenType:   tokenTypeBearear,
		ExpiresIn:   int64(uc.accessTokenTTL.Seconds()),
		RefreshTokenCookie: RefreshTokenCookieParams{
			RawToken: rawToken,
			MaxAge:   int(uc.refreshTokenTTL.Seconds()),
			IsSecure: !input.IsDev,
		},
	}, nil
}

var (
	allowedUsernameChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// deriveUsername membuat username dari data Google (name atau email prefix).
func deriveUsername(info *GoogleTokenInfo) string {
	base := info.GivenName
	if base == "" {
		base = info.Name
	}
	if base == "" {
		base = strings.Split(info.Email, "@")[0]
	}

	// Bersihkan karakter tidak valid dan pertahankan huruf/angka/underscore saja
	cleaned := allowedUsernameChars.ReplaceAllString(base, "_")
	cleaned = strings.Trim(cleaned, "_")

	// Pastikan panjang valid (3-30 chars)
	runes := []rune(cleaned)
	if len(runes) < 3 {
		cleaned = cleaned + "_g"
	}
	if len(runes) > 28 {
		cleaned = string(runes[:28])
	}

	// Pastikan dimulai dengan huruf atau angka
	if len(cleaned) > 0 {
		first := rune(cleaned[0])
		if !unicode.IsLetter(first) && !unicode.IsDigit(first) {
			cleaned = "u" + cleaned
		}
	}

	return cleaned
}

// ensureUniqueUsername menambahkan suffix angka jika username sudah dipakai.
func (uc *googleLoginUseCase) ensureUniqueUsername(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 1; i <= 99; i++ {
		exists, err := uc.userRepo.ExistsByUsername(ctx, candidate)
		if err != nil {
			return "", fmt.Errorf("gagal cek keberadaan username: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
		if len(candidate) > 30 {
			// Potong base agar muat
			maxBase := 30 - len(fmt.Sprintf("_%d", i))
			candidate = fmt.Sprintf("%s_%d", base[:maxBase], i)
		}
	}
	return "", domainerrors.ErrUsernameAlreadyExists
}
