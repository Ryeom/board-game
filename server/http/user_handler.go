package http

import (
	"github.com/Ryeom/board-game/infra/db"
	apperr "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/log"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetUserProfile - 사용자 프로필 조회
// @Summary 사용자 프로필 조회
// @Description 로그인한 사용자의 프로필 정보를 조회합니다.
// @Tags User
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} HttpResult{data=object{userId=string,email=string,nickname=string,profileImage=string,role=string,isActive=bool,lastLoginAt=string}} "사용자 프로필 조회 성공"
// @Failure 401 {object} HttpResult "인증되지 않은 사용자"
// @Failure 404 {object} HttpResult "사용자를 찾을 수 없음"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/api/user/profile [get]
func GetUserProfile(c echo.Context) error {
	userID, ok := c.Get("userID").(string)
	if !ok || userID == "" {
		log.Logger.Error("GetUserProfile - UserID not found in context")
		return apperr.Unauthorized(apperr.ErrorCodeUserUnauthorized, nil)
	}

	u, err := user.FindUserByID(userID)
	if err != nil {
		log.Logger.Errorf("GetUserProfile - FindUserByID Error for ID %s: %v", userID, err)
		return apperr.InternalServerError(apperr.ErrorCodeUserProfileFetchFailed, err)
	}
	if u == nil {
		return apperr.NotFound(apperr.ErrorCodeUserNotFound, nil)
	}

	responseData := map[string]interface{}{
		"userId":       u.ID.String(),
		"email":        u.Email,
		"nickname":     u.Nickname,
		"profileImage": u.ProfileImage,
		"role":         u.Role,
		"isActive":     u.IsActive,
		"lastLoginAt":  u.LastLoginAt,
		"createdAt":    u.CreatedAt,
		"updatedAt":    u.UpdatedAt,
	}

	return c.JSON(http.StatusOK, Success(responseData, "사용자 프로필 조회 성공"))
}

type UpdateProfileRequest struct {
	Nickname     *string `json:"nickname" validate:"omitempty,min=2,max=20"`
	ProfileImage *string `json:"profileImage" validate:"omitempty,url"`
}

// UpdateUserProfile - 사용자 프로필 업데이트
// @Summary 사용자 프로필 업데이트
// @Description 로그인한 사용자의 프로필 정보를 업데이트합니다 (닉네임, 프로필 이미지).
// @Tags User
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "업데이트할 프로필 정보"
// @Success 200 {object} HttpResult "프로필 업데이트 성공"
// @Failure 400 {object} HttpResult "잘못된 요청 형식 또는 유효성 검사 실패"
// @Failure 401 {object} HttpResult "인증되지 않은 사용자"
// @Failure 404 {object} HttpResult "사용자를 찾을 수 없음"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/api/user/profile [patch]
func UpdateUserProfile(c echo.Context) error {
	userID, ok := c.Get("userID").(string)
	if !ok || userID == "" {
		log.Logger.Error("UpdateUserProfile - UserID not found in context")
		return apperr.Unauthorized(apperr.ErrorCodeUserUnauthorized, nil)
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		log.Logger.Errorf("UpdateUserProfile - Bind Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeRoomInvalidRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("UpdateUserProfile - Validation Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthValidation, err)
	}

	u, err := user.FindUserByID(userID)
	if err != nil {
		log.Logger.Errorf("UpdateUserProfile - FindUserByID Error for ID %s: %v", userID, err)
		return apperr.InternalServerError(apperr.ErrorCodeUserProfileFetchFailed, err)
	}
	if u == nil {
		return apperr.NotFound(apperr.ErrorCodeUserNotFound, nil)
	}

	if req.Nickname != nil && *req.Nickname != u.Nickname {
		u.Nickname = *req.Nickname
	}
	if req.ProfileImage != nil {
		if *req.ProfileImage == "" {
			u.ProfileImage = nil
		} else if u.ProfileImage == nil || *req.ProfileImage != *u.ProfileImage {
			u.ProfileImage = req.ProfileImage
		}
	}

	if err := db.DB.Save(u).Error; err != nil {
		log.Logger.Errorf("UpdateUserProfile - DB Save Error for ID %s: %v", userID, err)
		return apperr.InternalServerError(apperr.ErrorCodeUserProfileUpdateFailed, err)
	}

	return c.JSON(http.StatusOK, Success(nil, "프로필이 성공적으로 업데이트되었습니다."))
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// ChangePassword - 사용자 비밀번호 변경
// @Summary 사용자 비밀번호 변경
// @Description 로그인한 사용자의 비밀번호를 변경합니다.
// @Tags User
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "비밀번호 변경 요청"
// @Success 200 {object} HttpResult "비밀번호 변경 성공"
// @Failure 400 {object} HttpResult "잘못된 요청 형식 또는 유효성 검사 실패"
// @Failure 401 {object} HttpResult "인증되지 않은 사용자 또는 현재 비밀번호 불일치"
// @Failure 404 {object} HttpResult "사용자를 찾을 수 없음"
// @Failure 500 {object} HttpResult "서버 오류"
// @Router /board-game/api/user/change-password [post]
func ChangePassword(c echo.Context) error {
	userID, ok := c.Get("userID").(string)
	if !ok || userID == "" {
		log.Logger.Error("ChangePassword - UserID not found in context")
		return apperr.Unauthorized(apperr.ErrorCodeUserUnauthorized, nil)
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		log.Logger.Errorf("ChangePassword - Bind Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeRoomInvalidRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("ChangePassword - Validation Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeAuthValidation, err)
	}

	u, err := user.FindUserByID(userID)
	if err != nil {
		log.Logger.Errorf("ChangePassword - FindUserByID Error for ID %s: %v", userID, err)
		return apperr.InternalServerError(apperr.ErrorCodeUserProfileFetchFailed, err)
	}
	if u == nil {
		return apperr.NotFound(apperr.ErrorCodeUserNotFound, err)
	}

	if !util.CheckPasswordHash(req.CurrentPassword, u.Password) {
		return apperr.Unauthorized(apperr.ErrorCodeUserCurrentPasswordMismatch, nil)
	}

	hashedNewPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		log.Logger.Errorf("ChangePassword - Password Hashing Error: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeAuthPasswordHashingFailed, err)
	}

	u.Password = hashedNewPassword
	if err := db.DB.Save(u).Error; err != nil {
		log.Logger.Errorf("ChangePassword - DB Save Error for ID %s: %v", userID, err)
		return apperr.InternalServerError(apperr.ErrorCodeUserPasswordChangeFailed, err)
	}

	return c.JSON(http.StatusOK, Success(nil, "비밀번호가 성공적으로 변경되었습니다."))
}
