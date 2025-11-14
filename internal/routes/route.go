package routes

import (
	"net/http"

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
		// Rute yang berpusat pada Course
		courses := internal.Group("/courses")
		{
			courses.GET("", courseHandler.GetCourses) 											// GET /internal/courses?status=...
			courses.POST("", courseHandler.CreateCourse)                 		// POST /internal/courses
			courses.GET("/slug/:slug", courseHandler.GetCourseBySlug)    		// GET /internal/courses/slug/nama-slug
			courses.GET("/public", courseHandler.GetPublishedCourses) 			// GET /internal/courses/public
			courses.PATCH("/:id/status", courseHandler.UpdateCourseStatus) 	// PATCH /internal/courses/uuid/status
			courses.PATCH("/:id/tags", courseHandler.UpdateCourseTags) 			// PATCH /internal/courses/uuid/tags
			courses.POST("/:courseId/chapters", courseHandler.CreateChapter)// POST /internal/courses/:courseId/chapters
			
			// Endpoint pricing untuk Payment-service
			// courses.GET("/:id/pricing", courseHandler.GetPricingDetails)
		}

		// Rute yang berpusat pada Teacher
		teachers := internal.Group("/teachers")
		{
			// GET /internal/teachers/uuid/courses
			teachers.GET("/:teacherId/courses", courseHandler.GetCoursesByTeacherID)
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