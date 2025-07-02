package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

// SupportedLanguages 서버가 지원하는 언어 코드 목록.
var SupportedLanguages = map[string]bool{
	"ko":    true,
	"ko-KR": true,
	"en-US": true,
	"en":    true,
	"":      true,
}

const DefaultLanguage = "ko"

type LanguageCandidate struct {
	Lang string
	Q    float64
}

// GetUserLanguage 사용자의 선호 언어 결정
// 우선순위 : 쿠키 > Accept-Language 헤더 > 기본 언어
func GetUserLanguage(c echo.Context) string {
	// 1. 쿠키에서 'lang' 쿠키 확인 (최우선)
	if cookie, err := c.Cookie("lang"); err == nil && cookie.Value != "" {
		if _, ok := SupportedLanguages[cookie.Value]; ok {
			return cookie.Value
		}
	}

	// 2. Accept-Language 헤더 확인
	acceptLanguageHeader := c.Request().Header.Get("Accept-Language")
	if acceptLanguageHeader != "" {
		candidates := parseAcceptLanguage(acceptLanguageHeader)

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Q > candidates[j].Q
		})

		for _, candidate := range candidates {
			if _, ok := SupportedLanguages[candidate.Lang]; ok {
				return candidate.Lang
			}
			baseLang := strings.Split(candidate.Lang, "-")[0]
			if _, ok := SupportedLanguages[baseLang]; ok {
				return baseLang
			}
		}
	}

	// 3. 모든 방법으로 언어를 찾지 못했거나 지원하지 않는 언어일 경우, 기본 언어 반환
	return DefaultLanguage
}

func parseAcceptLanguage(header string) []LanguageCandidate {
	var candidates []LanguageCandidate
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.Split(part, ";q=")
		langCode := segments[0]
		qValue := 1.0

		if len(segments) > 1 {
			if q, err := parseFloat(segments[1]); err == nil {
				qValue = q
			}
		}
		candidates = append(candidates, LanguageCandidate{Lang: langCode, Q: qValue})
	}
	return candidates
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f) // sscanf를 사용하여 float 파싱
	if err != nil {
		return 0, err
	}
	// q-value는 0.0에서 1.0 사이여야 함
	if f < 0.0 {
		f = 0.0
	} else if f > 1.0 {
		f = 1.0
	}
	return f, nil
}
