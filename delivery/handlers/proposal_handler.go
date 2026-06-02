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

type ProposalHandler struct {
	propUsecase *usecase.ProposalUsecase
}

func NewProposalHandler(propUsecase *usecase.ProposalUsecase) *ProposalHandler {
	return &ProposalHandler{propUsecase: propUsecase}
}

type CreateProposalRequest struct {
	JobID       uint   `json:"job_id"`
	CoverLetter string `json:"cover_letter"`
}

// DTOs used for Swagger documentation
type ProposalResponseDTO struct {
	ID          uint   `json:"id" example:"1"`
	JobID       uint   `json:"job_id" example:"2"`
	SenderID    uint   `json:"sender_id" example:"3"`
	Description string `json:"description" example:"I would like to work on this project."`
	Status      string `json:"status" example:"pending"`
}

type CreateProposalResponse struct {
	Message  string              `json:"message" example:"proposal created"`
	Proposal ProposalResponseDTO `json:"proposal"`
}

type ListProposalsResponse struct {
	Proposals []ProposalResponseDTO `json:"proposals"`
}

type GetProposalResponse struct {
	Proposal ProposalResponseDTO `json:"proposal"`
}

type GenericMessageResponse struct {
	Message string `json:"message" example:"proposal deleted"`
}
type UpdateProposalRequest struct {
	CoverLetter *string `json:"cover_letter,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// CreateProposal godoc
// @Summary Create a proposal
// @Description Create a new proposal for a job
// @Tags Proposals
// @Accept json
// @Produce json
// @Param payload body CreateProposalRequest true "Create Proposal Request"
// @Success 201 {object} CreateProposalResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /proposals [post]
func (h *ProposalHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	userID, userRole, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if userRole != string(domain.RoleFreelancer) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req CreateProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	proposal := &domain.Proposal{
		JobID:       req.JobID,
		SenderID:    parseUint(userID),
		Description: req.CoverLetter,
	}

	if err := h.propUsecase.CreateProposal(proposal); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "proposal created",
		"proposal": proposal,
	})
}

// GetProposalByID godoc
// @Summary Get a proposal by ID
// @Description Retrieve a proposal using its ID
// @Tags Proposals
// @Accept json
// @Produce json
// @Param id path int true "Proposal ID"
// @Success 200 {object} GetProposalResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 404 {object} GenericMessageResponse
// @Router /proposals/{id} [get]
func (h *ProposalHandler) GetProposalByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	idUint, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid proposal ID", http.StatusBadRequest)
		return
	}
	proposal, err := h.propUsecase.GetProposalByID(uint(idUint))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"proposal": proposal,
	})
}

type ListProposalsByJobRequest struct {
	JobID uint `json:"job_id"`
}

// ListProposalsByJobID godoc
// @Summary List proposals for a job
// @Description List all proposals submitted for a specific job
// @Tags Proposals
// @Accept json
// @Produce json
// @Param request body handlers.ListProposalsByJobRequest true "Job ID"
// @Success 200 {object} domain.ProposalWithUserResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /proposals/jobs [post]
func (h *ProposalHandler) ListProposalsByJobID(w http.ResponseWriter, r *http.Request) {
	var req ListProposalsByJobRequest
	println("am i here")
	// decode body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	println("am i here")
	// validate
	if req.JobID == 0 {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// fetch proposals
	proposals, err := h.propUsecase.ListProposalsByJobID(req.JobID)
	if err != nil {
		http.Error(w, "failed to list proposals", http.StatusInternalServerError)
		return
	}

	// response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":    req.JobID,
		"proposals": proposals,
	})
}

// UpdateProposal godoc
// @Summary Update a proposal
// @Description Update proposal fields such as cover letter or status
// @Tags Proposals
// @Accept json
// @Produce json
// @Param id path int true "Proposal ID"
// @Param payload body UpdateProposalRequest true "Update Proposal Request"
// @Success 200 {object} CreateProposalResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 404 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /proposals/{id} [patch]
func (h *ProposalHandler) UpdateProposal(w http.ResponseWriter, r *http.Request) {
	_, role, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if role != string(domain.RoleClient) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := mux.Vars(r)["id"]
	idUint, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid proposal ID", http.StatusBadRequest)
		return
	}

	proposal, err := h.propUsecase.GetProposalByID(uint(idUint))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var req UpdateProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.CoverLetter != nil {
		proposal.Description = *req.CoverLetter
	}
	if req.Status != nil {
		proposal.Status = domain.ProposalStatus(*req.Status)
	}

	if err := h.propUsecase.UpdateProposal(proposal); err != nil {
		http.Error(w, "failed to update proposal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "proposal updated",
		"proposal": proposal,
	})
}

// DeleteProposal godoc
// @Summary Delete a proposal
// @Description Delete a proposal by ID
// @Tags Proposals
// @Accept json
// @Produce json
// @Param id path int true "Proposal ID"
// @Success 200 {object} GenericMessageResponse
// @Failure 400 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /proposals/{id} [delete]
func (h *ProposalHandler) DeleteProposal(w http.ResponseWriter, r *http.Request) {
	_, role, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if role != string(domain.RoleClient) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := mux.Vars(r)["id"]
	idUint, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid proposal ID", http.StatusBadRequest)
		return
	}

	if err := h.propUsecase.DeleteProposal(uint(idUint)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "proposal deleted",
	})
}

// ListMyProposals godoc
// @Summary List my proposals
// @Description List all proposals submitted by the authenticated user
// @Tags Proposals
// @Accept json
// @Produce json
// @Success 200 {object} ListProposalsResponse
// @Failure 401 {object} GenericMessageResponse
// @Failure 500 {object} GenericMessageResponse
// @Router /proposals/mine [get]
func (h *ProposalHandler) ListMyProposals(w http.ResponseWriter, r *http.Request) {
	userID, _, err := auth.GetUserFromToken(r)
	println(userID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	proposals, err := h.propUsecase.ListMyProposals(parseUint(userID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"proposals": proposals,
	})
}
