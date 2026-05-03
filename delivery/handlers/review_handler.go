package handlers

import (
	"encoding/json"
	"errors"
	"job-connect/auth"
	"job-connect/domain"
	usecase "job-connect/usecase"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type ReviewHandler struct {
	reviewUsecase *usecase.ReviewUsecase
	userUsecase   *usecase.UserUsecase
}

type CreateReviewRequest struct {
	ContractID uint   `json:"contract_id"`
	Rating     int    `json:"rating"`
	Note       string `json:"note"`
}

type UpdateReviewRequest struct {
	Rating *int    `json:"rating,omitempty"`
	Note   *string `json:"note,omitempty"`
}

type UpdateReviewReplyRequest struct {
	Reply *string `json:"reply"`
}

type ReviewResponse struct {
	ID              uint    `json:"id"`
	ContractID      uint    `json:"contract_id"`
	ClientID        uint    `json:"client_id"`
	FreelancerID    uint    `json:"freelancer_id"`
	Rating          int     `json:"rating"`
	Note            string  `json:"note"`
	FreelancerReply *string `json:"freelancer_reply,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type ListReviewsResponse struct {
	AverageRating float64          `json:"average_rating"`
	ReviewCount   int64            `json:"review_count"`
	Reviews       []ReviewResponse `json:"reviews"`
}

func NewReviewHandler(reviewUsecase *usecase.ReviewUsecase, userUsecase *usecase.UserUsecase) *ReviewHandler {
	return &ReviewHandler{
		reviewUsecase: reviewUsecase,
		userUsecase:   userUsecase,
	}
}

// CreateReview godoc
// @Summary Create freelancer review
// @Description Create a client review for a completed contract
// @Tags Reviews
// @Accept json
// @Produce json
// @Param request body handlers.CreateReviewRequest true "Create review request"
// @Success 201 {object} handlers.ReviewResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Contract not found"
// @Failure 409 {string} string "Review already exists"
// @Router /reviews [post]
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	clientID, err := currentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	note := strings.TrimSpace(req.Note)
	if req.ContractID == 0 || note == "" {
		http.Error(w, "contract_id, rating, and note are required", http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	review, err := h.reviewUsecase.CreateReview(req.ContractID, clientID, req.Rating, note)
	if err != nil {
		writeReviewError(w, err, "contract")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "review created",
		"review":  newReviewResponse(review),
	})
}

// UpdateReview godoc
// @Summary Update freelancer review
// @Description Update a client review for a freelancer
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path int true "Review ID"
// @Param request body handlers.UpdateReviewRequest true "Update review request"
// @Success 200 {object} handlers.ReviewResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Review not found"
// @Router /reviews/{id} [patch]
func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	clientID, err := currentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}

	reviewID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || reviewID <= 0 {
		http.Error(w, "invalid review id", http.StatusBadRequest)
		return
	}

	var req UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Rating == nil && req.Note == nil {
		http.Error(w, "at least one field is required", http.StatusBadRequest)
		return
	}

	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		http.Error(w, "rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	if req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		if trimmed == "" {
			http.Error(w, "note cannot be empty", http.StatusBadRequest)
			return
		}
		req.Note = &trimmed
	}

	review, err := h.reviewUsecase.UpdateReview(uint(reviewID), clientID, req.Rating, req.Note)
	if err != nil {
		writeReviewError(w, err, "review")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "review updated",
		"review":  newReviewResponse(review),
	})
}

// UpdateReviewReply godoc
// @Summary Update freelancer review reply
// @Description Add or update the reviewed freelancer's public reply
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path int true "Review ID"
// @Param request body handlers.UpdateReviewReplyRequest true "Update reply request"
// @Success 200 {object} handlers.ReviewResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Review not found"
// @Router /reviews/{id}/reply [patch]
func (h *ReviewHandler) UpdateReviewReply(w http.ResponseWriter, r *http.Request) {
	freelancerID, err := currentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}

	reviewID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || reviewID <= 0 {
		http.Error(w, "invalid review id", http.StatusBadRequest)
		return
	}

	var req UpdateReviewReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Reply == nil {
		http.Error(w, "reply is required", http.StatusBadRequest)
		return
	}

	review, err := h.reviewUsecase.UpdateFreelancerReply(uint(reviewID), freelancerID, req.Reply)
	if err != nil {
		writeReviewError(w, err, "review")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "review reply updated",
		"review":  newReviewResponse(review),
	})
}

// ListReviewsByFreelancerID godoc
// @Summary Get public freelancer reviews
// @Description Retrieve public reviews for a freelancer
// @Tags Reviews
// @Produce json
// @Param id path int true "Freelancer User ID"
// @Success 200 {object} handlers.ListReviewsResponse
// @Failure 404 {string} string "User not found"
// @Router /users/{id}/reviews [get]
func (h *ReviewHandler) ListReviewsByFreelancerID(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.GetUserByID(uint(userID))
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.Role != domain.RoleFreelancer {
		http.Error(w, "reviews not found", http.StatusNotFound)
		return
	}

	result, err := h.reviewUsecase.ListReviewsByFreelancerID(user.ID)
	if err != nil {
		http.Error(w, "failed to fetch reviews", http.StatusInternalServerError)
		return
	}

	reviews := make([]ReviewResponse, 0, len(result.Reviews))
	for _, review := range result.Reviews {
		reviews = append(reviews, newReviewResponse(review))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListReviewsResponse{
		AverageRating: result.AverageRating,
		ReviewCount:   result.ReviewCount,
		Reviews:       reviews,
	})
}

func newReviewResponse(review *domain.Review) ReviewResponse {
	return ReviewResponse{
		ID:              review.ID,
		ContractID:      review.ContractID,
		ClientID:        review.ClientID,
		FreelancerID:    review.FreelancerID,
		Rating:          review.Rating,
		Note:            review.Note,
		FreelancerReply: review.FreelancerReply,
		CreatedAt:       review.CreatedAt.Format(timeLayout),
		UpdatedAt:       review.UpdatedAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func currentUserID(r *http.Request) (uint, error) {
	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(parsed), nil
}

func writeReviewError(w http.ResponseWriter, err error, resourceName string) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, resourceName+" not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, domain.ErrConflict):
		http.Error(w, "review already exists", http.StatusConflict)
	case errors.Is(err, domain.ErrInvalidState):
		http.Error(w, "contract must be completed before review", http.StatusBadRequest)
	default:
		http.Error(w, "failed to process "+resourceName, http.StatusInternalServerError)
	}
}
