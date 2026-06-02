package repository

import (
	"job-connect/auth"
	"job-connect/chapa"
	"job-connect/domain"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}
func (r *UserRepository) Login(email, password string) (*domain.User, error) {
	oldUser, err := r.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	matched := auth.CheckPasswordHash(password, oldUser.Password)
	if !matched {
		return nil, gorm.ErrRecordNotFound
	}
	var user domain.User
	err = r.db.Where("email = ? ", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepository) CreateUser(user *domain.User) error {
	if user.Role == domain.RoleFreelancer {
		user.Connect = 50
	}
	return r.db.Create(user).Error
}
func (r *UserRepository) GetUserByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepository) UpdateUser(user *domain.User) error {
	return r.db.Save(user).Error
}
func (r *UserRepository) DeleteUser(id uint) error {
	return r.db.Delete(&domain.User{}, id).Error
}

func (r *UserRepository) GetUsersById(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUsersByName(name string) ([]*domain.User, error) {
	var users []*domain.User

	search := "%" + strings.ToLower(name) + "%"

	err := r.db.
		Where(
			"LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ?",
			search,
			search,
		).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetUserBySkill(filter domain.UserFilter) ([]*domain.User, error) {
	var users []*domain.User

	query := r.db.Model(&domain.User{})

	// 1. Skills filter
	if filter.Skills != "" {
		search := "%" + strings.ToLower(filter.Skills) + "%"
		query = query.Where("LOWER(skills) LIKE ?", search)
	}

	// 2. Location filter
	if filter.Location != "" {
		loc := "%" + strings.ToLower(filter.Location) + "%"
		query = query.Where("LOWER(location) LIKE ?", loc)
	}

	// 3. Min hourly rate filter
	if filter.MinHourlyRate > 0 {
		query = query.Where("hourly_rate >= ?", filter.MinHourlyRate)
	}

	// fetch users
	err := query.Find(&users).Error
	if err != nil {
		return nil, err
	}

	// 🔥 brute force rating calculation per user
	for _, u := range users {
		var stats struct {
			Avg   *float64
			Count int64
		}

		err := r.db.Model(&domain.Review{}).
			Select("AVG(rating) as avg, COUNT(*) as count").
			Where("freelancer_id = ?", u.ID).
			Scan(&stats).Error

		if err != nil {
			return nil, err
		}

		// if no reviews
		if stats.Count == 0 || stats.Avg == nil {
			u.AverageRating = 0
			u.TotalReviews = 0
		} else {
			u.AverageRating = *stats.Avg
			u.TotalReviews = int(stats.Count)
		}
	}

	return users, nil
}

func (r *UserRepository) SendOtp(email string) error {
	// FIRST CHECK IF WE ALREADY HAVE, INSTEAD OF CREATING ONE OVERRIDE IT
	var existingOtp domain.Otp
	err := r.db.Where("email = ?", email).First(&existingOtp).Error
	if err == nil {
		// record exists, update it
		existingOtp.OtpCode = GenerateOtp()
		existingOtp.ExpiresAt = GetOtpExpiryTime()
		err := chapa.NewBrevoEmailService().SendOTP(email, existingOtp.OtpCode)
		if err != nil {
			return err
		}
		return r.db.Save(&existingOtp).Error
	} else if err != gorm.ErrRecordNotFound {
		// some other error
		return err
	}

	// no existing record, create new one
	otp := domain.Otp{
		Email:     email,
		OtpCode:   GenerateOtp(),
		ExpiresAt: GetOtpExpiryTime(),
	}
	err = chapa.NewBrevoEmailService().SendOTP(email, otp.OtpCode)
	if err != nil {
		return err
	}
	return r.db.Create(&otp).Error
}

func (r *UserRepository) VerifyOtp(email, otp string) (bool, error) {
	var record domain.Otp
	err := r.db.Where("email = ? AND otp_code = ?", email, otp).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // OTP not found or incorrect
		}
		return false, err // some other error
	}

	if record.ExpiresAt.Before(time.Now()) {
		return false, nil // OTP expired
	}

	return true, nil // OTP valid
}

func (r *UserRepository) ModifyPassword(email, newPassword string) error {
	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return r.db.Model(&domain.User{}).
		Where("email = ?", email).
		Update("password", hashedPassword).Error
}
func GenerateOtp() string {
	const otpLength = 4
	const charset = "0123456789"
	var otp strings.Builder
	for i := 0; i < otpLength; i++ {
		randomIndex := rand.Intn(len(charset))
		otp.WriteByte(charset[randomIndex])
	}
	return otp.String()
}

func GetOtpExpiryTime() time.Time {
	return time.Now().Add(15 * time.Minute)
}
