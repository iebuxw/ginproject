package controller

import (
	"context"
	"time"

	"ginproject/internal/config"
	"ginproject/internal/utils"
	"ginproject/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type WSController struct {
	hub *ws.Hub
	rdb *redis.Client
	cfg *config.Config
}

func NewWSController(hub *ws.Hub, rdb *redis.Client, cfg *config.Config) *WSController {
	return &WSController{hub: hub, rdb: rdb, cfg: cfg}
}

func (ctl *WSController) Handle(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(401, gin.H{"code": 401, "message": "缺少token"})
		return
	}

	_, err := ctl.rdb.Get(context.Background(), "blacklist:"+token).Result()
	if err == nil {
		c.JSON(401, gin.H{"code": 401, "message": "Token已失效"})
		return
	}

	claims, err := utils.ParseToken(token, ctl.cfg.JWT.Secret)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "message": "Token无效"})
		return
	}

	conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	userID := claims.UserID
	ctl.hub.Register(userID, conn)
	defer ctl.hub.Unregister(userID)
	defer conn.Close()

	stopCh := make(chan struct{})

	// 读协程：每次读之前刷新 deadline，客户端任何消息都能续期
	// 读协程 和 心跳，两边任意一边断线，另一边都会跟着退出
	go func() {
		defer close(stopCh)
		for {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))// 读超时 60 秒
			if _, _, err := conn.NextReader(); err != nil {
				break
			}
		}
	}()

	// 心跳
	ticker := time.NewTicker(30 * time.Second)// 每 30 秒下发心跳
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))// 读超时 10 秒
			if err := conn.WriteJSON(ws.Message{Type: "heartbeat"}); err != nil {
				return
			}
		case <-stopCh:
			return
		}
	}
}
