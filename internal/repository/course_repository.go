package repository

import (
	"context"
	"time"
	"errors"

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

type ICourseRepository interface {
	// Operasi untuk Teacher/Admin
	CreateCourse(ctx context.Context, course *models.Course) error
	UpdateCourse(ctx context.Context, courseID uuid.UUID, input UpdateCourseInput) (*models.Course, error)
	UpdateCourseStatus(ctx context.Context, courseID uuid.UUID, newStatus models.CourseStatus) error
	CreateChapter(ctx context.Context, chapter *models.Chapter) error
	CreateLesson(ctx context.Context, lesson *models.Lesson) error
	
	// Operasi untuk Publik/User (via BFF)
	GetCourseBySlug(ctx context.Context, slug string) (*models.Course, error) // Ini yang kita perbaiki
	GetPublishedCourses(ctx context.Context, page, limit int) ([]*models.Course, error)
	GetCourseDetails(ctx context.Context, courseID uuid.UUID) (*models.Course, error) // Mirip dengan Slug, tapi by ID
	GetCoursesByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]*models.Course, error)
	
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

// ✅ IMPLEMENTASI FUNGSI BARU
// FindOrCreateTeacherByAuthID menukar "Paspor" (AuthID) dengan "Profil" (Teacher)
// Ini adalah jantung dari sinkronisasi data microservice.
func (r *courseRepository) FindOrCreateTeacherByAuthID(ctx context.Context, authID string) (*models.Teacher, error) {
	var teacher models.Teacher
	
	// 1. Coba temukan teacher berdasarkan AuthID (Paspor)
	err := r.db.WithContext(ctx).Where("auth_id = ?", authID).First(&teacher).Error
	
	if err == nil {
		// Ditemukan, kembalikan profil yang ada
		return &teacher, nil
	}

	// 2. Jika tidak ditemukan (gorm.ErrRecordNotFound)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Buat profil teacher baru.
		// CATATAN: Kita tidak tahu 'Name' atau 'Username' teacher.
		// BFF (App-service) harus bertanggung jawab untuk memanggil
		// endpoint lain (misal: PATCH /internal/teachers/sync-profile)
		// untuk mengisi data ini nanti.
		newTeacher := models.Teacher{
			AuthID:   authID,
			Name:     "New Teacher", // Nama sementara
			Username: "user-" + uuid.NewString()[:8], // Username unik sementara
		}

		// Simpan teacher baru
		if err := r.db.WithContext(ctx).Create(&newTeacher).Error; err != nil {
			return nil, err // Gagal membuat profil baru
		}
		
		// Kembalikan profil yang baru dibuat
		return &newTeacher, nil
	}

	// 3. Jika terjadi error database lain
	return nil, err
}

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

// IsSlugInUse memeriksa apakah slug sudah ada di database
func (r *courseRepository) IsSlugInUse(ctx context.Context, slug string) (bool, error) {
	var count int64
	// Gunakan query COUNT() yang sangat cepat
	err := r.db.WithContext(ctx).
		Model(&models.Course{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	
	return count > 0, nil
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