package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/wtppaul/course-service/internal/handler"
	"github.com/wtppaul/course-service/internal/middleware"
)

// SetupCourseRoutes merakit semua rute untuk service ini
func SetupCourseRoutes(router *gin.Engine, courseHandler *handler.CourseHandler) {

	// Grup /internal dilindungi oleh middleware
	// Ini adalah service "bodoh", tidak ada rute publik
	internal := router.Group("/internal")
	internal.Use(middleware.InternalAuthMiddleware())
	{
		courses := internal.Group("/courses")
		{
			// Endpoint yang kita diskusikan di strategi
			courses.POST("", courseHandler.CreateCourse)                 		// POST /internal/courses
			courses.GET("/slug/:slug", courseHandler.GetCourseBySlug)    		// GET /internal/courses/slug/nama-slug
			courses.GET("/public", courseHandler.GetPublishedCourses) 			// GET /internal/courses/public
			courses.PATCH("/:id/status", courseHandler.UpdateCourseStatus) 	// PATCH /internal/courses/uuid/status
		
			// Endpoint pricing untuk Payment-service
			// courses.GET("/:id/pricing", courseHandler.GetPricingDetails)
		}

		// coupons := internal.Group("/coupons")
		// {
		// 	coupons.POST("/validate", courseHandler.ValidateCoupon)
		// }
	}
	
	// Rute Health Check Sederhana (Publik)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP", "service": "course-service"})
	})
}