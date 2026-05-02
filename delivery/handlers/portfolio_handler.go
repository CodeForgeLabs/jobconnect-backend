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
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type PortfolioHandler struct {
	portfolioUsecase *usecase.PortfolioUsecase
	userUsecase      *usecase.UserUsecase
}

type CreatePortfolioItemRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	TechStack   []string `json:"tech_stack"`
	StartDate   string   `json:"start_date"`
	EndDate     *string  `json:"end_date,omitempty"`
}

type UpdatePortfolioItemRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	TechStack   *[]string `json:"tech_stack,omitempty"`
	StartDate   *string   `json:"start_date,omitempty"`
	EndDate     *string   `json:"end_date,omitempty"`
}

type PortfolioItemResponse struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	TechStack   []string  `json:"tech_stack"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewPortfolioHandler(portfolioUsecase *usecase.PortfolioUsecase, userUsecase *usecase.UserUsecase) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioUsecase: portfolioUsecase,
		userUsecase:      userUsecase,
	}
}

// CreatePortfolioItem godoc
// @Summary Create portfolio item
// @Description Create a portfolio gallery item for the current freelancer
// @Tags Portfolio
// @Accept json
// @Produce json
// @Param request body handlers.CreatePortfolioItemRequest true "Create portfolio item"
// @Success 200 {object} handlers.PortfolioItemResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Router /portfolio [post]
func (h *PortfolioHandler) CreatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	user, err := h.getCurrentFreelancer(r)
	if err != nil {
		writePortfolioAuthError(w, err)
		return
	}

	var req CreatePortfolioItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	imageURL := strings.TrimSpace(req.ImageURL)
	startDate, err := parsePortfolioDate(req.StartDate)
	if err != nil {
		http.Error(w, "start_date must be in YYYY-MM-DD format", http.StatusBadRequest)
		return
	}

	endDate, err := parseOptionalPortfolioDate(req.EndDate)
	if err != nil {
		http.Error(w, "end_date must be in YYYY-MM-DD format", http.StatusBadRequest)
		return
	}

	if title == "" || description == "" || imageURL == "" {
		http.Error(w, "title, description, image_url, and start_date are required", http.StatusBadRequest)
		return
	}

	if endDate != nil && endDate.Before(startDate) {
		http.Error(w, "end_date cannot be before start_date", http.StatusBadRequest)
		return
	}

	item := &domain.PortfolioItem{
		UserID:      user.ID,
		Title:       title,
		Description: description,
		ImageURL:    imageURL,
		TechStack:   joinTechStack(req.TechStack),
		StartDate:   startDate,
		EndDate:     endDate,
	}

	if err := h.portfolioUsecase.CreatePortfolioItem(item); err != nil {
		http.Error(w, "failed to create portfolio item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "portfolio item created",
		"item":    newPortfolioItemResponse(item),
	})
}

// GetPortfolioByUserID godoc
// @Summary Get public portfolio gallery
// @Description Retrieve portfolio gallery items for a freelancer
// @Tags Portfolio
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {array} handlers.PortfolioItemResponse
// @Failure 404 {string} string "User not found"
// @Router /users/{id}/portfolio [get]
func (h *PortfolioHandler) GetPortfolioByUserID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || id <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.GetUserByID(uint(id))
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.Role != domain.RoleFreelancer {
		http.Error(w, "portfolio not found", http.StatusNotFound)
		return
	}

	items, err := h.portfolioUsecase.ListPortfolioByUserID(user.ID)
	if err != nil {
		http.Error(w, "failed to fetch portfolio", http.StatusInternalServerError)
		return
	}

	responses := make([]PortfolioItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newPortfolioItemResponse(item))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portfolio": responses,
	})
}

// UpdatePortfolioItem godoc
// @Summary Update portfolio item
// @Description Update a portfolio gallery item owned by the current freelancer
// @Tags Portfolio
// @Accept json
// @Produce json
// @Param id path int true "Portfolio Item ID"
// @Param request body handlers.UpdatePortfolioItemRequest true "Update portfolio item"
// @Success 200 {object} handlers.PortfolioItemResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Portfolio item not found"
// @Router /portfolio/{id} [patch]
func (h *PortfolioHandler) UpdatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	user, err := h.getCurrentFreelancer(r)
	if err != nil {
		writePortfolioAuthError(w, err)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || id <= 0 {
		http.Error(w, "invalid portfolio item id", http.StatusBadRequest)
		return
	}

	item, err := h.portfolioUsecase.GetPortfolioItemByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "portfolio item not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch portfolio item", http.StatusInternalServerError)
		return
	}

	if item.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req UpdatePortfolioItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			http.Error(w, "title cannot be empty", http.StatusBadRequest)
			return
		}
		item.Title = title
	}

	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if description == "" {
			http.Error(w, "description cannot be empty", http.StatusBadRequest)
			return
		}
		item.Description = description
	}

	if req.ImageURL != nil {
		imageURL := strings.TrimSpace(*req.ImageURL)
		if imageURL == "" {
			http.Error(w, "image_url cannot be empty", http.StatusBadRequest)
			return
		}
		item.ImageURL = imageURL
	}

	if req.TechStack != nil {
		item.TechStack = joinTechStack(*req.TechStack)
	}

	if req.StartDate != nil {
		startDate, err := parsePortfolioDate(*req.StartDate)
		if err != nil {
			http.Error(w, "start_date must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}
		item.StartDate = startDate
	}

	if req.EndDate != nil {
		endDate, err := parseOptionalPortfolioDate(req.EndDate)
		if err != nil {
			http.Error(w, "end_date must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}
		item.EndDate = endDate
	}

	if item.EndDate != nil && item.EndDate.Before(item.StartDate) {
		http.Error(w, "end_date cannot be before start_date", http.StatusBadRequest)
		return
	}

	if err := h.portfolioUsecase.UpdatePortfolioItem(item); err != nil {
		http.Error(w, "failed to update portfolio item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "portfolio item updated",
		"item":    newPortfolioItemResponse(item),
	})
}

// DeletePortfolioItem godoc
// @Summary Delete portfolio item
// @Description Delete a portfolio gallery item owned by the current freelancer
// @Tags Portfolio
// @Produce json
// @Param id path int true "Portfolio Item ID"
// @Success 200 {object} map[string]string
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Portfolio item not found"
// @Router /portfolio/{id} [delete]
func (h *PortfolioHandler) DeletePortfolioItem(w http.ResponseWriter, r *http.Request) {
	user, err := h.getCurrentFreelancer(r)
	if err != nil {
		writePortfolioAuthError(w, err)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || id <= 0 {
		http.Error(w, "invalid portfolio item id", http.StatusBadRequest)
		return
	}

	item, err := h.portfolioUsecase.GetPortfolioItemByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "portfolio item not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch portfolio item", http.StatusInternalServerError)
		return
	}

	if item.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.portfolioUsecase.DeletePortfolioItem(item.ID); err != nil {
		http.Error(w, "failed to delete portfolio item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "portfolio item deleted",
	})
}

func (h *PortfolioHandler) getCurrentFreelancer(r *http.Request) (*domain.User, error) {
	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, err
	}

	user, err := h.userUsecase.GetUserByID(uint(uid))
	if err != nil {
		return nil, err
	}

	if user.Role != domain.RoleFreelancer {
		return nil, errPortfolioForbidden
	}

	return user, nil
}

func newPortfolioItemResponse(item *domain.PortfolioItem) PortfolioItemResponse {
	var endDate *string
	if item.EndDate != nil {
		formatted := formatPortfolioDate(*item.EndDate)
		endDate = &formatted
	}

	return PortfolioItemResponse{
		ID:          item.ID,
		UserID:      item.UserID,
		Title:       item.Title,
		Description: item.Description,
		ImageURL:    item.ImageURL,
		TechStack:   splitTechStack(item.TechStack),
		StartDate:   formatPortfolioDate(item.StartDate),
		EndDate:     endDate,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func joinTechStack(values []string) string {
	if len(values) == 0 {
		return ""
	}

	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	return strings.Join(cleaned, ",")
}

func splitTechStack(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	return cleaned
}

func parsePortfolioDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func parseOptionalPortfolioDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := parsePortfolioDate(trimmed)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func formatPortfolioDate(value time.Time) string {
	return value.Format("2006-01-02")
}

var errPortfolioForbidden = errors.New("portfolio forbidden")

func writePortfolioAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errPortfolioForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
	}
}
