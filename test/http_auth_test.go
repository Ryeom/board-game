package test

import (
	"bytes"
	"encoding/json"
	"github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/util"
	appHttp "github.com/Ryeom/board-game/server/http"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSignUpInvalidEmail 유효하지 않은 이메일로 회원가입 시도 테스트
func TestSignUpInvalidEmail(t *testing.T) {
	// 1. 테스트 서버 시작 및 리소스 정리
	ts, _ := startTestServer(t) // wsURL은 이 테스트에서 사용하지 않으므로 무시
	defer ts.Close()

	// 2. Echo 인스턴스 및 Context 생성
	e := echo.New()
	e.Validator = util.NewValidator() // Echo에 Validator 설정

	// 3. 유효하지 않은 요청 본문 생성 (잘못된 이메일 형식)
	reqBody := map[string]string{
		"email":    "invalid-email", // 유효하지 않은 이메일 형식
		"password": "testpassword123",
		"nickname": "TestUser",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 4. HTTP 요청 및 응답 레코더 설정
	req := httptest.NewRequest(http.MethodPost, "/board-game/auth/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 5. SignUp 핸들러 실행
	err := appHttp.SignUp(c)
	assert.NoError(t, err)

	// 6. 응답 검증
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result response.HttpResult
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, response.ErrorCodeAuthValidation, result.Error.Code)
	assert.Contains(t, result.Message, "입력 값이 유효하지 않")
}

// TestSignUpShortPassword 비밀번호 길이가 너무 짧을 때 회원가입 시도 테스트
func TestSignUpShortPassword(t *testing.T) {
	// 1. 테스트 서버 시작 및 리소스 정리
	ts, _ := startTestServer(t)
	defer ts.Close()

	// 2. Echo 인스턴스 및 Context 생성
	e := echo.New()
	e.Validator = util.NewValidator()

	// 3. 유효하지 않은 요청 본문 생성 (너무 짧은 비밀번호)
	reqBody := map[string]string{
		"email":    "testuser@example.com",
		"password": "short", // 최소 8자 미만
		"nickname": "TestUser",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 4. HTTP 요청 및 응답 레코더 설정
	req := httptest.NewRequest(http.MethodPost, "/board-game/auth/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 5. SignUp 핸들러 실행
	err := appHttp.SignUp(c)
	assert.NoError(t, err)

	// 6. 응답 검증
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result response.HttpResult
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, response.ErrorCodeAuthValidation, result.Error.Code)
	assert.Contains(t, result.Message, "입력 값이 유효하지 않습니다.")
}

// TestSignUpMissingRequiredFields 필수 필드 누락 시 회원가입 시도 테스트
func TestSignUpMissingRequiredFields(t *testing.T) {
	// 1. 테스트 서버 시작 및 리소스 정리
	ts, _ := startTestServer(t)
	defer ts.Close()

	// 2. Echo 인스턴스 및 Context 생성
	e := echo.New()
	e.Validator = util.NewValidator()

	// 3. 유효하지 않은 요청 본문 생성 (email 필드 누락)
	reqBody := map[string]string{
		"password": "testpassword123",
		"nickname": "TestUser",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 4. HTTP 요청 및 응답 레코더 설정
	req := httptest.NewRequest(http.MethodPost, "/board-game/auth/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 5. SignUp 핸들러 실행
	err := appHttp.SignUp(c)
	assert.NoError(t, err)

	// 6. 응답 검증
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result response.HttpResult
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, response.ErrorCodeAuthValidation, result.Error.Code)
	assert.Contains(t, result.Message, "입력 값이 유효하지 않습니다.")
}
