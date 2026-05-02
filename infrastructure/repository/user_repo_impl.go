package repository

import (
	"job-connect/auth"
	"job-connect/domain"

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
