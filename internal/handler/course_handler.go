package handler

import (
	"errors"
	"net/http"
	"strconv" 
	"fmt"

	"gorm.io/gorm"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/wtppaul/course-service/internal/models"
	"github.com/wtppaul/course-service/internal/repository"
	"github.com/wtppaul/course-service/internal/utils" 
)

const (
	DefaultPage  = 1
	DefaultLimit = 10
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


// ✅ 
// GetCoursesByTeacherID (GET /internal/teachers/:teacherId/courses)
func (h *CourseHandler) GetCoursesByTeacherID(c *gin.Context) {
	// 1. Ambil Teacher ID (Profil UUID) dari URL
	teacherIDStr := c.Param("teacherId")
	teacherID, err := uuid.Parse(teacherIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid teacher ID format"})
		return
	}

	// 2. ⛔️ HAPUS LOGIKA KEAMANAN DARI SINI
	//    (Middleware internal kita sudah memvalidasi X-Internal-Secret)
	//    Logika "apakah user ini pemilik" adalah tugas BFF, BUKAN service ini.
	/*
	authIDStr, _ := c.Get("authenticatedUserID")
	teacher, _ := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authIDStr.(string))
	if teacher.ID != teacherID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You can only view your own courses"})
		return
	}
	*/

	// 3. Panggil Repository (Logika "Bodoh")
	//    "Ambilkan saya semua kursus (termasuk draft) untuk teacher ini"
	courses, err := h.repo.GetCoursesByTeacherID(c.Request.Context(), teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get courses"})
		return
	}

	c.JSON(http.StatusOK, courses)
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

// ✅
// GetCourses (GET /internal/courses)
// Menerima query params dinamis dari BFF
func (h *CourseHandler) GetCourses(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Parsing Paginasi
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > 100 { // Batasi max limit
		limit = DefaultLimit
	}

	// 2. Parsing Filter
	statusFilter := c.QueryArray("status")
	levelFilter := c.QueryArray("level")
	
	// ✅ (BARU) Ambil filter kategori dan tag
	categoryFilter := c.QueryArray("category") // ?category=web&category=mobile
	tagFilter := c.QueryArray("tag")         // ?tag=react&tag=go

	filters := repository.CourseFilters{
		Status:        statusFilter,
		Level:         levelFilter,
		CategorySlugs: categoryFilter, // ✅ (BARU)
		TagSlugs:      tagFilter,      // ✅ (BARU)
		Page:          page,
		Limit:         limit,
	}

	// 3. Panggil Repository
	courses, total, err := h.repo.GetCourses(ctx, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	// 4. Kembalikan respons terstruktur (untuk paginasi)
	c.JSON(http.StatusOK, gin.H{
		"data": courses,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ✅ --- HANDLER BARU ---
// GetCourseById (GET /internal/courses/:id)
func (h *CourseHandler) GetCourseById(c *gin.Context) {
	ctx := c.Request.Context()
	
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}

	course, err := h.repo.GetCourseDetails(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, course)
}

// ✅
// UpdateCourseTags (PATCH /internal/courses/:id/tags)
func (h *CourseHandler) UpdateCourseTags(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Ambil CourseID dari URL
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}
	
	// (BFF sudah memvalidasi kepemilikan)
	
	// 2. Bind JSON body (array of tag IDs)
	var input struct {
		TagIDs []uuid.UUID `json:"tagIds" binding:"required"` // Mengharapkan array UUID
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 3. Panggil repository
	//    (Jika input.TagIDs kosong, GORM akan menghapus semua tag)
	if err := h.repo.UpdateCourseTags(ctx, courseID, input.TagIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags updated successfully"})
}

// ✅
// CreateChapter (POST /internal/courses/:courseId/chapters)
func (h *CourseHandler) CreateChapter(c *gin.Context) {
	ctx := c.Request.Context()
	
	// 1. Ambil CourseID dari URL
	courseIDStr := c.Param("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}

	// 2. Ambil AuthID (Teacher) dari header
	// (Middleware sudah memvalidasi X-Internal-Secret)
	// BFF "pintar" HARUS sudah memvalidasi kepemilikan.
	// Service "bodoh" ini hanya mengeksekusi.
	authID := c.GetString("authenticatedUserID")
	if authID == "" {
		// Ini seharusnya tidak pernah terjadi jika middleware internal berjalan
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Missing auth context"})
		return
	}
	
	// 3. Bind JSON body (data Chapter)
	var input struct {
		Title string `json:"title" binding:"required"`
		Order int    `json:"order"` // Order bisa 0 atau di-set
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Buat objek Chapter
	chapter := &models.Chapter{
		CourseID: courseID,
		Title:    input.Title,
		Order:    input.Order,
		// Buat slug unik untuk chapter (misal: "chapter-judul-chapter-acak")
		// Kita tidak perlu cek DB untuk ini, cukup buat unik
		Slug:     fmt.Sprintf("chapter-%s-%s", utils.CreateSlug(input.Title), utils.RandomString(6)),
	}

	// 5. Simpan ke DB
	if err := h.repo.CreateChapter(ctx, chapter); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chapter"})
		return
	}

	c.JSON(http.StatusCreated, chapter)
}

// ✅
// UpdateChapter (PATCH /internal/courses/:courseId/chapters/:chapterId)
func (h *CourseHandler) UpdateChapter(c *gin.Context) {
	// 1. Ambil ID dari URL
	courseIDStr := c.Param("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}
	chapterIDStr := c.Param("chapterId")
	chapterID, err := uuid.Parse(chapterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter ID format"})
		return
	}

	// 2. Ambil "Paspor" (AuthID) dari context
	authID, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 3. Bind JSON body (hanya field yang boleh di-update)
	var input struct {
		Title string `json:"title"`
		Order *int   `json:"order"` // Pointer agar bisa bedakan 0 vs. 'tidak dikirim'
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Verifikasi Kepemilikan (Defense in Depth)
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify teacher"})
		return
	}
	course, err := h.repo.GetCourseDetails(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	if course.TeacherID != teacher.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this course"})
		return
	}

	// 5. Ambil data Chapter yang ada
	chapter, err := h.repo.GetChapterByID(c.Request.Context(), chapterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
		return
	}

	// 6. Terapkan perubahan
	if input.Title != "" {
		chapter.Title = input.Title
		// (Opsional: Buat slug baru jika title berubah?
		//  Untuk Chapter, ini mungkin OK, tapi kita biarkan dulu)
	}
	if input.Order != nil {
		chapter.Order = *input.Order
	}

	// 7. Simpan ke DB
	updatedChapter, err := h.repo.UpdateChapter(c.Request.Context(), chapter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chapter"})
		return
	}

	c.JSON(http.StatusOK, updatedChapter)
}

// ✅
// ReorderChapters (POST /internal/courses/:courseId/chapters/reorder)
func (h *CourseHandler) ReorderChapters(c *gin.Context) {
	// 1. Ambil Course ID dari URL
	courseIDStr := c.Param("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}

	// 2. Ambil "Paspor" (AuthID) dari context
	authID, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 3. Bind JSON body (Array of updates)
	var input []repository.ChapterReorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: expected array of {id, order}"})
		return
	}

	// 4. Verifikasi Kepemilikan (Defense in Depth)
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify teacher"})
		return
	}
	course, err := h.repo.GetCourseDetails(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	if course.TeacherID != teacher.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this course"})
		return
	}

	// 5. Panggil Repository (yang akan menjalankan Transaksi)
	err = h.repo.ReorderChapters(c.Request.Context(), courseID, input)
	if err != nil {
		// Jika transaksi gagal (misal, salah satu chapterId tidak valid)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to reorder chapters: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chapters reordered successfully"})
}


// ✅ 
// DeleteChapter (DELETE /internal/courses/:courseId/chapters/:chapterId)
func (h *CourseHandler) DeleteChapter(c *gin.Context) {
	// 1. Ambil ID dari URL
	courseIDStr := c.Param("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}
	chapterIDStr := c.Param("chapterId")
	chapterID, err := uuid.Parse(chapterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter ID format"})
		return
	}

	// 2. Ambil "Paspor" (AuthID) dari context
	authID, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 3. Verifikasi Kepemilikan (Defense in Depth)
	//    (Repository akan memverifikasi ini lagi di dalam transaksi,
	//     tapi kita cek di sini dulu untuk 'fail-fast')
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify teacher"})
		return
	}
	course, err := h.repo.GetCourseDetails(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	if course.TeacherID != teacher.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this course"})
		return
	}

	// 4. Panggil Repository (yang akan menjalankan Transaksi)
	//    Repository akan menghapus lesson DAN chapter
	err = h.repo.DeleteChapter(c.Request.Context(), courseID, chapterID)
	if err != nil {
		// Jika transaksi gagal (misal, chapter tidak ditemukan)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Failed to delete chapter: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chapter deleted successfully"})
}


// ✅
// CreateLesson (POST /internal/chapters/:chapterId/lessons)
func (h *CourseHandler) CreateLesson(c *gin.Context) {
	// 1. Ambil Chapter ID dari URL
	chapterIDStr := c.Param("chapterId")
	chapterID, err := uuid.Parse(chapterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter ID format"})
		return
	}

	// 2. Ambil "Paspor" (AuthID) dari context
	authID, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 3. Bind JSON body
	var input struct {
		Title      string `json:"title" binding:"required"`
		Order      int    `json:"order"`
		PlaybackID string `json:"playbackId"` // ID Video (dari Upload-service)
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Verifikasi Kepemilikan (Defense in Depth)
	//    Kita harus memastikan chapter ini milik teacher yang benar
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify teacher"})
		return
	}

	chapter, err := h.repo.GetChapterByID(c.Request.Context(), chapterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
		return
	}

	course, err := h.repo.GetCourseDetails(c.Request.Context(), chapter.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course (owner of chapter) not found"})
		return
	}

	if course.TeacherID != teacher.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this chapter's course"})
		return
	}
	
	// 5. Buat objek Lesson
	lesson := &models.Lesson{
		Title:      input.Title,
		Order:      input.Order,
		ChapterID:  chapterID,
		PlaybackID: input.PlaybackID, // 💡 (Akan diisi nanti oleh upload-service)
	}

	// 6. Simpan ke DB
	if err := h.repo.CreateLesson(c.Request.Context(), lesson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lesson"})
		return
	}

	c.JSON(http.StatusCreated, lesson)
}


// ✅ --- HANDLER BARU UNTUK UPDATE LESSON ---
// UpdateLesson (PATCH /internal/lessons/:lessonId)
func (h *CourseHandler) UpdateLesson(c *gin.Context) {
	// 1. Ambil Lesson ID dari URL
	lessonIDStr := c.Param("lessonId")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lesson ID format"})
		return
	}

	// 2. Ambil "Paspor" (AuthID) dari context
	authID, exists := c.Get("authenticatedUserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// 3. Bind JSON body (field yang boleh di-update)
	var input struct {
		Title      *string `json:"title"`
		Order      *int    `json:"order"`
		PlaybackID *string `json:"playbackId"`
		IsPreview  *bool   `json:"isPreview"`
		// 'duration' akan di-update oleh service lain (upload-pipeline)
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Ambil data Lesson yang ada (termasuk courseId)
	lesson, err := h.repo.GetLessonByID(c.Request.Context(), lessonID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lesson not found"})
		return
	}

	// 5. Verifikasi Kepemilikan (Defense in Depth)
	teacher, err := h.repo.FindOrCreateTeacherByAuthID(c.Request.Context(), authID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify teacher"})
		return
	}
	
	// (lesson.CourseID didapat dari 'Join' di GetLessonByID)
	course, err := h.repo.GetCourseDetails(c.Request.Context(), lesson.CourseID) 
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course (owner of lesson) not found"})
		return
	}
	if course.TeacherID != teacher.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this lesson's course"})
		return
	}

	// 6. Terapkan perubahan (hanya jika nilainya dikirim)
	if input.Title != nil {
		lesson.Title = *input.Title
	}
	if input.Order != nil {
		lesson.Order = *input.Order
	}
	if input.PlaybackID != nil {
		lesson.PlaybackID = *input.PlaybackID
	}
	if input.IsPreview != nil {
		lesson.IsPreview = *input.IsPreview
	}

	// 7. Simpan ke DB
	updatedLesson, err := h.repo.UpdateLesson(c.Request.Context(), lesson)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lesson"})
		return
	}

	c.JSON(http.StatusOK, updatedLesson)
}
