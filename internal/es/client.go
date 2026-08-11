package es

import (
	"ginproject/internal/config"

	"github.com/elastic/go-elasticsearch/v7"
)

// Client ES 客户端封装
type Client struct{ raw *elasticsearch.Client }

// NewClient 构建 ES 客户端；不做连通性校验，由 main 决定是否启用降级
func NewClient(cfg *config.ElasticsearchConfig) (*Client, error) {
	esCfg := elasticsearch.Config{Addresses: []string{cfg.Addr()}}
	if cfg.Username != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
	}
	raw, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}
	return &Client{raw: raw}, nil
}

// Ping 探活；失败说明 ES 不可用，调用方据此降级
func (c *Client) Ping() error {
	res, err := c.raw.Ping()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

// RawClient 暴露底层客户端，供 LogRepo 等使用
func (c *Client) RawClient() *elasticsearch.Client { return c.raw }
