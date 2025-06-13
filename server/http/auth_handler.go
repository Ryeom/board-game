package http

import (
	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/internal/auth"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/log"
	"github.com/labstack/echo/v4"
	"net/http"
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
		return c.JSON(http.StatusBadRequest, Failure("잘못된 요청 형식입니다", http.StatusBadRequest))
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("SignUp - Validation Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure(err.Error(), http.StatusBadRequest))
	}

	// 이메일 중복 확인
	existingUser, err := user.FindUserByEmail(req.Email)
	if err != nil {
		log.Logger.Errorf("SignUp - FindUserByEmail Error: %v", err)
		return c.JSON(http.StatusInternalServerError, Failure("회원가입 처리 중 오류가 발생했습니다", http.StatusInternalServerError))
	}
	if existingUser != nil {
		return c.JSON(http.StatusConflict, Failure("이미 존재하는 이메일입니다", http.StatusConflict))
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		log.Logger.Errorf("SignUp - Password Hashing Error: %v", err)
		return c.JSON(http.StatusInternalServerError, Failure("비밀번호 처리 중 오류가 발생했습니다", http.StatusInternalServerError))
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
		return c.JSON(http.StatusInternalServerError, Failure("회원가입에 실패했습니다", http.StatusInternalServerError))
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
		return c.JSON(http.StatusBadRequest, Failure("잘못된 요청 형식입니다", http.StatusBadRequest))
	}
	if err := c.Validate(&payload); err != nil {
		log.Logger.Errorf("Login - Validation Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure(err.Error(), http.StatusBadRequest))
	}

	u, err := user.FindUserByEmail(payload.Email)
	if err != nil {
		log.Logger.Errorf("Login - FindUserByEmail Error: %v", err)
		return c.JSON(http.StatusInternalServerError, Failure("로그인 처리 중 오류가 발생했습니다", http.StatusInternalServerError))
	}
	if u == nil {
		return c.JSON(http.StatusUnauthorized, Failure("이메일 또는 비밀번호가 올바르지 않습니다", http.StatusUnauthorized))
	}

	if !util.CheckPasswordHash(payload.Password, u.Password) {
		return c.JSON(http.StatusUnauthorized, Failure("이메일 또는 비밀번호가 올바르지 않습니다", http.StatusUnauthorized))
	}

	if err := user.UpdateLastLoginAt(db.DB, u.ID.String()); err != nil {
		log.Logger.Errorf("Login - UpdateLastLoginAt Error: %v", err)
	}

	token, err := auth.GenerateJWT(u.ID.String()) // auth 패키지의 GenerateJWT 사용
	if err != nil {
		log.Logger.Errorf("Login - JWT Generation Error: %v", err)
		return c.JSON(http.StatusInternalServerError, Failure("인증 토큰 생성에 실패했습니다", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, Success(map[string]any{
		"token":    token,
		"user_id":  u.ID.String(),
		"nickname": u.Nickname,
	}, "로그인이 성공적으로 완료되었습니다."))
}
