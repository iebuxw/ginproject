package worker

import (
	"encoding/json"

	"ginproject/internal/logger"
	"ginproject/internal/service"

	"go.uber.org/zap"

	"github.com/rabbitmq/amqp091-go"
)

// MailWorker 消费 mail.send 队列，发送登录告警邮件
type MailWorker struct {
	amqpConn  *amqp091.Connection
	alertMail *service.AlertMailService
}

type mailTaskMessage struct {
	Username string `json:"username"`
	IP       string `json:"ip"`
	Message  string `json:"message"`
}

func NewMailWorker(amqpConn *amqp091.Connection, alertMail *service.AlertMailService) *MailWorker {
	return &MailWorker{amqpConn: amqpConn, alertMail: alertMail}
}

func (w *MailWorker) Start() {
	ch, err := w.amqpConn.Channel()
	if err != nil {
		panic("RabbitMQ channel 创建失败: " + err.Error())
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(service.LoginAlertQueue, true, false, false, false, nil)
	if err != nil {
		panic("RabbitMQ 队列声明失败: " + err.Error())
	}

	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		panic("RabbitMQ 消费注册失败: " + err.Error())
	}

	for msg := range msgs {
		var task mailTaskMessage
		if err := json.Unmarshal(msg.Body, &task); err != nil {
			logger.Error("邮件任务解析失败", zap.Error(err))
			msg.Ack(false)
			continue
		}
		if err := w.alertMail.SendLoginAlert(task.Username, task.IP, task.Message); err != nil {
			logger.Error("登录告警邮件发送失败", zap.Error(err))
		}
		msg.Ack(false)
	}
}
