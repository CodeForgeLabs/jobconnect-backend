package handlers

import (
	"encoding/json"
	"job-connect/auth"
	"job-connect/domain"
	usecase "job-connect/usecase"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ContractHandler struct {
	contractUsecase *usecase.ContractUsecase
}
type CreateContractRequest struct {
	JobID        string `json:"job_id"`
	FreelancerID string `json:"freelancer_id"`
}
type WorkSessionRequest struct {
	ContractID uint `json:"contract_id"`
}

func NewContractHandler(contractUsecase *usecase.ContractUsecase) *ContractHandler {
	return &ContractHandler{contractUsecase: contractUsecase}
}

// CreateContract godoc
// @Summary Create a contract
// @Description Create a new contract for a job and freelancer
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body CreateContractRequest true "Contract creation request"
// @Success 201 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts [post]
func (h *ContractHandler) CreateContract(w http.ResponseWriter, r *http.Request) {
	var req CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.contractUsecase.CreateContract(req.JobID, req.FreelancerID); err != nil {
		http.Error(w, "failed to create contract", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "contract created"})
}

// GetMyContracts godoc
// @Summary Get my contracts
// @Description Get all contracts for the authenticated user
// @Tags Contracts
// @Accept json
// @Produce json
// @Success 200 {array} domain.MyContractResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/mine [get]
func (h *ContractHandler) GetMyContracts(w http.ResponseWriter, r *http.Request) {
	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	println("*******************************")
	print(userID)
	println("*******************************")
	contracts, err := h.contractUsecase.GetMyContracts(parseUint(userID))
	if err != nil {
		http.Error(w, "failed to get contracts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contracts": contracts,
	})
}

// GetContractByID godoc
// @Summary Get contract by ID
// @Description Get a specific contract by its ID
// @Tags Contracts
// @Accept json
// @Produce json
// @Param id path int true "Contract ID"
// @Success 200 {object} domain.MyContractResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 404 {object} GenericMessageResponse
// @Router /contracts/{id} [get]
func (h *ContractHandler) GetContractByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	idUint, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid contract ID", http.StatusBadRequest)
		return
	}

	contract, err := h.contractUsecase.GetContractByID(uint(idUint))
	if err != nil {
		http.Error(w, "contract not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contract": contract,
	})
}

// SubmitMilestone godoc
// @Summary Submit milestone work
// @Description Submit work for a specific milestone in a contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body domain.SubmitMilestoneRequest true "Milestone submission request"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/milestone/submit [post]
func (h *ContractHandler) SubmitMilestone(w http.ResponseWriter, r *http.Request) {
	var req domain.SubmitMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.contractUsecase.SubmitMilestone(&req); err != nil {
		http.Error(w, "failed to submit milestone", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "milestone submitted"})
}

// ModifyMilestoneStatus godoc
// @Summary Modify milestone status
// @Description Update the status of a specific milestone (e.g., approve or reject)
// @Tags Contracts
// @Accept json
// @Produce json
// @Param milestone_id path int true "Milestone ID"
// @Param new_status query string true "New status (approved/rejected)"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/milestone/{milestone_id}/status [patch]
func (h *ContractHandler) ModifyMilestoneStatus(w http.ResponseWriter, r *http.Request) {
	milestoneIDStr := mux.Vars(r)["milestone_id"]
	milestoneID, err := strconv.Atoi(milestoneIDStr)
	if err != nil {
		http.Error(w, "invalid milestone ID", http.StatusBadRequest)
		return
	}

	newStatus := r.URL.Query().Get("new_status")
	if newStatus == "" {
		http.Error(w, "new_status query parameter is required", http.StatusBadRequest)

		return
	}
	if newStatus != string(domain.MilestoneApproved) && newStatus != string(domain.MilestoneRevisionRequested) {
		http.Error(w, "new_status must be 'APPROVED' or 'REVISION_REQUESTED'", http.StatusBadRequest)
		return
	}

	if err := h.contractUsecase.ModifyMilestoneStatus(uint(milestoneID), domain.ContractMilestoneStatus(newStatus)); err != nil {
		http.Error(w, "failed to modify milestone status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "milestone status updated"})

}

// ModifyContractStatus godoc
// @Summary Modify contract status
// @Description Update the status of a contract (e.g., mark as completed or cancelled)
// @Tags Contracts
// @Accept json
// @Produce json
// @Param contract_id path int true "Contract ID"
// @Param new_status query string true "New status (ACTIVE/COMPLETED/CANCELLED)"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/{contract_id}/status [patch]
func (h *ContractHandler) ModifyContractStatus(w http.ResponseWriter, r *http.Request) {
	contractIDStr := mux.Vars(r)["contract_id"]
	contractID, err := strconv.Atoi(contractIDStr)
	if err != nil {
		http.Error(w, "invalid contract ID", http.StatusBadRequest)
		return
	}

	newStatus := r.URL.Query().Get("new_status")
	println(newStatus)
	println(contractID)
	if newStatus == "" {
		http.Error(w, "new_status query parameter is required", http.StatusBadRequest)
		return
	}
	if newStatus != string(domain.ContractActive) && newStatus != string(domain.ContractCompleted) && newStatus != string(domain.ContractCancelled) && newStatus != string(domain.ContractPaused) {
		http.Error(w, "new_status must be 'ACTIVE', 'COMPLETED', 'CANCELLED' or 'PAUSED'", http.StatusBadRequest)
		return
	}

	if err := h.contractUsecase.ModifyContractStatus(uint(contractID), domain.ContractStatus(newStatus)); err != nil {
		http.Error(w, "failed to modify contract status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "contract status updated"})
}

// StartWorkSession godoc
// @Summary Start work session
// @Description Start a work session for an hourly contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body WorkSessionRequest true "Work session request"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/work-session/start [post]
func (h *ContractHandler) StartWorkSession(w http.ResponseWriter, r *http.Request) {

	var req WorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.contractUsecase.StartWorkSession(req.ContractID, parseUint(userID)); err != nil {
		http.Error(w, "failed to start work session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "work session started"})
}

// EndWorkSession godoc
// @Summary End work session
// @Description End a work session for an hourly contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body WorkSessionRequest true "Work session request"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/work-session/end [post]
func (h *ContractHandler) EndWorkSession(w http.ResponseWriter, r *http.Request) {
	var req WorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.contractUsecase.EndWorkSession(req.ContractID, parseUint(userID)); err != nil {
		http.Error(w, "failed to end work session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "work session ended"})
}

// FetchTimeLogs godoc
// @Summary Fetch time logs
// @Description Fetch all time logs for an hourly contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body WorkSessionRequest true "Work session request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} GenericMessageResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/work-session/time-logs [post]
func (h *ContractHandler) FetchTimeLogs(w http.ResponseWriter, r *http.Request) {
	var req WorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	timeLogs, err := h.contractUsecase.FetchTimeLogs(req.ContractID, parseUint(userID))
	if err != nil {
		http.Error(w, "failed to fetch time logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"time_logs": timeLogs,
	})
}

// FetchTimeElapsed godoc
// @Summary Fetch time elapsed
// @Description Fetch total time elapsed for an hourly contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body WorkSessionRequest true "Work session request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} GenericMessageResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/work-session/time-elapsed [post]
func (h *ContractHandler) FetchTimeElapsed(w http.ResponseWriter, r *http.Request) {
	var req WorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	timeElapsed, err := h.contractUsecase.FetchTimeElapsed(req.ContractID, parseUint(userID))
	if err != nil {
		http.Error(w, "failed to fetch time elapsed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"time_elapsed": timeElapsed,
	})
}

// FetchWeeklyHours godoc
// @Summary Fetch weekly hours
// @Description Fetch total hours worked in the current week for an hourly contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param request body WorkSessionRequest true "Work session request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} GenericMessageResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /contracts/work-session/weekly-hours [post]
func (h *ContractHandler) FetchWeeklyHours(w http.ResponseWriter, r *http.Request) {
	var req WorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	weeklyHours, err := h.contractUsecase.FetchWeeklyHours(req.ContractID, parseUint(userID))
	if err != nil {
		http.Error(w, "failed to fetch weekly hours", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"weekly_hours": weeklyHours,
	})
}
