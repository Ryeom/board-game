package http

import (
	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/internal/auth"
	apperr "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/log"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

type SignupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Nickname string `json:"nickname" validate:"required,min=2,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// SignUp은 새로운 사용자를 등록합니다.
// @Summary 회원가입
// @Description 새로운 사용자 계정을 생성합니다.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body SignupRequest true "회원가입 요청"
// @Success 200 {object} HttpResult "회원가입 성공"
// @Failure 400 {object} HttpResult "잘못된 요청 형식 또는 유효성 검사 실패"
// @Failure 409 {object} HttpResult "이미 존재하는 이메일"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/auth/signup [post]
func SignUp(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		log.Logger.Errorf("SignUp - Bind Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthBind, err)
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("SignUp - Validation Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthValidation, err)
	}

	// 이메일 중복 확인
	existingUser, err := user.FindUserByEmail(req.Email)
	if err != nil {
		log.Logger.Errorf("SignUp - FindUserByEmail Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthUserLookupFailed, err)
	}
	if existingUser != nil {
		return apperr.Conflict(apperr.ErrorCodeAuthEmailDuplicate, nil)
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		log.Logger.Errorf("SignUp - Password Hashing Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthPasswordHashingFailed, err)
	}

	u := user.User{
		Email:    req.Email,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Role:     user.RoleUser,
		IsActive: true,
	}

	if err := db.DB.Create(&u).Error; err != nil {
		log.Logger.Errorf("SignUp - DB Create User Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthCreateUserFailed, err)
	}

	return c.JSON(http.StatusOK, Success(map[string]string{"message": "회원가입 성공"}, "회원가입이 성공적으로 완료되었습니다."))
}

// Login은 사용자를 인증하고 JWT를 반환합니다.
// @Summary 로그인
// @Description 사용자를 인증하고 JWT(JSON Web Token)를 발급합니다.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "로그인 요청"
// @Success 200 {object} HttpResult{data=object{token=string,user_id=string,nickname=string}} "로그인 성공 및 JWT 토큰 반환"
// @Failure 400 {object} HttpResult "잘못된 요청 형식 또는 유효성 검사 실패"
// @Failure 401 {object} HttpResult "인증 실패 (잘못된 이메일 또는 비밀번호)"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/auth/login [post]
func Login(c echo.Context) error {
	var payload LoginRequest
	if err := c.Bind(&payload); err != nil {
		log.Logger.Errorf("Login - Bind Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthBind, err)
	}
	if err := c.Validate(&payload); err != nil {
		log.Logger.Errorf("Login - Validation Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthValidation, err)
	}

	u, err := user.FindUserByEmail(payload.Email)
	if err != nil {
		log.Logger.Errorf("Login - FindUserByEmail Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthUserLookupFailed, err)
	}
	if u == nil || !util.CheckPasswordHash(payload.Password, u.Password) {
		return apperr.Unauthorized(apperr.ErrorCodeAuthInvalidCredentials, err)
	}

	if err := user.UpdateLastLoginAt(db.DB, u.ID.String()); err != nil {
		log.Logger.Errorf("Login - UpdateLastLoginAt Error: %v", err)
		//return apperr.InternalServerError(apperr.ErrorCodeDbUpdateFailed, err)
	}

	token, err := auth.GenerateJWT(u.ID.String())
	if err != nil {
		log.Logger.Errorf("Login - JWT Generation Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthJwtGenerationFailed, err)
	}

	return c.JSON(http.StatusOK, Success(map[string]any{
		"token":    token,
		"user_id":  u.ID.String(),
		"nickname": u.Nickname,
	}, "로그인이 성공적으로 완료되었습니다."))
}

// @Summary 로그아웃
// @Description 현재 사용 중인 JWT 토큰을 무효화하여 즉시 로그아웃 처리합니다.
// @Tags Auth
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} HttpResult "로그아웃 성공"
// @Failure 400 {object} HttpResult "잘못된 요청 형식 (토큰 없음)"
// @Failure 401 {object} HttpResult "인증되지 않은 토큰 또는 처리 실패"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/api/auth/logout [post]
func Logout(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		log.Logger.Error("Logout - Authorization header missing or malformed")
		return apperr.BadRequest(apperr.ErrorCodeAuthInvalidToken, nil)
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	// 토큰을 블랙리스트에 추가
	if err := auth.AddTokenToBlacklist(c.Request().Context(), tokenStr); err != nil {
		log.Logger.Errorf("Logout - Failed to add token to blacklist: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthLogoutFailed, err)
	}

	return c.JSON(http.StatusOK, Success(nil, "성공적으로 로그아웃되었습니다."))
}
