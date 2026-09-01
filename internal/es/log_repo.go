package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"ginproject/internal/logger"
	"ginproject/internal/model"

	"go.uber.org/zap"

	"github.com/elastic/go-elasticsearch/v7/esapi"
)

// LogRepo ES 操作日志数据访问层（对标 dao.LogDAO 的 ES 版本）
type LogRepo struct{ cli *Client }

func NewLogRepo(cli *Client) *LogRepo { return &LogRepo{cli: cli} }

// Enabled 判断 ES 是否可用；nil 时组件仍可调用，方法内部降级
func (r *LogRepo) Enabled() bool { return r != nil && r.cli != nil }

// Index 同步写入单条日志，_id 复用 MySQL 主键实现双写幂等；
// Refresh=true 使写入即时可见（学习 refresh 语义，写入量小可接受）
func (r *LogRepo) Index(log *model.OperationLog) error {
	if r.cli == nil {
		return fmt.Errorf("ES 未启用")
	}
	body, err := json.Marshal(log)
	if err != nil {
		return err
	}
	res, err := (&esapi.IndexRequest{
		Index:      LogIndex,
		DocumentID: fmt.Sprintf("%d", log.ID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}).Do(context.Background(), r.cli.RawClient())
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("写入失败: %s", res.String())
	}
	return nil
}

// SearchQuery 全文检索条件；时间格式 "2006-01-02 15:04:05"
type SearchQuery struct {
	Keyword   string
	Module    string
	Method    string
	StartTime string
	EndTime   string
	From      int
	Size      int
}

// SearchHitDoc 搜索结果（_source 字段 + 高亮字段）
type SearchHitDoc struct {
	model.OperationLog
	HighlightPath   string `json:"highlight_path,omitempty"`
	HighlightParams string `json:"highlight_params,omitempty"`
}

// Search 组装 bool 查询 DSL 并解析命中与高亮
func (r *LogRepo) Search(ctx context.Context, q SearchQuery) ([]SearchHitDoc, int64, error) {
	if r.cli == nil {
		return nil, 0, fmt.Errorf("ES 未启用")
	}

	body := map[string]interface{}{
		"query": buildBoolQuery(q),
		"from":  q.From,
		"size":  q.Size,
		"sort": []map[string]interface{}{
			{"created_at": map[string]interface{}{"order": "desc"}},
		},
		"highlight": map[string]interface{}{
			"pre_tags":  []string{"<em>"},
			"post_tags": []string{"</em>"},
			"fields": map[string]interface{}{
				"path":   map[string]interface{}{},
				"params": map[string]interface{}{},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, 0, err
	}
	raw := r.cli.RawClient()
	res, err := raw.Search(raw.Search.WithContext(ctx), raw.Search.WithIndex(LogIndex), raw.Search.WithBody(&buf))
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, 0, fmt.Errorf("查询失败: %s", res.String())
	}

	var parsed struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    json.RawMessage     `json:"_source"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}

	docs := make([]SearchHitDoc, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		var doc SearchHitDoc
		if err := json.Unmarshal(h.Source, &doc.OperationLog); err != nil {
			logger.Warn("ES 命中解析失败", zap.Error(err))
			continue
		}
		doc.HighlightPath = sanitizeHighlight(join(h.Highlight["path"]))
		doc.HighlightParams = sanitizeHighlight(join(h.Highlight["params"]))
		docs = append(docs, doc)
	}
	return docs, parsed.Hits.Total.Value, nil
}

// buildBoolQuery 构造 bool 查询：must=关键词全文检索，filter=精确/范围筛选
func buildBoolQuery(q SearchQuery) map[string]interface{} {
	must := make([]interface{}, 0, 1)
	if q.Keyword != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  q.Keyword,
				"fields": []string{"path", "params"},
			},
		})
	}
	filter := make([]interface{}, 0, 4)
	if q.Module != "" {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"module": q.Module}})
	}
	if q.Method != "" {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"method": q.Method}})
	}
	if q.StartTime != "" {
		filter = append(filter, map[string]interface{}{"range": map[string]interface{}{"created_at": map[string]interface{}{"gte": q.StartTime}}})
	}
	if q.EndTime != "" {
		filter = append(filter, map[string]interface{}{"range": map[string]interface{}{"created_at": map[string]interface{}{"lte": q.EndTime}}})
	}
	return map[string]interface{}{"bool": map[string]interface{}{"must": must, "filter": filter}}
}

func join(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "…")
}

// sanitizeHighlight 转义高亮片段防止 v-html XSS，仅放行 <em>/</em>
func sanitizeHighlight(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "<em>", "\x00em\x00")
	s = strings.ReplaceAll(s, "</em>", "\x00/em\x00")
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "\x00em\x00", "<em>")
	s = strings.ReplaceAll(s, "\x00/em\x00", "</em>")
	return s
}

// DeleteByTime 删除创建时间早于 before 的操作日志文档（delete_by_query），返回删除条数
func (r *LogRepo) DeleteByTime(before time.Time) (int64, error) {
	if r.cli == nil {
		return 0, fmt.Errorf("ES 未启用")
	}
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": map[string]interface{}{
					"lt": before.Format("2006-01-02 15:04:05"),
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return 0, err
	}
	refresh := true
	res, err := (&esapi.DeleteByQueryRequest{
		Index:   []string{LogIndex},
		Body:    bytes.NewReader(buf.Bytes()),
		Refresh: &refresh,
	}).Do(context.Background(), r.cli.RawClient())
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return 0, fmt.Errorf("删除失败: %s", res.String())
	}
	var parsed struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Deleted, nil
}
