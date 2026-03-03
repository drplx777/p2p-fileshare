package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DefaultExpire = 7 * 24 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"userId"`
}

func NewToken(secret []byte, userID string, exp time.Duration) (string, error) {
	if exp <= 0 {
		exp = DefaultExpire
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseToken(secret []byte, tokenString string) (userID string, err error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return "", errors.New("invalid token")
	}
	return claims.UserID, nil
}
