package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ginproject/internal/service"
	"ginproject/internal/ws"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
)

const (
	queueName  = "excel.export"
	taskPrefix = "excel:task:"
	exportDir  = "exports"
	taskTTL    = 24 * time.Hour
	batchSize  = 5000
)

type ExportWorker struct {
	rdb        *redis.Client
	amqpConn   *amqp091.Connection
	logService *service.LogService
	hub        *ws.Hub
}

type queueMessage struct {
	TaskID string `json:"task_id"`
}

func NewExportWorker(rdb *redis.Client, amqpConn *amqp091.Connection, logService *service.LogService, hub *ws.Hub) *ExportWorker {
	return &ExportWorker{rdb: rdb, amqpConn: amqpConn, logService: logService, hub: hub}
}

func (w *ExportWorker) Start() {
	os.MkdirAll(exportDir, 0755)

	ch, err := w.amqpConn.Channel()
	if err != nil {
		panic("RabbitMQ channel 创建失败: " + err.Error())
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		panic("RabbitMQ 队列声明失败: " + err.Error())
	}

	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		panic("RabbitMQ 消费注册失败: " + err.Error())
	}

	for msg := range msgs {
		var qm queueMessage
		if json.Unmarshal(msg.Body, &qm) == nil {
			w.processTask(qm.TaskID)
		}
		msg.Ack(false)
	}
}

func (w *ExportWorker) processTask(taskID string) {
	taskKey := taskPrefix + taskID
	ctx := context.Background()

	w.rdb.HSet(ctx, taskKey, "status", "processing")

	userIDStr, _ := w.rdb.HGet(ctx, taskKey, "user_id").Result()
	module, _ := w.rdb.HGet(ctx, taskKey, "module").Result()
	method, _ := w.rdb.HGet(ctx, taskKey, "method").Result()

	var uid uint
	fmt.Sscanf(userIDStr, "%d", &uid)

	filename := fmt.Sprintf("操作日志_%s.xlsx", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportDir, taskID+".xlsx")

	err := w.buildExcel(module, method, filePath)
	if err != nil {
		w.rdb.HSet(ctx, taskKey, "status", "failed", "error", err.Error())
		w.hub.Send(uid, ws.Message{
			Type:  "export_failed",
			TaskID: taskID,
			Error:  err.Error(),
		})
		return
	}

	w.rdb.HSet(ctx, taskKey,
		"status", "success",
		"filename", filename,
	)
	w.rdb.Expire(ctx, taskKey, taskTTL)

	w.hub.Send(uid, ws.Message{
		Type:        "export_complete",
		TaskID:      taskID,
		Filename:    filename,
		DownloadURL: "/api/logs/download/" + taskID,
	})
}

func (w *ExportWorker) buildExcel(module, method, filePath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "操作日志"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "操作人ID", "请求方式", "请求路径", "参数", "耗时(ms)", "IP", "操作时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	offset := 0
	row := 2

	for {
		logs, err := w.logService.FindBatch(module, method, offset, batchSize)
		if err != nil {
			return err
		}
		if len(logs) == 0 {
			break
		}

		for _, log := range logs {
			cell, _ := excelize.CoordinatesToCellName(1, row)
			values := []interface{}{
				log.ID, log.OperatorID, log.Method, log.Path,
				log.Params, log.Duration, log.IP,
				time.Time(log.CreatedAt).Format("2006-01-02 15:04:05"),
			}
			if err := sw.SetRow(cell, values); err != nil {
				return err
			}
			row++
		}

		offset += batchSize
		if len(logs) < batchSize {
			break
		}
	}

	if err := sw.Flush(); err != nil {
		return err
	}

	return f.SaveAs(filePath)
}
