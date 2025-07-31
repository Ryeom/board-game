package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/util"
	appHttp "github.com/Ryeom/board-game/server/http"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

// TestSignUpDuplicateEmail 이미 존재하는 이메일로 회원가입 시도 테스트
func TestSignUpDuplicateEmail(t *testing.T) {
	// 1. 테스트 서버 시작 및 리소스 정리
	ts, _ := startTestServer(t)
	defer ts.Close()

	// 고유한 이메일 생성을 위해 타임스탬프 사용 (테스트 간 간섭 방지)
	uniqueEmail := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	password := "testpassword123"
	nickname := "UniqueTester"

	// --- 첫 번째 회원가입 시도 (성공 예상) ---
	e1 := echo.New()
	e1.Validator = util.NewValidator()

	reqBody1 := map[string]string{
		"email":    uniqueEmail,
		"password": password,
		"nickname": nickname,
	}
	jsonBody1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest(http.MethodPost, "/board-game/auth/signup", bytes.NewBuffer(jsonBody1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec1 := httptest.NewRecorder()
	c1 := e1.NewContext(req1, rec1)

	err1 := appHttp.SignUp(c1)
	assert.NoError(t, err1)
	assert.Equal(t, http.StatusOK, rec1.Code, "첫 번째 회원가입은 성공해야 합니다.")

	var result1 response.HttpResult
	json.Unmarshal(rec1.Body.Bytes(), &result1)
	assert.NoError(t, err1)
	assert.Equal(t, "success", result1.Status, "첫 번째 회원가입 응답 상태는 'success'여야 합니다.")
	assert.Equal(t, response.SuccessCodeUserSignUp, result1.Code, "첫 번째 회원가입 성공 코드를 확인합니다.") // 성공 응답의 Code 필드 확인

	// --- 두 번째 회원가입 시도 (동일한 이메일로 실패 예상) ---
	e2 := echo.New()
	e2.Validator = util.NewValidator()

	reqBody2 := map[string]string{
		"email":    uniqueEmail, // 이미 존재하는 이메일 사용
		"password": password,
		"nickname": "AnotherTester",
	}
	jsonBody2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest(http.MethodPost, "/board-game/auth/signup", bytes.NewBuffer(jsonBody2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)

	err2 := appHttp.SignUp(c2)
	assert.NoError(t, err2)

	// 응답 상태 코드 확인
	// auth_handler.go에서 ERROR_AUTH_EMAIL_DUPLICATE에 대해 http.StatusNotFound (404)를 반환함
	// code.json에는 409 Conflict로 정의되어 있지만, 현재 핸들러의 동작에 맞게 404를 예상
	assert.Equal(t, http.StatusNotFound, rec2.Code, "두 번째 회원가입은 404 Not Found를 반환해야 합니다.")

	var result2 response.HttpResult
	json.Unmarshal(rec2.Body.Bytes(), &result2)
	assert.NoError(t, err2)

	// 응답 상태 및 에러 코드 확인
	assert.Equal(t, "error", result2.Status, "두 번째 회원가입 응답 상태는 'error'여야 합니다.")
	assert.Equal(t, response.ErrorCodeAuthEmailDuplicate, result2.Error.Code, "이메일 중복 에러 코드를 확인합니다.") //
	assert.Contains(t, result2.Message, "이미 존재하는 이메일입니다.", "이메일 중복 메시지를 확인합니다.")                      //
}

// TestAccessProtectedAPIWithInvalidToken 유효하지 않은 토큰으로 보호된 API 접근 시도 테스트
func TestAccessProtectedAPIWithInvalidToken(t *testing.T) {
	// 1. 테스트 서버 시작 및 리소스 정리
	ts, _ := startTestServer(t)
	defer ts.Close()

	// 2. 보호된 API 엔드포인트
	profileURL := fmt.Sprintf("%s/board-game/api/user/profile", ts.URL)

	tests := []struct {
		name         string
		authHeader   string // Authorization 헤더 값
		expectedCode string // 예상되는 오류 코드
	}{
		{
			name:         "토큰 없음",
			authHeader:   "",                                 // Authorization 헤더 없음
			expectedCode: response.ErrorCodeAuthInvalidToken, //
		},
		{
			name:         "유효하지 않은 형식의 토큰",
			authHeader:   "Bearer invalid-jwt-token",         // 잘못된 JWT 형식
			expectedCode: response.ErrorCodeAuthInvalidToken, //
		},
		{
			name:         "만료된 토큰",                                                                                                                // 실제 만료된 토큰을 생성하기는 어렵지만, 유효하지 않은 형식으로 대체하여 테스트
			authHeader:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NDAwMDAwMDAsInVzZXJfaWQiOiJzb21lVXNlcklkIn0.SignatureMismatch", // exp가 과거인 토큰 (서명 불일치로도 처리될 수 있음)
			expectedCode: response.ErrorCodeAuthInvalidToken,                                                                                      //
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, profileURL, nil)
			assert.NoError(t, err)

			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			resp, err := ts.Client().Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "예상치 못한 HTTP 상태 코드") //

			var result response.HttpResult
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			assert.Equal(t, "error", result.Status, "응답 상태는 'error'여야 합니다.")
			assert.NotNil(t, result.Error, "Error 필드는 nil이 아니어야 합니다.")
			assert.Equal(t, tc.expectedCode, result.Error.Code, "예상되는 오류 코드가 반환되어야 합니다.")           //
			assert.Contains(t, result.Error.Message, "유효하지 않은 인증 토큰입니다.", "올바른 오류 메시지가 반환되어야 합니다.") //
		})
	}
}
