package auth

import (
	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/labstack/echo/v4"
	"net/http"
)

func SignUp(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	u := user.User{
		Email:    req.Email,
		Password: hashed,
		Nickname: req.Nickname,
	}
	if err := db.DB.Create(&u).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "회원가입 실패"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "회원가입 성공"})
}

func Login(c echo.Context) error {
	var payload LoginRequest
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, "invalid payload")
	}
	if err := c.Validate(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, "validation error")
	}

	u, err := user.FindUserByEmail(payload.Email)
	if err != nil || !user.CheckPassword(payload.Password, u.Password) {
		return c.JSON(http.StatusUnauthorized, "invalid credentials")
	}
	token, err := GenerateJWT(u.ID.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "token generation failed")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token": token,
		"user":  u.Nickname,
	})
}
