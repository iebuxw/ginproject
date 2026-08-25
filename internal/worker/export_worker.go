package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"ginproject/internal/dao"
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
	method, _ := w.rdb.HGet(ctx, taskKey, "method").Result()
	keyword, _ := w.rdb.HGet(ctx, taskKey, "keyword").Result()
	startTime, _ := w.rdb.HGet(ctx, taskKey, "start_time").Result()
	endTime, _ := w.rdb.HGet(ctx, taskKey, "end_time").Result()

	var uid uint
	fmt.Sscanf(userIDStr, "%d", &uid)

	filename := fmt.Sprintf("操作日志_%s.xlsx", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportDir, taskID+".xlsx")

	filter := dao.LogFilter{
		Method:    method,
		Keyword:   keyword,
		StartTime: startTime,
		EndTime:   endTime,
	}

	err := w.buildExcel(filter, filePath)
	if err != nil {
		log.Printf("导出任务 %s 失败: %v", taskID, err)
		w.rdb.HSet(ctx, taskKey, "status", "failed")
		w.rdb.HSet(ctx, taskKey, "error", err.Error())
		w.hub.Send(uid, ws.Message{
			Type:  "export_failed",
			TaskID: taskID,
			Error:  err.Error(),
		})
		return
	}

	w.rdb.HSet(ctx, taskKey, "status", "success")
	w.rdb.HSet(ctx, taskKey, "filename", filename)
	w.rdb.Expire(ctx, taskKey, taskTTL)

	w.hub.Send(uid, ws.Message{
		Type:        "export_complete",
		TaskID:      taskID,
		Filename:    filename,
		DownloadURL: "/api/logs/download/" + taskID,
	})
}

func (w *ExportWorker) buildExcel(filter dao.LogFilter, filePath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "操作日志"
	f.SetSheetName("Sheet1", sheet)

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "操作人ID", "请求方式", "请求路径", "参数", "耗时(ms)", "IP", "操作时间"}
	headerVals := make([]interface{}, len(headers))
	for i, h := range headers {
		headerVals[i] = h
	}
	sw.SetRow("A1", headerVals)

	offset := 0
	row := 2

	for {
		logs, err := w.logService.FindBatch(filter, offset, batchSize)
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

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)

	return f.SaveAs(filePath)
}
