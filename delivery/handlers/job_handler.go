package handlers

import (
	"encoding/json"
	"job-connect/auth"
	"job-connect/domain"
	usecase "job-connect/usecase"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type CreateJobRequest struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	JobType         string `json:"job_type"`
	ExperienceLevel string `json:"experience_level"`
	WorkMode        string `json:"work_mode"`

	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	MaxWeeklyHours *int     `json:"max_weekly_hours,omitempty"`
	Budget         *float64 `json:"budget,omitempty"`

	Location    string             `json:"location,omitempty"`
	CompanyName string             `json:"company_name,omitempty"`
	IsPrivate   bool               `json:"is_private"`
	Milestones  []MilestoneRequest `json:"milestones,omitempty"`

	Skills []string `json:"skills"`
}

type MilestoneRequest struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type UpdateJobRequest struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	Category        *string `json:"category,omitempty"`
	JobType         *string `json:"job_type,omitempty"`
	ExperienceLevel *string `json:"experience_level,omitempty"`
	WorkMode        *string `json:"work_mode,omitempty"`

	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	MaxWeeklyHours *int     `json:"max_weekly_hours,omitempty"`
	Budget         *float64 `json:"budget,omitempty"`

	Location    *string   `json:"location,omitempty"`
	CompanyName *string   `json:"company_name,omitempty"`
	IsPrivate   *bool     `json:"is_private,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Skills      *[]string `json:"skills,omitempty"`
}
type JobHandler struct {
	jobUsecase *usecase.JobUsecase
}

func NewJobHandler(jobUsecase *usecase.JobUsecase) *JobHandler {
	return &JobHandler{jobUsecase: jobUsecase}
}

// CreateJob godoc
// @Summary Create job
// @Tags Jobs
// @Accept json
// @Produce json
// @Param request body handlers.CreateJobRequest true "Create Job"
// @Success 200 {object} domain.Job
// @Router /jobs [post]
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	jobType := domain.ParseJobType(req.JobType)

	// 🚨 VALIDATION: FIXED JOB MUST HAVE MILESTONES
	if jobType == domain.JobTypeFixed && len(req.Milestones) == 0 {
		http.Error(w, "fixed jobs require milestones", http.StatusBadRequest)
		return
	}

	job := &domain.Job{
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		JobType:         jobType,
		ExperienceLevel: domain.ParseExperienceLevel(req.ExperienceLevel),
		WorkMode:        domain.ParseWorkMode(req.WorkMode),
		HourlyRate:      req.HourlyRate,
		MaxWeeklyHours:  req.MaxWeeklyHours,
		Budget:          req.Budget,
		Location:        req.Location,
		CompanyName:     req.CompanyName,
		IsPrivate:       req.IsPrivate,
		CreatedBy:       parseUint(userID),
		Skills:          strings.Join(req.Skills, ","),
	}

	// ⭐ attach milestones
	if jobType == domain.JobTypeFixed {
		for _, m := range req.Milestones {
			job.Milestones = append(job.Milestones, domain.Milestone{
				Description: m.Description,
				Amount:      m.Amount,
			})
		}
	}

	if err := h.jobUsecase.CreateJob(job); err != nil {
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "job created",
		"job":     job,
	})
}

// GetJobByID godoc
// @Summary Get job by ID
// @Tags Jobs
// @Produce json
// @Param id path int true "Job ID"
// @Success 200 {object} domain.Job
// @Router /jobs/{id} [get]
func (h *JobHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	job, err := h.jobUsecase.GetJobByID(uint(id))
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job": job,
	})
}

// UpdateJob godoc
// @Summary Update job
// @Tags Jobs
// @Accept json
// @Produce json
// @Param id path int true "Job ID"
// @Param request body handlers.UpdateJobRequest true "Update Job"
// @Success 200 {object} domain.Job
// @Router /jobs/{id} [patch]
func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	job, err := h.jobUsecase.GetJobByID(uint(id))
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	var req UpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// partial updates
	if req.Title != nil {
		job.Title = *req.Title
	}
	if req.Description != nil {
		job.Description = *req.Description
	}
	if req.Category != nil {
		job.Category = *req.Category
	}
	if req.JobType != nil {
		job.JobType = domain.JobType(*req.JobType)
	}
	if req.ExperienceLevel != nil {
		job.ExperienceLevel = domain.ExperienceLevel(*req.ExperienceLevel)
	}
	if req.WorkMode != nil {
		job.WorkMode = domain.WorkMode(*req.WorkMode)
	}
	if req.HourlyRate != nil {
		job.HourlyRate = req.HourlyRate
	}
	if req.MaxWeeklyHours != nil {
		job.MaxWeeklyHours = req.MaxWeeklyHours
	}
	if req.Budget != nil {
		job.Budget = req.Budget
	}
	if req.Location != nil {
		job.Location = *req.Location
	}
	if req.CompanyName != nil {
		job.CompanyName = *req.CompanyName
	}
	if req.IsPrivate != nil {
		job.IsPrivate = *req.IsPrivate
	}
	if req.Skills != nil {
		job.Skills = strings.Join(*req.Skills, ",")
	}
	if req.Status != nil {
		job.Status = domain.JobStatus(*req.Status)
	}
	if err := h.jobUsecase.UpdateJob(job); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "job updated",
		"job":     job,
	})
}

// DeleteJob godoc
// @Summary Delete job
// @Tags Jobs
// @Param id path int true "Job ID"
// @Success 200 {object} map[string]string
// @Router /jobs/{id} [delete]
func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	err := h.jobUsecase.DeleteJob(uint(id))
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "job deleted",
	})
}

// ListJobs godoc
// @Summary List jobs with filters
// @Tags Jobs
// @Produce json
// @Param title query string false "Title"
// @Param category query string false "Category"
// @Param job_type query string false "Job Type"
// @Param work_mode query string false "Work Mode"
// @Param experience_level query string false "Experience Level"
// @Param budget_min query number false "Minimum Budget"
// @Success 200 {array} domain.Job
// @Router /jobs [get]
func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	filter := domain.JobFilter{
		Title:    query.Get("title"),
		Company:  query.Get("company"),
		Location: query.Get("location"),
		Category: query.Get("category"),

		ExperienceLevel: domain.ParseExperienceLevel(query.Get("experience_level")),
		JobType:         domain.ParseJobType(query.Get("job_type")),
		WorkMode:        domain.ParseWorkMode(query.Get("work_mode")),
		Status:          domain.ParseJobStatus(query.Get("status")),
	}

	// ======================
	// ENUM FILTERS
	// ======================
	if jt := query.Get("job_type"); jt != "" {
		filter.JobType = domain.JobType(jt)
	}

	if el := query.Get("experience_level"); el != "" {
		filter.ExperienceLevel = domain.ExperienceLevel(el)
	}

	if wm := query.Get("work_mode"); wm != "" {
		filter.WorkMode = domain.WorkMode(wm)
	}

	if st := query.Get("status"); st != "" {
		filter.Status = domain.JobStatus(st)
	}

	// ======================
	// SKILLS (comma-separated)
	// ======================
	if skills := query.Get("skills"); skills != "" {
		filter.Skills = strings.Split(skills, ",")
	}

	// ======================
	// NUMERIC FILTERS
	// ======================
	if budgetMin := query.Get("budget_min"); budgetMin != "" {
		if v, err := strconv.ParseFloat(budgetMin, 64); err == nil {
			filter.BudgetMin = &v
		}
	}

	if hourlyMin := query.Get("hourly_rate_min"); hourlyMin != "" {
		if v, err := strconv.ParseFloat(hourlyMin, 64); err == nil {
			filter.HourlyRateMin = &v
		}
	}

	// ======================
	// CALL USECASE
	// ======================
	jobs, err := h.jobUsecase.ListJobs(filter)
	if err != nil {
		http.Error(w, "failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}

func parseUint(s string) uint {
	v, _ := strconv.Atoi(s)
	return uint(v)
}
