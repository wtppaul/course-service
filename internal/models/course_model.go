package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tipe Enum (string)
type CourseStatus 	string
type CourseLicense 	string
type CourseLevel 		string
type DiscountType 	string
type Role 					string

const (
	StatusDraft        CourseStatus = "DRAFT"
	StatusIncomplete   CourseStatus = "INCOMPLETE"
	StatusPending      CourseStatus = "PENDING_REVIEW"
	StatusFollowedUp   CourseStatus = "FOLLOWED_UP"
	StatusApproved	   CourseStatus = "APPROVED"
	StatusPublished    CourseStatus = "PUBLISHED"
	StatusRejected     CourseStatus = "REJECTED"
	StatusUnpublished	 CourseStatus = "UNPUBLISHED"
	StatusArchived		 CourseStatus = "ARCHIVED"

	LicenseEE 				 CourseLicense = "EE" // Exclusive-Estaphet
	LicenseET 				 CourseLicense = "ET" // Exclusive-Teacher
	LicenseNT 				 CourseLicense = "NT" // Non-Exclusive

	LevelBeginner 		 CourseLevel = "BEGINNER"
	LevelIntermediate  CourseLevel = "INTERMEDIATE"
	LevelAdvanced 		 CourseLevel = "ADVANCED"
	
	DiscountPercentage DiscountType = "PERCENTAGE"
	DiscountFixed      DiscountType = "FIXED_AMOUNT"
)

// Course memetakan tabel 'courses'
type Course struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Title       string          `gorm:"not null" json:"title"`
	Description string          `json:"description"`
	Thumbnail   string          `json:"thumbnail,omitempty"`
	Price       float64         `gorm:"default:0" json:"price"` // Harga dasar
	TeacherID   uuid.UUID       `gorm:"type:uuid;not null" json:"teacherId"`
	Slug        string          `gorm:"unique;not null" json:"slug"`
	Level       CourseLevel     `gorm:"type:varchar(50)" json:"level"`
	Status      CourseStatus    `gorm:"type:varchar(50);default:'DRAFT'" json:"status"`
	IsFree      bool            `gorm:"default:false" json:"isFree"`
	License     CourseLicense   `gorm:"type:varchar(10);default:'NT'" json:"license"`
	CreatedAt   time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`

	// Relasi (GORM akan menanganinya)
	Teacher   Teacher     `json:"teacher,omitempty"`
	Chapters  []Chapter   `json:"chapters,omitempty"`
	Categories[]Category  `gorm:"many2many:course_categories;" json:"categories,omitempty"`
	Tags      []Tag       `gorm:"many2many:course_tags;" json:"tags,omitempty"`
	Sales     []Sale      `gorm:"many2many:course_sales;" json:"sales,omitempty"`
	Coupons   []Coupon    `gorm:"many2many:coupon_courses;" json:"coupons,omitempty"`
}

type Teacher struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AuthID string    `gorm:"unique;not null" json:"authId"`
	Name   string    `gorm:"not null" json:"name"`
	Bio    string    `json:"bio,omitempty"`
	Username string  `gorm:"unique;not null" json:"username"`
	Courses  []Course  `json:"-"` // Hindari circular dependency
}

// (Mungkin tidak diperlukan oleh course-service, tapi baik untuk kelengkapan)
type Student struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AuthID string    `gorm:"unique;not null" json:"authId"`
	Name   string    `gorm:"not null" json:"name"`
	Username string  `gorm:"unique;not null" json:"username"`
}

// Chapter memetakan tabel 'chapters'
type Chapter struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Title     string    `gorm:"not null" json:"title"`
	Order     int       `gorm:"not null" json:"order"`
	CourseID  uuid.UUID `gorm:"type:uuid;not null" json:"courseId"`
	Slug      string    `gorm:"unique;not null" json:"slug"`
	Lessons   []Lesson  `json:"lessons,omitempty"`
}

// Lesson memetakan tabel 'lessons'
type Lesson struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Order       int       `gorm:"not null" json:"order"`
	ChapterID   uuid.UUID `gorm:"type:uuid;not null" json:"chapterId"`
	Duration    int       `json:"duration,omitempty"` // durasi dalam detik
	PlaybackID  string    `gorm:"not null" json:"playbackId"` // ID dari Cloudflare Stream
	IsPreview   bool      `gorm:"default:false" json:"isPreview"`
}

// Category memetakan tabel 'categories'
type Category struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"unique;not null" json:"name"`
	Slug        string    `gorm:"unique;not null" json:"slug"`
	ParentID    *uuid.UUID `gorm:"type:uuid" json:"parentId,omitempty"` // Pointer untuk null
	// Relasi (jika diperlukan)
	Parent      *Category `gorm:"foreignkey:ParentID" json:"parent,omitempty"`
	Children    []Category `gorm:"foreignkey:ParentID" json:"children,omitempty"`
}

// Tag memetakan tabel 'tags'
type Tag struct {
	ID    uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name  string    `gorm:"unique;not null" json:"name"`
	Slug  string    `gorm:"unique;not null" json:"slug"`
}

// Sale memetakan tabel 'sales'
type Sale struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name          string       `gorm:"not null" json:"name"`
	DiscountType  DiscountType `gorm:"type:varchar(50);not null" json:"discountType"`
	DiscountValue float64      `gorm:"not null" json:"discountValue"`
	StartDate     time.Time    `gorm:"not null" json:"startDate"`
	EndDate       time.Time    `gorm:"not null" json:"endDate"`
	Courses       []Course     `gorm:"many2many:course_sales;" json:"-"` // Hindari circular dependency di JSON
}

// Coupon memetakan tabel 'coupons'
type Coupon struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Code          string       `gorm:"unique;not null" json:"code"`
	DiscountType  DiscountType `gorm:"type:varchar(50);not null" json:"discountType"`
	DiscountValue float64      `gorm:"not null" json:"discountValue"`
	ExpiresAt     *time.Time   `json:"expiresAt,omitempty"`
	MaxUses       int          `json:"maxUses,omitempty"`
	CurrentUses   int          `gorm:"default:0" json:"currentUses"`
	Courses       []Course     `gorm:"many2many:coupon_courses;" json:"-"`
	Categories    []Category   `gorm:"many2many:coupon_categories;" json:"-"`
}

// Fungsi hook GORM untuk UUID
func (m *Course) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Chapter) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Lesson) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Sale) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Coupon) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Tag) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
func (m *Category) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
// ... (tambahkan hook serupa untuk Chapter, Lesson, Category, Tag, Sale, Coupon) ...