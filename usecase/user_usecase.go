package usecase

import "job-connect/domain"

type UserUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{userRepo: userRepo}
}

func (u *UserUsecase) Login(email, password string) (*domain.User, error) {
	return u.userRepo.Login(email, password)
}

func (u *UserUsecase) CreateUser(user *domain.User) error {
	return u.userRepo.CreateUser(user)
}

func (u *UserUsecase) GetUserByID(id uint) (*domain.User, error) {
	return u.userRepo.GetUserByID(id)
}

func (u *UserUsecase) GetUserByEmail(email string) (*domain.User, error) {
	return u.userRepo.GetUserByEmail(email)
}

func (u *UserUsecase) UpdateUser(user *domain.User) error {
	return u.userRepo.UpdateUser(user)
}

func (u *UserUsecase) DeleteUser(id uint) error {
	return u.userRepo.DeleteUser(id)
}
