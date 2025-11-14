package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	
	// --- GANTI SEMUA IMPORT KE 'course-service' ---
	"github.com/wtppaul/course-service/internal/config"
	"github.com/wtppaul/course-service/internal/database"
	"github.com/wtppaul/course-service/internal/handler"
	"github.com/wtppaul/course-service/internal/redis"
	"github.com/wtppaul/course-service/internal/repository"
	"github.com/wtppaul/course-service/internal/routes"
)

func main() {
	// 0️⃣ Kustom Validator (Saat ini tidak ada, bisa ditambahkan nanti)
	// if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
	//     v.RegisterValidation(...)
	// }

	// 1️⃣ Load environment variables
	config.Load()
	
	// 2️⃣ Setup database & redis
	database.InitDB()
	redis.InitRedis()

	// 3️⃣ Hapus Inisialisasi WebAuthn (Tidak relevan)

	// 4️⃣ Init Gin
	router := gin.Default()
	router.SetTrustedProxies(nil) // Atur trusted proxies Anda

	// 5️⃣ Setup handlers & services (Versi Course-service)
	
	// A. Inisialisasi Repository (Dependensi: Database)
	courseRepo := repository.NewCourseRepository(database.DB)

	// B. Inisialisasi Handler (Dependensi: Repository)
	courseHandler := handler.NewCourseHandler(courseRepo)
	
	// (Jika Anda meng-upgrade /health, inisialisasi health handler di sini)
	// healthService := service.NewHealthService()
	// healthHandler := handler.NewHealthHandler(healthService)


	// 6️⃣ Centralized route setup
	// Memanggil SetupCourseRoutes dari 'course-service/internal/routes'
	routes.SetupCourseRoutes(
		router,
		courseHandler, 
		// healthHandler, // (Tambahkan ini jika Anda upgrade /health)
	)

	// 7️⃣ Run server
	port := config.GetEnv("SERVER_PORT", "8081")
	fmt.Println("🚀 Course-service running at http://localhost:" + port)
	log.Fatal(router.Run(":" + port))
}