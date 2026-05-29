package usecase

import "job-connect/domain"

type JobUsecase struct {
	jobRepo domain.JobRepository
}

func NewJobUsecase(jobRepo domain.JobRepository) *JobUsecase {
	return &JobUsecase{jobRepo: jobRepo}
}

func (j *JobUsecase) CreateJob(job *domain.Job) error {
	return j.jobRepo.CreateJob(job)
}

func (j *JobUsecase) GetJobByID(id uint) (*domain.Job, error) {
	return j.jobRepo.GetJobByID(id)
}

func (j *JobUsecase) UpdateJob(job *domain.Job) error {
	return j.jobRepo.UpdateJob(job)
}

func (j *JobUsecase) DeleteJob(id uint) error {
	return j.jobRepo.DeleteJob(id)
}

func (j *JobUsecase) ListJobs(filter domain.JobFilter) ([]*domain.Job, error) {
	return j.jobRepo.ListJobs(filter)
}

func (j *JobUsecase) ListMyJobs(userID uint) ([]*domain.Job, error) {
	return j.jobRepo.ListMyJobs(userID)
}

func (j *JobUsecase) ListJobByClientId(clientID uint) ([]*domain.Job, error) {
	return j.jobRepo.ListJobByClientId(clientID)
}
