package http

import (
	"github.com/Ryeom/board-game/infra/db"
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
	// JWTMiddleware에서 설정한 userID 가져오기
	// c.Get("userID")는 interface{} 타입을 반환하므로 타입 어설션 필요
	userID, ok := c.Get("userID").(string)
	if !ok || userID == "" {
		log.Logger.Error("GetUserProfile - UserID not found in context")
		return c.JSON(http.StatusUnauthorized, Failure("인증되지 않은 사용자입니다.", http.StatusUnauthorized))
	}

	u, err := user.FindUserByID(userID) //
	if err != nil {
		log.Logger.Errorf("GetUserProfile - FindUserByID Error for ID %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, Failure("사용자 정보를 가져오는 데 실패했습니다.", http.StatusInternalServerError))
	}
	if u == nil { //
		return c.JSON(http.StatusNotFound, Failure("사용자를 찾을 수 없습니다.", http.StatusNotFound))
	}

	// 비밀번호와 같은 민감 정보는 제외하고 반환
	responseData := map[string]interface{}{
		"userId":       u.ID.String(),
		"email":        u.Email,
		"nickname":     u.Nickname,
		"profileImage": u.ProfileImage, // nil일 수 있음
		"role":         u.Role,
		"isActive":     u.IsActive,
		"lastLoginAt":  u.LastLoginAt, // nil일 수 있음
		"createdAt":    u.CreatedAt,
		"updatedAt":    u.UpdatedAt,
	}

	return c.JSON(http.StatusOK, Success(responseData, "사용자 프로필 조회 성공"))
}

// UpdateProfileRequest는 사용자 프로필 업데이트 요청의 구조를 정의합니다.
type UpdateProfileRequest struct {
	Nickname     *string `json:"nickname" validate:"omitempty,min=2,max=20"` // 닉네임 (선택적, 최소 2자, 최대 20자)
	ProfileImage *string `json:"profileImage" validate:"omitempty,url"`      // 프로필 이미지 URL (선택적, URL 형식 검사)
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
		return c.JSON(http.StatusUnauthorized, Failure("인증되지 않은 사용자입니다.", http.StatusUnauthorized))
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		log.Logger.Errorf("UpdateUserProfile - Bind Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure("잘못된 요청 형식입니다", http.StatusBadRequest))
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("UpdateUserProfile - Validation Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure(err.Error(), http.StatusBadRequest))
	}

	u, err := user.FindUserByID(userID) //
	if err != nil {
		log.Logger.Errorf("UpdateUserProfile - FindUserByID Error for ID %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, Failure("사용자 정보를 가져오는 데 실패했습니다.", http.StatusInternalServerError))
	}
	if u == nil {
		return c.JSON(http.StatusNotFound, Failure("사용자를 찾을 수 없습니다.", http.StatusNotFound))
	}

	// 닉네임 업데이트
	if req.Nickname != nil && *req.Nickname != u.Nickname {
		u.Nickname = *req.Nickname
	}
	// 프로필 이미지 업데이트
	if req.ProfileImage != nil {
		if *req.ProfileImage == "" { // 클라이언트가 이미지 삭제를 요청한 경우
			u.ProfileImage = nil
		} else if u.ProfileImage == nil || *req.ProfileImage != *u.ProfileImage {
			u.ProfileImage = req.ProfileImage
		}
	}

	// 변경 사항 저장
	if err := db.DB.Save(u).Error; err != nil { //
		log.Logger.Errorf("UpdateUserProfile - DB Save Error for ID %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, Failure("프로필 업데이트에 실패했습니다.", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, Success(nil, "프로필이 성공적으로 업데이트되었습니다."))
}

// ChangePasswordRequest는 비밀번호 변경 요청의 구조를 정의합니다.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"` // 새 비밀번호 최소 길이 8자
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
		return c.JSON(http.StatusUnauthorized, Failure("인증되지 않은 사용자입니다.", http.StatusUnauthorized))
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		log.Logger.Errorf("ChangePassword - Bind Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure("잘못된 요청 형식입니다", http.StatusBadRequest))
	}
	if err := c.Validate(&req); err != nil {
		log.Logger.Errorf("ChangePassword - Validation Error: %v", err)
		return c.JSON(http.StatusBadRequest, Failure(err.Error(), http.StatusBadRequest))
	}

	u, err := user.FindUserByID(userID) //
	if err != nil {
		log.Logger.Errorf("ChangePassword - FindUserByID Error for ID %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, Failure("사용자 정보를 가져오는 데 실패했습니다.", http.StatusInternalServerError))
	}
	if u == nil {
		return c.JSON(http.StatusNotFound, Failure("사용자를 찾을 수 없습니다.", http.StatusNotFound))
	}

	// 현재 비밀번호 검증
	if !util.CheckPasswordHash(req.CurrentPassword, u.Password) { //
		return c.JSON(http.StatusUnauthorized, Failure("현재 비밀번호가 올바르지 않습니다.", http.StatusUnauthorized))
	}

	// 새 비밀번호 해싱
	hashedNewPassword, err := util.HashPassword(req.NewPassword) //
	if err != nil {
		log.Logger.Errorf("ChangePassword - Password Hashing Error: %v", err)
		return c.JSON(http.StatusInternalServerError, Failure("비밀번호 처리 중 오류가 발생했습니다.", http.StatusInternalServerError))
	}

	// DB에 새 비밀번호 저장
	u.Password = hashedNewPassword
	if err := db.DB.Save(u).Error; err != nil { //
		log.Logger.Errorf("ChangePassword - DB Save Error for ID %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, Failure("비밀번호 변경에 실패했습니다.", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, Success(nil, "비밀번호가 성공적으로 변경되었습니다."))
}
