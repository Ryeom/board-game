package auth

import (
	"errors"
	"strings"

	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/log"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenStr := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(tokenStr, "Bearer ") {
			log.Logger.Warningf("JWTMiddleware - No token provided or malformed Authorization header from IP: %s", c.RealIP())
			return errors.New(resp.ErrorCodeAuthInvalidToken)
		}

		tokenOnly := strings.TrimPrefix(tokenStr, "Bearer ")

		token, err := jwt.Parse(tokenOnly, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			log.Logger.Errorf("JWTMiddleware - Token parsing failed: %v from IP: %s", err, c.RealIP())
			return errors.New(resp.ErrorCodeAuthInvalidToken)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			log.Logger.Errorf("JWTMiddleware - Invalid token claims or token not valid from IP: %s", c.RealIP())
			return errors.New(resp.ErrorCodeAuthInvalidToken)
		}

		isBlacklisted, err := IsTokenBlacklisted(c.Request().Context(), tokenOnly)
		if err != nil {
			log.Logger.Errorf("JWTMiddleware - Error checking blacklist for token: %v from IP: %s", err, c.RealIP())
			return errors.New(resp.ErrorCodeAuthTokenBlacklistCheckFailed)
		}
		if isBlacklisted {
			log.Logger.Warningf("JWTMiddleware - Blacklisted token detected for user ID: %s from IP: %s", claims["user_id"], c.RealIP())
			return errors.New(resp.ErrorCodeAuthTokenBlacklisted)
		}

		// 클레임에서 "userID" 대신 "user_id" 사용
		userID, ok := claims["user_id"].(string)
		if !ok {
			log.Logger.Errorf("JWTMiddleware - user_id claim not found or invalid type in token from IP: %s", c.RealIP())
			return errors.New(resp.ErrorCodeAuthInvalidToken)
		}
		c.Set("userID", userID)
		return next(c)
	}
}
