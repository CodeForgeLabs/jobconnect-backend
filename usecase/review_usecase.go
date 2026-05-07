package usecase

import "job-connect/domain"

type ReviewUsecase struct {
	reviewRepo domain.ReviewRepository
}

func NewReviewUsecase(reviewRepo domain.ReviewRepository) *ReviewUsecase {
	return &ReviewUsecase{reviewRepo: reviewRepo}
}

func (u *ReviewUsecase) CreateReview(contractID, clientID uint, rating int, note string) (*domain.Review, error) {
	return u.reviewRepo.CreateReview(contractID, clientID, rating, note)
}

func (u *ReviewUsecase) GetReviewByID(id uint) (*domain.Review, error) {
	return u.reviewRepo.GetReviewByID(id)
}

func (u *ReviewUsecase) UpdateReview(reviewID, clientID uint, rating *int, note *string) (*domain.Review, error) {
	return u.reviewRepo.UpdateReview(reviewID, clientID, rating, note)
}

func (u *ReviewUsecase) UpdateFreelancerReply(reviewID, freelancerID uint, reply *string) (*domain.Review, error) {
	return u.reviewRepo.UpdateFreelancerReply(reviewID, freelancerID, reply)
}

func (u *ReviewUsecase) ListReviewsByFreelancerID(freelancerID uint) (*domain.ReviewListResult, error) {
	return u.reviewRepo.ListReviewsByFreelancerID(freelancerID)
}
