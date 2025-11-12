package utils

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"math/rand"
	"time"
	"github.com/wtppaul/course-service/internal/repository"
)

// Pola regex untuk karakter yang tidak aman dalam slug
var (
	nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9\s-]`)
	spaceRegex       = regexp.MustCompile(`[\s-]+`)
	// Inisialisasi generator angka acak
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// createSlug mengubah "Judul Kursus Keren!" menjadi "judul-kursus-keren"
func createSlug(title string) string {
	lower := strings.ToLower(title)
	noSpecial := nonAlphaNumRegex.ReplaceAllString(lower, "")
	slug := spaceRegex.ReplaceAllString(noSpecial, "-")
	return strings.Trim(slug, "-")
}

// GenerateUniqueSlug adalah fungsi utama
// Ia membuat slug dan memeriksanya ke DB
func GenerateUniqueSlug(ctx context.Context, title string, repo repository.ICourseRepository) (string, error) {
	// 1. Buat slug dasar
	baseSlug := createSlug(title)
	if baseSlug == "" {
		baseSlug = "course" // Fallback jika judul hanya berisi simbol
	}
	
	slug := baseSlug
	
	// 2. Periksa keunikan
	for i := 1; i < 10; i++ { // Coba 10 kali
		exists, err := repo.IsSlugInUse(ctx, slug)
		if err != nil {
			return "", fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		
		if !exists {
			return slug, nil // Slug ini unik!
		}
		
		// 3. Jika ada, tambahkan sufiks
		// Setelah 3x percobaan, gunakan sufiks acak
		if i < 3 {
			slug = fmt.Sprintf("%s-%d", baseSlug, i+1) // "judul-2", "judul-3"
		} else {
			// "judul-a1b2c"
			slug = fmt.Sprintf("%s-%s", baseSlug, randomString(5)) 
		}
	}
	
	// Jika 10x gagal (sangat tidak mungkin), kembalikan error
	return "", fmt.Errorf("failed to generate a unique slug for title: %s", title)
}

// randomString menghasilkan string acak
func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[random.Intn(len(letters))]
    }
    return string(b)
}