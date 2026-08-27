package scheduler

import "fmt"

// CommandResult 预定义命令执行结果（写入执行日志的 Response）
type CommandResult struct {
	Message string
}

// CommandDef 预定义命令定义
type CommandDef struct {
	Name    string                        // 命令标识，如 "clean_logs"
	Label   string                        // 中文名称，如 "清理过期日志"
	Handler func(days int) (CommandResult, error) // 进程内执行（days 为任务超时外的预留参数，暂未使用传 0）
}

// Commands 预定义命令注册表（main.go 组装时注入真实 Handler）
var Commands = map[string]CommandDef{
	"clean_logs": {
		Name:   "clean_logs",
		Label:  "清理过期日志",
		Handler: func(days int) (CommandResult, error) {
			return CommandResult{}, fmt.Errorf("未注入实现")
		},
	},
	"backup_db": {
		Name:   "backup_db",
		Label:  "数据库备份",
		Handler: func(days int) (CommandResult, error) {
			return CommandResult{}, fmt.Errorf("命令未实现")
		},
	},
	"clean_backup": {
		Name:   "clean_backup",
		Label:  "清理过期备份",
		Handler: func(days int) (CommandResult, error) {
			return CommandResult{}, fmt.Errorf("命令未实现")
		},
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
