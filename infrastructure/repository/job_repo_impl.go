package repository

import (
	"job-connect/domain"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) CreateJob(job *domain.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetJobByID(id uint) (*domain.Job, error) {
	var job domain.Job
	err := r.db.First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) UpdateJob(job *domain.Job) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) DeleteJob(id uint) error {
	return r.db.Delete(&domain.Job{}, id).Error
}

func (r *JobRepository) ListJobs(filter domain.JobFilter) ([]*domain.Job, error) {
	var jobs []*domain.Job

	query := r.db.Model(&domain.Job{})

	// ======================
	// TEXT SEARCH FILTERS
	// ======================
	if filter.Title != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Title+"%")
	}

	if filter.Company != "" {
		query = query.Where("company_name ILIKE ?", "%"+filter.Company+"%")
	}

	if filter.Location != "" {
		query = query.Where("location ILIKE ?", "%"+filter.Location+"%")
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	// ======================
	// ENUM FILTERS
	// ======================
	if filter.JobType != "" {
		query = query.Where("job_type = ?", filter.JobType)
	}

	if filter.ExperienceLevel != "" {
		query = query.Where("experience_level = ?", filter.ExperienceLevel)
	}

	if filter.WorkMode != "" {
		query = query.Where("work_mode = ?", filter.WorkMode)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// ======================
	// BUDGET FILTERS
	// ======================
	if filter.BudgetMin != nil {
		query = query.Where("budget >= ?", *filter.BudgetMin)
	}

	if filter.HourlyRateMin != nil {
		query = query.Where("hourly_rate >= ?", *filter.HourlyRateMin)
	}

	// ======================
	// SKILLS FILTER (IMPORTANT)
	// ======================
	if len(filter.Skills) > 0 {
		for _, skill := range filter.Skills {
			query = query.Where("skills ILIKE ?", "%"+skill+"%")
		}
	}
	query = query.Where("status = ?", domain.StatusOpen).Order("created_at DESC")
	// EXECUTE
	// ======================
	err := query.Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}
