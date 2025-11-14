package repository

import (
	"context"
	"time"
	"errors"
	"fmt" 

	"gorm.io/gorm"
	"github.com/google/uuid"
	"github.com/wtppaul/course-service/internal/models"
)

// --- Input Struct untuk Update ---
// Ini adalah praktik yang baik agar kita tidak mengizinkan
// pembaruan field sensitif (seperti TeacherID atau Slug)
type UpdateCourseInput struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Thumbnail   string               `json:"thumbnail,omitempty"`
	Price       float64              `json:"price"`
	Level       models.CourseLevel   `json:"level"`
	IsFree      bool                 `json:"isFree"`
	License     models.CourseLicense `json:"license"`
	// (Tambahkan CategoryIDs, TagIDs jika Anda ingin mengizinkan pembaruan di sini)
}

type CourseFilters struct {
	Status    		[]string // ["PUBLISHED", "APPROVED", .....])
	Level     		[]string 
	CategorySlugs []string 
	TagSlugs      []string 
	TeacherID 		uuid.UUID
	Page      		int
	Limit     		int
}

type ICourseRepository interface {

	// --- FUNGSI COURSE ---
	CreateCourse(ctx context.Context, course *models.Course) error
	UpdateCourse(ctx context.Context, courseID uuid.UUID, input UpdateCourseInput) (*models.Course, error)
	UpdateCourseTags(ctx context.Context, courseID uuid.UUID, tagIDs []uuid.UUID) error
	UpdateCourseStatus(ctx context.Context, courseID uuid.UUID, newStatus models.CourseStatus) error
	
	// --- FUNGSI CHAPTER ---
	CreateChapter(ctx context.Context, chapter *models.Chapter) error
	GetChapterByID(ctx context.Context, chapterID uuid.UUID) (*models.Chapter, error)    
	UpdateChapter(ctx context.Context, chapter *models.Chapter) (*models.Chapter, error)
	ReorderChapters(ctx context.Context, courseID uuid.UUID, updates []ChapterReorderInput) error // ✅ BARU
	DeleteChapter(ctx context.Context, courseID uuid.UUID, chapterID uuid.UUID) error // ✅ BARU

		// --- FUNGSI LESSON ---
	CreateLesson(ctx context.Context, lesson *models.Lesson) error
	UpdateLesson(ctx context.Context, lesson *models.Lesson) (*models.Lesson, error) // ✅ BARU
	GetLessonByID(ctx context.Context, lessonID uuid.UUID) (*models.Lesson, error)    // ✅ BARU

	// Operasi untuk Publik/User (via BFF)
	GetCourseBySlug(ctx context.Context, slug string) (*models.Course, error) // Ini yang kita perbaiki
	GetPublishedCourses(ctx context.Context, page, limit int) ([]*models.Course, error)
	GetCourseDetails(ctx context.Context, courseID uuid.UUID) (*models.Course, error) // Mirip dengan Slug, tapi by ID
	GetCoursesByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]*models.Course, error)
	GetCourses(ctx context.Context, filters CourseFilters) ([]*models.Course, int64, error)
	
	// Operasi untuk Pricing (dipanggil oleh Payment-service)
	FindValidCoupon(ctx context.Context, code string) (*models.Coupon, error)
	GetActiveSalesForCourse(ctx context.Context, courseID uuid.UUID) ([]*models.Sale, error)

	IsSlugInUse(ctx context.Context, slug string) (bool, error)
	FindOrCreateTeacherByAuthID(ctx context.Context, authID string) (*models.Teacher, error)
}

type courseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) ICourseRepository {
	return &courseRepository{db: db}
}

// --- Implementasi Fungsi ---


func (r *courseRepository) CreateCourse(ctx context.Context, course *models.Course) error {
	return r.db.WithContext(ctx).Create(course).Error
}

// ✅ 
func (r *courseRepository) UpdateCourse(ctx context.Context, courseID uuid.UUID, input UpdateCourseInput) (*models.Course, error) {
	// 1. Ambil kursus yang ada
	var course models.Course
	if err := r.db.WithContext(ctx).First(&course, "id = ?", courseID).Error; err != nil {
		return nil, err // (Akan GORM.ErrRecordNotFound jika tidak ada)
	}

	// 2. Terapkan pembaruan dari input
	// (Ini mencegah 'slug', 'teacherId', 'status' di-update secara tidak sengaja)
	course.Title = input.Title
	course.Description = input.Description
	course.Thumbnail = input.Thumbnail
	course.Price = input.Price
	course.Level = input.Level
	course.IsFree = input.IsFree
	course.License = input.License
	
	// 3. Simpan perubahan
	if err := r.db.WithContext(ctx).Save(&course).Error; err != nil {
		return nil, err
	}
	
	return &course, nil
}

func (r *courseRepository) UpdateCourseStatus(ctx context.Context, courseID uuid.UUID, newStatus models.CourseStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.Course{}).
		Where("id = ?", courseID).
		Updates(map[string]interface{}{
			"status":     newStatus,
			"updated_at": time.Now(),
		}).Error
}

func (r *courseRepository) CreateChapter(ctx context.Context, chapter *models.Chapter) error {
	return r.db.WithContext(ctx).Create(chapter).Error
}

func (r *courseRepository) CreateLesson(ctx context.Context, lesson *models.Lesson) error {
	return r.db.WithContext(ctx).Create(lesson).Error
}

// ✅
func (r *courseRepository) GetCourseBySlug(ctx context.Context, slug string) (*models.Course, error) {
	var course models.Course
	err := r.db.WithContext(ctx).
		Preload("Chapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("chapters.order ASC")
		}).
		Preload("Chapters.Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lessons.order ASC")
		}).
		Preload("Categories").
		Preload("Tags").
		Preload("Teacher").
		// PERBAIKAN: Filter status DIHAPUS.
		// Service ini "bodoh" dan mengembalikan apa adanya.
		// BFF (Fastify) yang akan memutuskan apakah user boleh melihatnya.
		Where("slug = ?", slug).
		First(&course).Error
	
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetPublishedCourses(ctx context.Context, page, limit int) ([]*models.Course, error) {
	var courses []*models.Course
	offset := (page - 1) * limit

	// FUNGSI INI ("GetPublished") BOLEH memfilter status,
	// karena tujuannya jelas untuk katalog publik.
	err := r.db.WithContext(ctx).
		Preload("Categories"). 
		Preload("Tags").
		Where("status = ?", models.StatusPublished).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&courses).Error

	return courses, err
}

func (r *courseRepository) GetCourseDetails(ctx context.Context, courseID uuid.UUID) (*models.Course, error) {
	// Implementasi ini mirip dengan GetCourseBySlug, tapi pakai ID
	// dan juga tidak memfilter status.
	var course models.Course
	err := r.db.WithContext(ctx).
		Preload("Teacher").
		Preload("Chapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("chapters.order ASC")
		}).
		Preload("Chapters.Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lessons.order ASC")
		}).
		Preload("Categories").
		Preload("Tags").
		Where("id = ?", courseID).
		First(&course).Error
	
	return &course, err
}

// ✅ 
func (r *courseRepository) GetCoursesByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]*models.Course, error) {
	var courses []*models.Course
	err := r.db.WithContext(ctx).
		Where("teacher_id = ?", teacherID).
		Order("updated_at DESC").
		Find(&courses).Error
	return courses, err
}

// ✅ 
func (r *courseRepository) IsSlugInUse(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Course{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return true, err
	}
	return count > 0, nil
}

// ✅ 
func (r *courseRepository) FindOrCreateTeacherByAuthID(ctx context.Context, authID string) (*models.Teacher, error) {
	var teacher models.Teacher
	
	// 1. Coba temukan dulu
	err := r.db.WithContext(ctx).Where("auth_id = ?", authID).First(&teacher).Error
	
	if err == nil {
		return &teacher, nil // Ditemukan
	}

	// 2. Jika tidak ditemukan (error-nya GORM.ErrRecordNotFound)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 3. Buat "profil bayangan"
		// 💡 PENTING: Kita tidak tahu 'name' atau 'username' teacher.
		// BFF harus bertanggung jawab memanggil endpoint lain
		// untuk mengisi data ini nanti.
		// Untuk saat ini, kita buat placeholder.
		newTeacher := models.Teacher{
			AuthID:   authID,
			Name:     "Pending Sync", // Placeholder
			Username: "pending-" + uuid.NewString(), // Placeholder unik
		}
		
		if errCreate := r.db.WithContext(ctx).Create(&newTeacher).Error; errCreate != nil {
			return nil, errCreate // Gagal membuat
		}
		return &newTeacher, nil // Kembalikan profil baru
	}
	
	// 4. Jika error lain (misal DB mati)
	return nil, err
}

func (r *courseRepository) FindValidCoupon(ctx context.Context, code string) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.db.WithContext(ctx).
		Where("code = ? AND (expires_at IS NULL OR expires_at > ?)", code, time.Now()).
		Where("max_uses IS NULL OR current_uses < max_uses").
		First(&coupon).Error

	if err != nil {
		return nil, err // Kembalikan gorm.ErrRecordNotFound jika tidak ada
	}
	return &coupon, nil
}

func (r *courseRepository) GetActiveSalesForCourse(ctx context.Context, courseID uuid.UUID) ([]*models.Sale, error) {
	var sales []*models.Sale
	now := time.Now()

	// Ini query yang agak kompleks:
	// Cari 'sales' yang aktif (rentang waktu)
	// DAN terhubung ke 'courseID' ini melalui tabel 'course_sales'
	err := r.db.WithContext(ctx).
		Joins("JOIN course_sales cs ON cs.sale_id = sales.id").
		Where("cs.course_id = ?", courseID).
		Where("sales.start_date <= ? AND sales.end_date >= ?", now, now).
		Find(&sales).Error

	return sales, err
}

// ✅
// GetCourses secara dinamis memfilter dan melakukan paginasi kursus
func (r *courseRepository) GetCourses(ctx context.Context, filters CourseFilters) ([]*models.Course, int64, error) {
	var courses []*models.Course
	var total int64

	// Mulai query
	query := r.db.WithContext(ctx).Model(&models.Course{})
	countQuery := r.db.WithContext(ctx).Model(&models.Course{})

	// --- Terapkan Filter ---
	if len(filters.Status) > 0 {
		query = query.Where("status IN (?)", filters.Status)
		countQuery = countQuery.Where("status IN (?)", filters.Status)
	}

	if len(filters.Level) > 0 {
		query = query.Where("level IN (?)", filters.Level)
		countQuery = countQuery.Where("level IN (?)", filters.Level)
	}

	if filters.TeacherID != uuid.Nil {
		query = query.Where("teacher_id = ?", filters.TeacherID)
		countQuery = countQuery.Where("teacher_id = ?", filters.TeacherID)
	}

	// ✅ --- FILTER BARU UNTUK KATEGORI (via JOIN) ---
	if len(filters.CategorySlugs) > 0 {
		// Kita perlu JOIN ke tabel 'categories' melalui tabel 'course_categories'
		query = query.Joins(
			"JOIN course_categories cc ON cc.course_id = courses.id").
			Joins("JOIN categories cat ON cat.id = cc.category_id").
			Where("cat.slug IN (?)", filters.CategorySlugs)
		
		countQuery = countQuery.Joins(
			"JOIN course_categories cc ON cc.course_id = courses.id").
			Joins("JOIN categories cat ON cat.id = cc.category_id").
			Where("cat.slug IN (?)", filters.CategorySlugs)
	}

	// ✅ --- FILTER BARU UNTUK TAG (via JOIN) ---
	if len(filters.TagSlugs) > 0 {
		query = query.Joins(
			"JOIN course_tags ct ON ct.course_id = courses.id").
			Joins("JOIN tags t ON t.id = ct.tag_id").
			Where("t.slug IN (?)", filters.TagSlugs)
			
		countQuery = countQuery.Joins(
			"JOIN course_tags ct ON ct.course_id = courses.id").
			Joins("JOIN tags t ON t.id = ct.tag_id").
			Where("t.slug IN (?)", filters.TagSlugs)
	}

	// --- Hitung Total (sebelum Paginasi) ---
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.Course{}, 0, nil // Kembalikan array kosong jika tidak ada hasil
	}

	// --- Terapkan Paginasi & Urutan ---
	offset := (filters.Page - 1) * filters.Limit
	query = query.Order("created_at DESC").
		Offset(offset).
		Limit(filters.Limit)

	// --- Preload Relasi (Hanya yang ringan untuk list) ---
	query = query.Preload("Teacher", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, username") // Hanya pilih data yg perlu
	}).Preload("Categories")

	// --- Eksekusi Query ---
	if err := query.Find(&courses).Error; err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// ✅
func (r *courseRepository) UpdateCourseTags(ctx context.Context, courseID uuid.UUID, tagIDs []uuid.UUID) error {
	// GORM memiliki cara elegan untuk mengganti relasi many-to-many
	// menggunakan .Association(...).Replace(...)
	
	var course models.Course
	course.ID = courseID
	
	// 1. Buat slice dari struct Tag hanya dengan ID
	//    (Ini diperlukan GORM untuk .Replace)
	var tags []models.Tag
	for _, id := range tagIDs {
		tags = append(tags, models.Tag{ID: id})
	}

	// 2. Ganti semua relasi yang ada di tabel 'course_tags'
	//    untuk courseID ini dengan daftar tag yang baru.
	//    GORM akan menangani (DELETE lama) + (INSERT baru)
	//    secara transaksional.
	return r.db.WithContext(ctx).Model(&course).Association("Tags").Replace(tags)
}

// ✅
func (r *courseRepository) CreateChapter(ctx context.Context, chapter *models.Chapter) error {
	// ('chapter.ID' dan 'chapter.Slug' harus sudah di-set oleh handler)
	return r.db.WithContext(ctx).Create(chapter).Error
}

// ✅
// GetChapterByID mengambil satu bab
func (r *courseRepository) GetChapterByID(ctx context.Context, chapterID uuid.UUID) (*models.Chapter, error) {
	var chapter models.Chapter
	err := r.db.WithContext(ctx).Where("id = ?", chapterID).First(&chapter).Error
	if err != nil {
		return nil, err // Akan GORM.ErrRecordNotFound jika tidak ada
	}
	return &chapter, nil
}

// ✅
// UpdateChapter memperbarui data bab
func (r *courseRepository) UpdateChapter(ctx context.Context, chapter *models.Chapter) (*models.Chapter, error) {
	// Gunakan .Save() untuk memperbarui semua field,
	// atau .Model() & .Updates() untuk field spesifik
	err := r.db.WithContext(ctx).Save(chapter).Error
	if err != nil {
		return nil, err
	}
	return chapter, nil
}

// ✅ 
// ReorderChapters menjalankan batch update dalam satu transaksi
func (r *courseRepository) ReorderChapters(ctx context.Context, courseID uuid.UUID, updates []ChapterReorderInput) error {
	// Memulai transaksi
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		
		for _, item := range updates {
			// Menjalankan update di dalam transaksi
			// Kita juga memverifikasi 'course_id' untuk keamanan ekstra,
			// memastikan chapter yang di-update milik kursus yang benar.
			result := tx.Model(&models.Chapter{}).
				Where("id = ? AND course_id = ?", item.ID, courseID).
				Update("order", item.Order)

			if result.Error != nil {
				// Jika satu update gagal, seluruh transaksi di-rollback
				return result.Error
			}
			
			if result.RowsAffected == 0 {
				// Ini terjadi jika chapterId tidak ada ATAU tidak cocok dengan courseId
				// Kita batalkan transaksi untuk mencegah update parsial.
				return errors.New(
					fmt.Sprintf("Reorder failed: Chapter ID %s not found or does not belong to course ID %s", item.ID, courseID),
				)
			}
		}

		// Jika semua loop berhasil, commit transaksi
		return nil
	})
}


// ✅ IMPLEMENTASI BARU
// DeleteChapter menghapus chapter dan semua lesson di dalamnya (transaksional)
func (r *courseRepository) DeleteChapter(ctx context.Context, courseID uuid.UUID, chapterID uuid.UUID) error {
	// Memulai transaksi
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		
		// 1. Pastikan chapter ini milik course yang benar
		//    (Ini juga berfungsi sebagai cek kepemilikan)
		var chapter models.Chapter
		if err := tx.Where("id = ? AND course_id = ?", chapterID, courseID).First(&chapter).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("chapter not found or does not belong to this course")
			}
			return err // Error DB lain
		}

		// 2. Hapus semua lesson di dalam chapter ini
		//    (Sangat penting untuk menghindari data yatim piatu)
		if err := tx.Where("chapter_id = ?", chapterID).Delete(&models.Lesson{}).Error; err != nil {
			return err // Rollback jika gagal hapus lessons
		}

		// 3. Hapus chapter itu sendiri
		if err := tx.Where("id = ?", chapterID).Delete(&models.Chapter{}).Error; err != nil {
			return err // Rollback jika gagal hapus chapter
		}

		// 4. Commit transaksi
		return nil
	})
}


// ✅
// CreateLesson membuat lesson baru
func (r *courseRepository) CreateLesson(ctx context.Context, lesson *models.Lesson) error {
	return r.db.WithContext(ctx).Create(lesson).Error
}


// ✅
// GetLessonByID mengambil satu lesson berdasarkan ID
func (r *courseRepository) GetLessonByID(ctx context.Context, lessonID uuid.UUID) (*models.Lesson, error) {
	var lesson models.Lesson
	// Kita juga 'Join' untuk mendapatkan courseId, untuk validasi
	err := r.db.WithContext(ctx).
		Joins("JOIN chapters ON chapters.id = lessons.chapter_id").
		Select("lessons.*, chapters.course_id").
		Where("lessons.id = ?", lessonID).
		First(&lesson).Error
		
	if err != nil {
		return nil, err // Akan GORM.ErrRecordNotFound jika tidak ada
	}
	return &lesson, nil
}

// ✅
// UpdateLesson memperbarui data lesson
func (r *courseRepository) UpdateLesson(ctx context.Context, lesson *models.Lesson) (*models.Lesson, error) {
	// Gunakan .Save() untuk memperbarui semua field
	err := r.db.WithContext(ctx).Save(lesson).Error
	if err != nil {
		return nil, err
	}
	return lesson, nil
}
