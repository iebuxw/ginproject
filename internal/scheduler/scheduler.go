package scheduler

import (
	"context"
	"encoding/json"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// 6 段 cron 表达式解析器（秒 分 时 日 月 周）
var parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// 执行日志状态
const (
	ExecStatusSuccess = 0 // 成功
	ExecStatusFailed  = 1 // 失败
	ExecStatusSkipped = 2 // 跳过（上次未执行完）
)

const maxResponseLen = 2000

// ParseCron 校验并解析 6 段 cron 表达式（秒 分 时 日 月 周）
func ParseCron(expr string) (cron.Schedule, error) {
	return parser.Parse(expr)
}

type Scheduler struct {
	cron    *cron.Cron
	taskDAO *dao.CronTaskDAO
	execDAO *dao.CronTaskExecutionDAO
	running sync.Map // taskID → bool，防重叠执行
	client  *http.Client
}

func NewScheduler(taskDAO *dao.CronTaskDAO, execDAO *dao.CronTaskExecutionDAO) *Scheduler {
	return &Scheduler{
		taskDAO: taskDAO,
		execDAO: execDAO,
		client:  &http.Client{},
	}
}

// Start 全量加载启用任务并启动调度（main.go goroutine 调用）
func (s *Scheduler) Start() {
	s.Reload()
}

// Reload 热更新：停止旧实例并全量重建（任务增删改/启停后调用）
func (s *Scheduler) Reload() {
	if s.cron != nil {
		s.cron.Stop() // 等待运行中的 job 结束
	}
	c := cron.New(cron.WithParser(parser))
	tasks, err := s.taskDAO.FindEnabled()
	if err != nil {
		log.Printf("调度器加载任务失败: %v", err)
		return
	}
	for i := range tasks {
		task := tasks[i]
		_, err := c.AddFunc(task.Cron, func() {
			s.execute(&task, "cron")
		})
		if err != nil {
			log.Printf("任务 %d 注册失败: %v", task.ID, err)
		}
	}
	c.Start()
	s.cron = c
	log.Printf("调度器已加载 %d 个启用任务", len(tasks))
}

// RunNow 立即执行一次（写 trigger=manual 日志）
func (s *Scheduler) RunNow(id uint) error {
	task, err := s.taskDAO.FindByID(id)
	if err != nil {
		return err
	}
	s.execute(task, "manual")
	return nil
}

func (s *Scheduler) execute(task *model.CronTask, trigger string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("任务 %d 执行 panic: %v", task.ID, r)
		}
	}()

	// 防重叠：上次未执行完则跳过本次
	if _, loaded := s.running.LoadOrStore(task.ID, true); loaded {
		s.saveExec(task.ID, trigger, ExecStatusSkipped, 0, "", "上次未执行完，跳过本次", 0)
		return
	}
	defer s.running.Delete(task.ID)

	start := time.Now()

	// 预定义命令：进程内直接调用 Handler，不走 HTTP
	if task.Command != "" {
		cmd, ok := Commands[task.Command]
		if !ok {
			s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", "未知命令: "+task.Command, 0)
			return
		}
		res, err := cmd.Handler(0)
		if err != nil {
			s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", err.Error(), int(time.Since(start).Milliseconds()))
			return
		}
		s.saveExec(task.ID, trigger, ExecStatusSuccess, 0, res.Message, "", int(time.Since(start).Milliseconds()))
		return
	}

	// 自定义 HTTP 模式
	method := task.Method
	url := task.URL
	headers := task.Headers
	bodyStr := task.Body
	var body io.Reader
	if method == "POST" {
		body = strings.NewReader(bodyStr)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", "请求创建失败: "+err.Error(), int(time.Since(start).Milliseconds()))
		return
	}
	// 请求头（JSON 对象，创建时已校验）
	if strings.TrimSpace(headers) != "" {
		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err == nil {
			for k, v := range headerMap {
				req.Header.Set(k, v)
			}
		}
	}
	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(task.Timeout)*time.Second)
	defer cancel()

	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil {
		s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", err.Error(), int(time.Since(start).Milliseconds()))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	text := strings.TrimSpace(string(respBody))
	if len(text) > maxResponseLen {
		text = text[:maxResponseLen]
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.saveExec(task.ID, trigger, ExecStatusFailed, resp.StatusCode, text,
			"HTTP 状态码非 2xx: "+resp.Status, int(time.Since(start).Milliseconds()))
		return
	}
	s.saveExec(task.ID, trigger, ExecStatusSuccess, resp.StatusCode, text, "", int(time.Since(start).Milliseconds()))
}

func (s *Scheduler) saveExec(taskID uint, trigger string, status, httpStatus int, response, errMsg string, durationMS int) {
	e := &model.CronTaskExecution{
		TaskID:     taskID,
		Trigger:    trigger,
		Status:     status,
		HTTPStatus: httpStatus,
		Response:   response,
		ErrorMsg:   errMsg,
		DurationMS: durationMS,
	}
	if err := s.execDAO.Create(e); err != nil {
		log.Printf("任务 %d 执行日志写入失败: %v", taskID, err)
	}
}
