package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

// ---- CronTask DAO ----

type CronTaskDAO struct{ db *gorm.DB }

func NewCronTaskDAO(db *gorm.DB) *CronTaskDAO { return &CronTaskDAO{db: db} }

func (d *CronTaskDAO) Create(t *model.CronTask) error {
	return d.db.Create(t).Error
}

func (d *CronTaskDAO) Update(t *model.CronTask) error {
	return d.db.Omit("created_at").Save(t).Error
}

func (d *CronTaskDAO) Delete(id uint) error {
	return d.db.Delete(&model.CronTask{}, id).Error
}

func (d *CronTaskDAO) FindByID(id uint) (*model.CronTask, error) {
	var t model.CronTask
	err := d.db.First(&t, id).Error
	return &t, err
}

func (d *CronTaskDAO) FindPage(page, pageSize int, keyword string) ([]model.CronTask, int64, error) {
	var list []model.CronTask
	var total int64
	q := d.db.Model(&model.CronTask{})
	if keyword != "" {
		q = q.Where("name LIKE ? OR url LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Select("cron_tasks.*, COALESCE((SELECT status FROM cron_task_executions WHERE task_id = cron_tasks.id ORDER BY id DESC LIMIT 1), -1) AS last_exec_status").
		Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}

func (d *CronTaskDAO) FindEnabled() ([]model.CronTask, error) {
	var list []model.CronTask
	err := d.db.Where("status = 1").Find(&list).Error
	return list, err
}

// ---- CronTaskExecution DAO ----

type CronTaskExecutionDAO struct{ db *gorm.DB }

func NewCronTaskExecutionDAO(db *gorm.DB) *CronTaskExecutionDAO { return &CronTaskExecutionDAO{db: db} }

func (d *CronTaskExecutionDAO) Create(e *model.CronTaskExecution) error {
	return d.db.Create(e).Error
}

func (d *CronTaskExecutionDAO) FindByTaskIDPage(taskID uint, page, pageSize int) ([]model.CronTaskExecution, int64, error) {
	var list []model.CronTaskExecution
	var total int64
	q := d.db.Model(&model.CronTaskExecution{})
	if taskID > 0 {
		q = q.Where("task_id = ?", taskID)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}
