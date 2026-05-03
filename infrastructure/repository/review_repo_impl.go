package repository

import (
	"errors"
	"job-connect/domain"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) CreateReview(contractID, clientID uint, rating int, note string) (*domain.Review, error) {
	var created *domain.Review

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var contract domain.Contract
		if err := tx.First(&contract, contractID).Error; err != nil {
			return err
		}

		if contract.ClientID != clientID {
			return domain.ErrForbidden
		}

		if contract.Status != domain.ContractCompleted {
			return domain.ErrInvalidState
		}

		var existing domain.Review
		err := tx.Where("contract_id = ?", contractID).First(&existing).Error
		if err == nil {
			return domain.ErrConflict
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		review := &domain.Review{
			ContractID:   contract.ID,
			ClientID:     contract.ClientID,
			FreelancerID: contract.FreelancerID,
			Rating:       rating,
			Note:         note,
		}

		if err := tx.Create(review).Error; err != nil {
			return err
		}

		created = review
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *ReviewRepository) GetReviewByID(id uint) (*domain.Review, error) {
	var review domain.Review
	if err := r.db.First(&review, id).Error; err != nil {
		return nil, err
	}

	return &review, nil
}

func (r *ReviewRepository) UpdateReview(reviewID, clientID uint, rating *int, note *string) (*domain.Review, error) {
	var review domain.Review
	if err := r.db.First(&review, reviewID).Error; err != nil {
		return nil, err
	}

	if review.ClientID != clientID {
		return nil, domain.ErrForbidden
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if rating != nil {
		updates["rating"] = *rating
		review.Rating = *rating
	}

	if note != nil {
		trimmed := strings.TrimSpace(*note)
		updates["note"] = trimmed
		review.Note = trimmed
	}

	if err := r.db.Model(&review).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &review, nil
}

func (r *ReviewRepository) UpdateFreelancerReply(reviewID, freelancerID uint, reply *string) (*domain.Review, error) {
	var review domain.Review
	if err := r.db.First(&review, reviewID).Error; err != nil {
		return nil, err
	}

	if review.FreelancerID != freelancerID {
		return nil, domain.ErrForbidden
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if reply == nil || strings.TrimSpace(*reply) == "" {
		updates["freelancer_reply"] = nil
		review.FreelancerReply = nil
	} else {
		trimmed := strings.TrimSpace(*reply)
		updates["freelancer_reply"] = trimmed
		review.FreelancerReply = &trimmed
	}

	if err := r.db.Model(&review).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &review, nil
}

func (r *ReviewRepository) ListReviewsByFreelancerID(freelancerID uint) (*domain.ReviewListResult, error) {
	var reviews []*domain.Review
	if err := r.db.Where("freelancer_id = ?", freelancerID).Order("created_at DESC").Find(&reviews).Error; err != nil {
		return nil, err
	}

	var aggregate struct {
		AverageRating float64
		ReviewCount   int64
	}
	if err := r.db.Model(&domain.Review{}).
		Select("COALESCE(AVG(rating), 0) AS average_rating, COUNT(*) AS review_count").
		Where("freelancer_id = ?", freelancerID).
		Scan(&aggregate).Error; err != nil {
		return nil, err
	}

	return &domain.ReviewListResult{
		AverageRating: aggregate.AverageRating,
		ReviewCount:   aggregate.ReviewCount,
		Reviews:       reviews,
	}, nil
}
