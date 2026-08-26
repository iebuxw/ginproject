package service

import (
	"encoding/json"
	"errors"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/scheduler"
	"strings"
)

type CronTaskService struct {
	taskDAO   *dao.CronTaskDAO
	execDAO   *dao.CronTaskExecutionDAO
	scheduler *scheduler.Scheduler
}

func NewCronTaskService(taskDAO *dao.CronTaskDAO, execDAO *dao.CronTaskExecutionDAO, s *scheduler.Scheduler) *CronTaskService {
	return &CronTaskService{taskDAO: taskDAO, execDAO: execDAO, scheduler: s}
}

func (s *CronTaskService) validate(t *model.CronTask) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("任务名称不能为空")
	}
	if strings.TrimSpace(t.URL) == "" {
		return errors.New("回调地址不能为空")
	}
	if t.Method != "GET" && t.Method != "POST" {
		return errors.New("请求方式仅支持 GET/POST")
	}
	if t.Timeout < 1 || t.Timeout > 300 {
		return errors.New("超时时间需在 1-300 秒之间")
	}
	if _, err := scheduler.ParseCron(t.Cron); err != nil {
		return errors.New("cron 表达式不合法（格式：秒 分 时 日 月 周）")
	}
	if strings.TrimSpace(t.Headers) != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(t.Headers), &m); err != nil {
			return errors.New("请求头必须是 JSON 对象")
		}
	}
	return nil
}

func (s *CronTaskService) Create(t *model.CronTask) error {
	if err := s.validate(t); err != nil {
		return err
	}
	if err := s.taskDAO.Create(t); err != nil {
		return err
	}
	s.scheduler.Reload()
	return nil
}

func (s *CronTaskService) Update(t *model.CronTask) error {
	if err := s.validate(t); err != nil {
		return err
	}
	if err := s.taskDAO.Update(t); err != nil {
		return err
	}
	s.scheduler.Reload()
	return nil
}

func (s *CronTaskService) UpdateStatus(id uint, status int) error {
	if status != 0 && status != 1 {
		return errors.New("状态值不合法")
	}
	t, err := s.taskDAO.FindByID(id)
	if err != nil {
		return err
	}
	t.Status = status
	if err := s.taskDAO.Update(t); err != nil {
		return err
	}
	s.scheduler.Reload()
	return nil
}

func (s *CronTaskService) Delete(id uint) error {
	if err := s.taskDAO.Delete(id); err != nil {
		return err
	}
	s.scheduler.Reload()
	return nil
}

func (s *CronTaskService) FindByID(id uint) (*model.CronTask, error) {
	return s.taskDAO.FindByID(id)
}

func (s *CronTaskService) FindPage(page, pageSize int, keyword string) ([]model.CronTask, int64, error) {
	return s.taskDAO.FindPage(page, pageSize, keyword)
}

func (s *CronTaskService) RunNow(id uint) error {
	return s.scheduler.RunNow(id)
}

func (s *CronTaskService) FindExecutions(taskID uint, page, pageSize int) ([]model.CronTaskExecution, int64, error) {
	return s.execDAO.FindByTaskIDPage(taskID, page, pageSize)
}
