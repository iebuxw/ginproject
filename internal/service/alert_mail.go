package service

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"time"

	"ginproject/internal/config"
	"ginproject/internal/logger"
)

// LoginAlertQueue 登录告警邮件任务队列名（发布端与消费端共用）
const LoginAlertQueue = "mail.send"

const smtpDialTimeout = 5 * time.Second

// AlertMailService 登录告警邮件发送器，供 MailWorker 消费时调用
type AlertMailService struct {
	cfg *config.Config
}

func NewAlertMailService(cfg *config.Config) *AlertMailService {
	return &AlertMailService{cfg: cfg}
}

// SendLoginAlert 发送登录异常告警邮件；SMTP 未配置时静默跳过
func (s *AlertMailService) SendLoginAlert(username, ip, message string) error {
	m := s.cfg.Mail
	if m.SMTPHost == "" || m.SMTPPort == "" || m.SMTPTo == "" {
		logger.Warn("登录告警邮件未发送：SMTP 未配置")
		return nil
	}

	from := m.SMTPFrom
	if from == "" {
		from = m.SMTPUser
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	subject := "【登录异常告警】"
	body := fmt.Sprintf(
		"系统检测到一次失败登录，请及时关注。\n\n"+
			"时间：%s\n用户名：%s\nIP：%s\n错误：%s",
		now, username, ip, message)

	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, m.SMTPTo, mime.QEncoding.Encode("UTF-8", subject), body)
	msg := []byte(header)

	addr := net.JoinHostPort(m.SMTPHost, m.SMTPPort)
	if err := s.sendMail(addr, from, m.SMTPTo, msg); err != nil {
		return fmt.Errorf("发送登录告警邮件失败: %w", err)
	}
	return nil
}

func (s *AlertMailService) sendMail(addr, from, to string, msg []byte) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	var conn net.Conn
	if port == "465" {
		// 465 端口为隐式 SSL，需先建立 TLS 连接
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: smtpDialTimeout}, "tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.DialTimeout("tcp", addr, smtpDialTimeout)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if port != "465" {
		if hasStartTLS, _ := client.Extension("STARTTLS"); hasStartTLS {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return err
			}
		}
	}

	m := s.cfg.Mail
	if m.SMTPUser != "" {
		auth := smtp.PlainAuth("", m.SMTPUser, m.SMTPPassword, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
