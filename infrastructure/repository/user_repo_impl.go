package repository

import (
	"job-connect/auth"
	"job-connect/domain"
	"strings"

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

func (r *UserRepository) GetUserBySkill(skill string) ([]*domain.User, error) {
	var users []*domain.User

	search := "%" + strings.ToLower(skill) + "%"

	err := r.db.
		Where("LOWER(skills) LIKE ?", search).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}
