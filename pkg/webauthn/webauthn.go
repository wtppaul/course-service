// pkg/webauthn/webauthn.go
package webauthn

import (
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthn adalah instance global yang akan kita gunakan di service
var WebAuthn *webauthn.WebAuthn

// InitWebAuthn menginisialisasi konfigurasi WebAuthn
func InitWebAuthn() error {
	// Ambil konfigurasi dari kode Node.js Anda
	rpID := "localhost.test"
	rpName := "Estaphet"
	rpOrigins := []string{
		"https://auth.localhost.test",
		"https://app.localhost.test",
	}

	var err error
	WebAuthn, err = webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpName,
		RPOrigins:     rpOrigins,
		// Kita bisa tambahkan timeout, dll di sini nanti
	})

	if err != nil {
		return err
	}

	return nil
}