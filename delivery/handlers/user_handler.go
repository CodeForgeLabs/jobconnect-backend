package handlers

import (
	"encoding/json"
	"job-connect/auth"
	"job-connect/domain"
	usecase "job-connect/usecase"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userUsecase *usecase.UserUsecase
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type CreateUserRequest struct {
	Role      string `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func NewUserHandler(userUsecase *usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT cookie
// @Tags Users
// @Accept json
// @Produce json
// @Param request body handlers.LoginRequest true "Login credentials"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Invalid email or password"
// @Router /users/login [post]
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {

	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.Login(req.Email, req.Password)
	if err != nil {
		println(err)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateToken(user.ID, string(user.Role))
	if err != nil || token == "" {
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Local-friendly cookie configuration
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   5184000, // 2 months in seconds
		HttpOnly: true,    // Still keep this! It protects against JS scripts
		Secure:   true,    // Set to false so it works on http://localhost
		SameSite: http.SameSiteNoneMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"role":    string(user.Role),
	})
}

// CreateUser godoc
// @Summary Register new user
// @Description Create a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body handlers.CreateUserRequest true "User registration data"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "Invalid request payload"
// @Failure 500 {string} string "Failed to create user"
// @Router /users/register [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {

	var req CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	req.Password = hashedPassword
	user := &domain.User{
		Role:      domain.Role(req.Role),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password, // In production, hash this password!
	}

	err = h.userUsecase.CreateUser(user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	// set cookie after successful registration (optional, can also require login after registration)
	token, err := auth.CreateToken(user.ID, string(user.Role))
	// Local-friendly cookie configuration
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   5184000, // 2 months in seconds
		HttpOnly: true,    // Still keep this! It protects against JS scripts
		Secure:   true,    // Set to false so it works on http://localhost
		SameSite: http.SameSiteNoneMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created successfully",
	})
}

// GetUserByID godoc
// @Summary Get current user profile
// @Description Retrieve logged-in user profile
// @Tags Users
// @Produce json
// @Success 200 {object} domain.User
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "User not found"
// @Router /users/me [get]
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}
	// Convert userId from string to uint
	// Import "strconv" at the top of the file if not already imported
	uid, convErr := strconv.ParseUint(userId, 10, 64)
	if convErr != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	user, err := h.userUsecase.GetUserByID(uint(uid))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUserByEmail godoc
// @Summary Get user by email
// @Description Retrieve user profile using email
// @Tags Users
// @Produce json
// @Param email query string true "User email"
// @Success 200 {object} domain.User
// @Failure 400 {string} string "Email required"
// @Failure 404 {string} string "User not found"
// @Router /users/email [get]
func (h *UserHandler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Email query parameter is required", http.StatusBadRequest)
		return
	}
	user, err := h.userUsecase.GetUserByEmail(email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUser godoc
// @Summary Update current user
// @Description Update user profile fields partially
// @Tags Users
// @Accept json
// @Produce json
// @Param request body handlers.UpdateUserRequest true "User update payload"
// @Success 200 {object} domain.User
// @Failure 400 {string} string "Invalid request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Update failed"
// @Router /users/me [patch]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}
	// Convert userId from string to uint
	uid, convErr := strconv.ParseUint(userId, 10, 64)
	if convErr != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	existingUser, err := h.userUsecase.GetUserByID(uint(uid))
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	var payload UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Registration fields
	if payload.Role != nil {
		existingUser.Role = *payload.Role
	}
	if payload.FirstName != nil {
		existingUser.FirstName = *payload.FirstName
	}
	if payload.LastName != nil {
		existingUser.LastName = *payload.LastName
	}
	if payload.Email != nil {
		existingUser.Email = *payload.Email
	}
	if payload.Password != nil {
		// Hash password before saving
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*payload.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		existingUser.Password = string(hashedPassword)
	}

	// Optional onboarding fields
	if payload.Headline != nil {
		existingUser.Headline = *payload.Headline
	}
	if payload.Bio != nil {
		existingUser.Bio = *payload.Bio
	}
	if payload.Skills != nil {
		existingUser.Skills = strings.Join(*payload.Skills, ",")
	}
	if payload.HourlyRate != nil {
		existingUser.HourlyRate = *payload.HourlyRate
	}
	if payload.Availability != nil {
		existingUser.Availability = *payload.Availability
	}
	if payload.Location != nil {
		existingUser.Location = *payload.Location
	}
	if payload.PhoneNumber != nil {
		existingUser.PhoneNumber = *payload.PhoneNumber
	}
	if payload.ProfilePictureURL != nil {
		existingUser.ProfilePictureURL = *payload.ProfilePictureURL
	}

	if err := h.userUsecase.UpdateUser(existingUser); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingUser)
}

// DeleteUser godoc
// @Summary Delete current user
// @Description Delete logged-in user account
// @Tags Users
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Delete failed"
// @Router /users/me [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}
	// Convert userId from string to uint
	uid, convErr := strconv.ParseUint(userId, 10, 64)
	if convErr != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	err = h.userUsecase.DeleteUser(uint(uid))
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User deleted successfully",
	})
}

type UpdateUserRequest struct {
	Role      *domain.Role `json:"role"`
	FirstName *string      `json:"first_name"`
	LastName  *string      `json:"last_name"`
	Email     *string      `json:"email"`
	Password  *string      `json:"password"`

	Headline          *string              `json:"headline"`
	Bio               *string              `json:"bio"`
	Skills            *[]string            `json:"skills"`
	HourlyRate        *float64             `json:"hourly_rate"`
	Availability      *domain.Availability `json:"availability"`
	Location          *string              `json:"location"`
	PhoneNumber       *string              `json:"phone_number"`
	ProfilePictureURL *string              `json:"profile_picture_url"`
}
