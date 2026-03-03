package middleware

import (
	"strings"

	"github.com/drplx/p2p-fileshare/internal/auth"
	"github.com/gofiber/fiber/v3"
)

const UserIDKey = "userID"

// RequireAuth parses Bearer token and sets userID in Locals. Returns 401 if missing/invalid.
func RequireAuth(secret []byte) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing Authorization header"})
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			return c.Status(401).JSON(fiber.Map{"error": "invalid Authorization format"})
		}
		tokenStr := strings.TrimSpace(authHeader[len(prefix):])
		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}
		userID, err := auth.ParseToken(secret, tokenStr)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid or expired token"})
		}
		c.Locals(UserIDKey, userID)
		return c.Next()
	}
}

// GetUserID returns the user ID set by RequireAuth, or empty string if not authenticated.
func GetUserID(c fiber.Ctx) string {
	v := c.Locals(UserIDKey)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
