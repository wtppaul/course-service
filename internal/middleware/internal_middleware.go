package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// InternalAuthMiddleware memvalidasi X-Internal-Secret
func InternalAuthMiddleware() gin.HandlerFunc {
	// Ambil secret dari ENV sekali saja saat startup
	// Ini JAUH lebih cepat daripada os.Getenv() di setiap request
	internalSecret := os.Getenv("INTERNAL_API_SECRET")

	if internalSecret == "" {
		// Jika service tidak dikonfigurasi dengan benar,
		// jangan pernah biarkan request apa pun masuk.
		panic("FATAL: INTERNAL_API_SECRET is not set")
	}

	return func(c *gin.Context) {
		secret := c.GetHeader("X-Internal-Secret")

		if secret != internalSecret {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: Invalid internal secret"})
			return
		}

		// Jika secret valid, kita PERCAYA header X-Authenticated-User-ID
		// yang dikirim oleh gateway.
		userID := c.GetHeader("X-Authenticated-User-ID")
		if userID != "" {
			c.Set("authenticatedUserID", userID) // Set di context untuk handler
		}

		c.Next()
	}
}