package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/log"
	"github.com/golang-jwt/jwt"
	"github.com/spf13/viper"
)

var jwtSecret []byte

func Initialize() {
	jwtSecret = []byte(viper.GetString("jwt.secret"))
	if len(jwtSecret) == 0 {
		panic("JWT secret is not configured or empty. Please set 'jwt.secret' in settings.toml.")
	}
}

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", errors.New("user_id claim not found or invalid type")
		}
		return userID, nil
	}

	return "", err
}

func GetTokenRemainingValidity(tokenStr string) (time.Duration, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token claims or token not valid")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return 0, errors.New("exp claim not found or invalid type")
	}

	expirationTime := time.Unix(int64(exp), 0)
	remainingTime := time.Until(expirationTime)

	if remainingTime <= 0 {
		return 0, nil // 이미 만료되었거나 남은 시간이 없는 경우
	}

	return remainingTime, nil
}

func AddTokenToBlacklist(ctx context.Context, tokenStr string) error {
	remainingTime, err := GetTokenRemainingValidity(tokenStr)
	if err != nil {
		// 토큰 파싱 또는 유효성 검증 실패 시 블랙리스트에 추가하지 않음
		log.Logger.Warningf("AddTokenToBlacklist: Failed to get token validity for %s: %v", tokenStr, err)
		return fmt.Errorf("invalid token for blacklisting: %w", err)
	}
	if remainingTime <= 0 {
		// 이미 만료된 토큰은 블랙리스트에 추가할 필요 없음
		log.Logger.Infof("AddTokenToBlacklist: Token %s is already expired, no need to blacklist.", tokenStr)
		return nil
	}

	// 키는 "jwt:blacklist:<token_hash_or_jti>"
	// TODO : Better -> JTI 사용
	blacklistKey := fmt.Sprintf("jwt:blacklist:%s", tokenStr)
	err = redisutil.SetStringWithTTL(redisutil.RedisTargetUser, blacklistKey, "blacklisted", remainingTime)
	if err != nil {
		log.Logger.Errorf("Failed to add token %s to blacklist: %v", tokenStr, err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}
	log.Logger.Infof("Token %s added to blacklist with TTL: %v", tokenStr, remainingTime)
	return nil
}

func IsTokenBlacklisted(ctx context.Context, tokenStr string) (bool, error) {
	blacklistKey := fmt.Sprintf("jwt:blacklist:%s", tokenStr)
	val, err := redisutil.GetString(redisutil.RedisTargetUser, blacklistKey)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist for token %s: %w", tokenStr, err)
	}

	return val == "blacklisted", nil
}
