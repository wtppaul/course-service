package handler

import (
	"errors"
	"net/http"

	"gorm.io/gorm"
	"github.com/gin-gonic/gin"
	// "github.com/google/uuid"

	"github.com/wtppaul/course-service/internal/models"
	"github.com/wtppaul/course-service/internal/repository"
	"github.com/wtppaul/course-service/internal/utils" 
)

type CourseHandler struct {
	repo repository.ICourseRepository
}

func NewCourseHandler(repo repository.ICourseRepository) *CourseHandler {
	return &CourseHandler{repo: repo}
}

// === HANDLER PUBLIK (via BFF) ===

// GetCourseBySlug (GET /internal/courses/slug/:slug)
// Ini adalah endpoint "bodoh"
func (h *CourseHandler) GetCourseBySlug(c *gin.Context) {
	slug := c.Param("slug")

	course, err := h.repo.GetCourseBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, course)
}

// GetPublishedCourses (GET /internal/courses/public)
func (h *CourseHandler) GetPublishedCourses(c *gin.Context) {
	// (Di sini Anda bisa menambahkan parsing query param 'page' dan 'limit')
	courses, err := h.repo.GetPublishedCourses(c.Request.Context(), 1, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, courses)
}

// === HANDLER PRIVAT (via BFF/Gateway) ===

// CreateCourse (POST /internal/courses)
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	// 1. Ambil "Paspor" (AuthID String) dari context
	authIDStr, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 2. ✅ PERBAIKAN: Tukar "Paspor" (AuthID) dengan "Profil" (Teacher)
	// Ini akan mencari teacher, atau membuat profil baru jika tidak ada.
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve teacher profile"})
		return
	}

	// 3. Bind JSON body
	var input struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 4. Buat slug yang unik
	slug, err := utils.GenerateUniqueSlug(c.Request.Context(), input.Title, h.repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate slug"})
		return
	}

	// 5. Buat objek Course
	course := &models.Course{
		Title:     input.Title,
		Slug:      slug,
		TeacherID: teacher.ID, 
		Status:    models.StatusDraft,
		License:   models.LicenseNT,
	}

	// 6. Simpan ke DB
	if err := h.repo.CreateCourse(c.Request.Context(), course); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
		return
	}

	c.JSON(http.StatusCreated, course)
}

// UpdateCourseStatus (PATCH /internal/courses/:id/status)
func (h *CourseHandler) UpdateCourseStatus(c *gin.Context) {
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}

	// (Di sini, BFF seharusnya sudah memverifikasi bahwa
	// X-Authenticated-User-ID adalah pemilik kursus ini
	// atau seorang Admin/Curator)

	var input struct {
		Status models.CourseStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// (Validasi input status jika perlu)
	
	err = h.repo.UpdateCourseStatus(c.Request.Context(), courseID, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// --- TODO: Handler untuk Pricing ---
// (Handler ini akan dipanggil oleh Payment-service)

// GetPricingDetails (GET /internal/courses/:id/pricing)
func (h *CourseHandler) GetPricingDetails(c *gin.Context) {
	// 1. Ambil harga dasar
	// 2. Ambil sales aktif (via repo)
	// 3. Kembalikan semua info harga
}

// ValidateCoupon (POST /internal/coupons/validate)
func (h *CourseHandler) ValidateCoupon(c *gin.Context) {
	// 1. Ambil kode kupon
	// 2. Panggil repo.FindValidCoupon
	// 3. Kembalikan detail kupon jika valid
}