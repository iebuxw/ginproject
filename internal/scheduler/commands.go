package scheduler

// CommandDef 预定义命令定义
type CommandDef struct {
	Name    string            // 命令标识，如 "clean_logs"
	Label   string            // 中文名称，如 "清理过期日志"
	Method  string            // GET 或 POST
	URL     string            // 回调地址
	Headers map[string]string // 固定请求头（可选）
	Body    string            // 固定请求体（可选）
}

// Commands 预定义命令注册表
var Commands = map[string]CommandDef{
	"clean_logs": {
		Name:   "clean_logs",
		Label:  "清理过期日志",
		Method: "POST",
		URL:    "/api/logs/cleanup?days=30",
		Headers: map[string]string{
			"X-Cleanup-Secret": "{{CLEANUP_SECRET}}",
		},
	},
	"backup_db": {
		Name:   "backup_db",
		Label:  "数据库备份",
		Method: "POST",
		URL:    "/api/db/backup",
	},
}

// CommandList 返回所有预定义命令的 name + label（供前端下拉使用）
type CommandOption struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

func CommandList() []CommandOption {
	opts := make([]CommandOption, 0, len(Commands))
	for _, c := range Commands {
		opts = append(opts, CommandOption{Name: c.Name, Label: c.Label})
	}
	return opts
}
