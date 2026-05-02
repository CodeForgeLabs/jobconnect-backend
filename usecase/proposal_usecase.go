package usecase

import "job-connect/domain"

type ProposalUsecase struct {
	propRepo domain.ProposalRepository
}

func NewProposalUsecase(propRepo domain.ProposalRepository) *ProposalUsecase {
	return &ProposalUsecase{propRepo: propRepo}
}

func (u *ProposalUsecase) CreateProposal(proposal *domain.Proposal) error {
	return u.propRepo.CreateProposal(proposal)
}

func (u *ProposalUsecase) GetProposalByID(id uint) (*domain.Proposal, error) {
	return u.propRepo.GetProposalByID(id)
}

func (u *ProposalUsecase) ListProposalsByJobID(jobID uint) ([]*domain.ProposalWithUserResponse, error) {
	return u.propRepo.ListProposalsByJobID(jobID)
}

func (u *ProposalUsecase) UpdateProposal(proposal *domain.Proposal) error {
	return u.propRepo.UpdateProposal(proposal)
}

func (u *ProposalUsecase) DeleteProposal(id uint) error {
	return u.propRepo.DeleteProposal(id)
}

func (u *ProposalUsecase) ListMyProposals(userID uint) ([]*domain.Proposal, error) {
	return u.propRepo.ListMyProposals(userID)
}
