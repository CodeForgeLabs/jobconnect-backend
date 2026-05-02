package usecase

import "job-connect/domain"

type ContractUsecase struct {
	contractRepo domain.ContractRepository
}

func NewContractUsecase(contractRepo domain.ContractRepository) *ContractUsecase {
	return &ContractUsecase{contractRepo: contractRepo}
}

func (u *ContractUsecase) CreateContract(jobId, freelancerId string) error {
	return u.contractRepo.CreateContract(jobId, freelancerId)
}

func (u *ContractUsecase) GetContractByID(id uint) (*domain.MyContractResponse, error) {
	return u.contractRepo.GetContractByID(id)
}
func (u *ContractUsecase) GetMyContracts(userID uint) ([]*domain.MyContractResponse, error) {
	return u.contractRepo.GetMyContracts(userID)
}

func (u *ContractUsecase) SubmitMilestone(request *domain.SubmitMilestoneRequest) error {
	return u.contractRepo.SubmitMilestone(request)
}

func (u *ContractUsecase) ModifyMilestoneStatus(milestoneId uint, newStatus domain.ContractMilestoneStatus) error {
	return u.contractRepo.ModifyStatus(milestoneId, newStatus)
}

func (u *ContractUsecase) ModifyContractStatus(contractId uint, newStatus domain.ContractStatus) error {
	return u.contractRepo.ModifyContractStatus(contractId, newStatus)
}

func (u *ContractUsecase) StartWorkSession(contractId, freelancerId uint) error {
	return u.contractRepo.StartWorkSession(contractId, freelancerId)
}
func (u *ContractUsecase) EndWorkSession(contractId, freelancerId uint) error {
	return u.contractRepo.EndWorkSession(contractId, freelancerId)
}
func (u *ContractUsecase) FetchTimeLogs(contractId, freelancerId uint) ([]*domain.TimeLog, error) {
	return u.contractRepo.FetchTimeLogs(contractId, freelancerId)
}
func (u *ContractUsecase) FetchTimeElapsed(contractId, freelancerId uint) (float64, error) {
	return u.contractRepo.FetchTimeElapsed(contractId, freelancerId)
}
func (u *ContractUsecase) FetchWeeklyHours(contractId, freelancerId uint) (float64, error) {
	return u.contractRepo.FetchWeeklyHours(contractId, freelancerId)
}
