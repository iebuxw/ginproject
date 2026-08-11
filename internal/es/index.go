package es

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

// LogIndex 操作日志索引名
const LogIndex = "operation_logs"

// logMapping 索引映射（对标 MySQL operation_logs 表）：
//   - path/params/response 用 IK 中文分词，text 与 keyword 子字段对比学习
//   - created_at 格式与 model.DateTime.MarshalJSON 输出一致（yyyy-MM-dd HH:mm:ss）
const logMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "analyzer": {
        "ik_max_word": { "type": "ik_max_word" },
        "ik_smart":     { "type": "ik_smart" }
      }
    }
  },
  "mappings": {
    "properties": {
      "id":          { "type": "long" },
      "operator_id": { "type": "long" },
      "module":      { "type": "keyword" },
      "action":      { "type": "keyword" },
      "method":      { "type": "keyword" },
      "path":        { "type": "text", "fields": { "keyword": { "type": "keyword" } }, "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
      "params":      { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
      "response":    { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
      "duration":    { "type": "integer" },
      "ip":          { "type": "keyword" },
      "created_at":  { "type": "date", "format": "yyyy-MM-dd HH:mm:ss" }
    }
  }
}`

// EnsureIndex 幂等创建索引（对标 GORM AutoMigrate，已有则跳过）
func EnsureIndex(cli *elasticsearch.Client) error {
	exists, err := esapi.IndicesExistsRequest{Index: []string{LogIndex}}.Do(context.Background(), cli)
	if err != nil {
		return err
	}
	exists.Body.Close()
	if exists.StatusCode == 200 {
		return nil
	}
	res, err := esapi.IndicesCreateRequest{Index: LogIndex, Body: strings.NewReader(logMapping)}.Do(context.Background(), cli)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("创建索引失败: %s", res.String())
	}
	return nil
}
