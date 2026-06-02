package handlers

import (
	"encoding/json"
	"job-connect/auth"
	"job-connect/domain"
	usecase "job-connect/usecase"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userUsecase *usecase.UserUsecase
}
type RequestForgotPassword struct {
	IsForgotPassword bool `json:"is_forgot_password"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type CreateUserRequest struct {
	Role        string `json:"role"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CompanyName string `json:"company_name,omitempty"`
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
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateToken(user.ID, string(user.Role))
	if err != nil || token == "" {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		Role:        domain.Role(req.Role),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		CompanyName: req.CompanyName,
	}

	err = h.userUsecase.CreateUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// set cookie after successful registration (optional, can also require login after registration)
	token, err := auth.CreateToken(user.ID, string(user.Role))
	// Local-friendly cookie configuration
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   5184000,
		HttpOnly: true, // Still keep this! It protects against JS scripts
		Secure:   true, // it works on http://localhost
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
	if payload.CompanyName != nil {
		existingUser.CompanyName = *payload.CompanyName
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User deleted successfully",
	})
}

// UserLoggedIn godoc
// @Summary Check if user is logged in
// @Description Returns current user info from JWT cookie
// @Tags Users
// @Produce json
// @Success 200 {object} map[string]string "User is logged in"
// @Failure 401 {string} string "Unauthorized: Please login"
// @Router /users/logged [get]
func (h *UserHandler) UserLoggedIn(w http.ResponseWriter, r *http.Request) {
	userId, role, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User is logged in",
		"user_id": userId,
		"role":    role,
	})
}

// UserLogout godoc
// @Summary Logout user
// @Description Clears authentication cookie and logs user out
// @Tags Users
// @Produce json
// @Success 200 {object} map[string]string "Logged out successfully"
// @Router /users/logout [post]
func (h *UserHandler) UserLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// GetUsersById godoc
// @Summary Get user by ID
// @Description Retrieve user profile using ID
// @Tags Users
// @Produce json
// @Param id query int true "User ID"
// @Success 200 {object} domain.User
// @Failure 400 {string} string "ID required"
// @Failure 404 {string} string "User not found"
// @Router /users/byid [get]
func (h *UserHandler) GetUsersById(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID query parameter is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	user, err := h.userUsecase.GetUsersById(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUsersByName godoc
// @Summary Get users by name
// @Description Retrieve user profiles matching name query
// @Tags Users
// @Produce json
// @Param name query string true "User name"
// @Success 200 {array} domain.User
// @Failure 400 {string} string "Name required"
// @Failure 404 {string} string "No users found"
// @Router /users/search [get]
func (h *UserHandler) GetUsersByName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Name query parameter is required", http.StatusBadRequest)
		return
	}
	users, err := h.userUsecase.GetUsersByName(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetUsers godoc
// @Summary List users with filters
// @Description Retrieve user profiles with optional filters (skills, location, min hourly rate)
// @Tags Users
// @Produce json
// @Param skills query string false "Skills (comma-separated or keyword)"
// @Param location query string false "Location"
// @Param min_hourly_rate query number false "Minimum hourly rate"
// @Success 200 {array} domain.User
// @Failure 404 {string} string "No users found"
// @Router /users/fetch [get]
func (h *UserHandler) GetUserBySkill(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	filter := domain.UserFilter{
		Skills:        query.Get("skills"),
		Location:      query.Get("location"),
		MinHourlyRate: parseFloat(query.Get("min_hourly_rate")),
	}

	users, err := h.userUsecase.GetUserBySkill(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// SendOtp godoc
// @Summary Send OTP to user email
// @Description Generate and send a one-time password (OTP) to the specified email for verification
// @Tags Users
// @Produce json
// @Param email query string true "User email"
// @Param request body handlers.RequestForgotPassword true "Indicates if this is for forgot password flow"
// @Success 200 {object} map[string]string "Otp sent successfully"
// @Failure 400 {string} string "Email query parameter is required"
// @Failure 500 {string} string "Failed to send OTP"
// @Router /users/send-otp [post]
func (h *UserHandler) SendOtp(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email query parameter is required", http.StatusBadRequest)
		return
	}

	// var req RequestForgotPassword
	// err := json.NewDecoder(r.Body).Decode(&req)
	// if err != nil {
	// 	http.Error(w, "Invalid request payload", http.StatusBadRequest)
	// 	return
	// }
	// fmt.Println("********************************")
	// fmt.Println(req.IsForgotPassword)
	// if req.IsForgotPassword == true {
	// 	exists, err := h.userUsecase.CheckUserExists(email)
	// 	if err != nil {
	// 		http.Error(w, "Failed to check user existence", http.StatusInternalServerError)
	// 		return
	// 	}
	// 	if !exists {
	// 		http.Error(w, "No user found with this email", http.StatusBadRequest)
	// 		return
	// 	}
	// }

	err := h.userUsecase.SendOtp(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": "Otp sent successfully",
	})
}

// VerifyOtp godoc
// @Summary Verify OTP for user email
// @Description Verify the provided OTP against the stored value for the specified email
// @Tags Users
// @Produce json
// @Param email query string true "User email"
// @Param otp query string true "One-time password"
// @Success 200 {object} map[string]string "Otp verified successfully"
// @Failure 400 {string} string "Email and OTP query parameters are required"
// @Failure 500 {string} string "Failed to verify OTP"
// @Router /users/verify-otp [post]
func (h *UserHandler) VerifyOtp(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	otp := r.URL.Query().Get("otp")
	if email == "" {
		http.Error(w, "email query parameter is required", http.StatusBadRequest)
		return
	}
	if otp == "" {
		http.Error(w, "otp query parameter is required", http.StatusBadRequest)
		return
	}

	result, err := h.userUsecase.VerifyOtp(email, otp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !result {
		http.Error(w, "Invalid or expired OTP", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": "Otp verified successfully",
	})
}

// ModifyPassword godoc
// @Summary Modify user password
// @Description Update the user's password after verifying OTP
// @Tags Users
// @Produce json
// @Param email query string true "User email"
// @Param new_password query string true "New password"
// @Success 200 {object} map[string]string "Password modified successfully"
// @Failure 400 {string} string "Email and new_password query parameters are required"
// @Failure 500 {string} string "Failed to modify password"
// @Router /users/modify-password [post]
func (h *UserHandler) ModifyPassword(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	newPassword := r.URL.Query().Get("new_password")
	if email == "" {
		http.Error(w, "email query parameter is required", http.StatusBadRequest)
		return
	}
	if newPassword == "" {
		http.Error(w, "new_password query parameter is required", http.StatusBadRequest)
		return
	}

	err := h.userUsecase.ModifyPassword(email, newPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": "Password modified successfully",
	})
}
func parseFloat(value string) float64 {
	if value == "" {
		return 0
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return v
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
	CompanyName       *string              `json:"company_name"`
}
