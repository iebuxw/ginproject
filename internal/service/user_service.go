package service

import (
	"strings"

	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/utils"
)

type UserService struct {
	userDAO *dao.UserDAO
	roleDAO *dao.RoleDAO
}

func NewUserService(userDAO *dao.UserDAO, roleDAO *dao.RoleDAO) *UserService {
	return &UserService{userDAO: userDAO, roleDAO: roleDAO}
}

func (s *UserService) Create(u *model.User, roleIDs []uint) error {
	hashed, err := utils.HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashed
	u.Roles = buildRoles(roleIDs)
	return s.userDAO.Create(u)
}

func (s *UserService) Update(u *model.User, roleIDs []uint) error {
	if u.Password != "" {
		hashed, err := utils.HashPassword(u.Password)
		if err != nil {
			return err
		}
		u.Password = hashed
	}
	u.Roles = buildRoles(roleIDs)
	return s.userDAO.Update(u)
}

func buildRoles(ids []uint) []model.Role {
	roles := make([]model.Role, len(ids))
	for i, id := range ids {
		roles[i] = model.Role{ID: id}
	}
	return roles
}

func (s *UserService) Delete(id uint) error { return s.userDAO.Delete(id) }

func (s *UserService) FindByID(id uint) (*model.User, error) { return s.userDAO.FindByID(id) }

func (s *UserService) FindBatch(keyword string, offset, limit int) ([]model.User, error) {
	return s.userDAO.FindBatch(keyword, offset, limit)
}

func (s *UserService) FindPage(page, pageSize int, keyword string) ([]model.User, int64, error) {
	return s.userDAO.FindPage(page, pageSize, keyword)
}

// ImportRow 导入文件中的一行原始数据（Excel 行号从 2 开始，含表头行）
type ImportRow struct {
	Username    string
	Password    string
	Email       string
	Phone       string
	Description string
	Status      int
	RoleNames   string
	Row         int
}

// ImportFailure 校验失败的行及原因
type ImportFailure struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportResult 导入结果汇总
type ImportResult struct {
	Total            int             `json:"total"`
	Success          int             `json:"success"`
	Skipped          int             `json:"skipped"`
	SkippedUsernames []string        `json:"skipped_usernames"`
	Failed           []ImportFailure `json:"failed"`
}

// Import 批量导入用户：用户名已存在（库中或文件内重复）跳过，校验失败记入 failed，逐条创建互不影响
func (s *UserService) Import(rows []ImportRow) (*ImportResult, error) {
	result := &ImportResult{
		Total:            len(rows),
		SkippedUsernames: []string{},
		Failed:           []ImportFailure{},
	}

	roles, err := s.roleDAO.FindAll()
	if err != nil {
		return nil, err
	}
	roleIDs := make(map[string]uint, len(roles))
	for _, r := range roles {
		roleIDs[r.Name] = r.ID
	}

	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.Username == "" {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "用户名为空"})
			continue
		}
		if row.Password == "" {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "密码为空"})
			continue
		}
		if seen[row.Username] {
			result.Skipped++
			result.SkippedUsernames = append(result.SkippedUsernames, row.Username)
			continue
		}
		if _, err := s.userDAO.FindByUsername(row.Username); err == nil {
			result.Skipped++
			result.SkippedUsernames = append(result.SkippedUsernames, row.Username)
			continue
		}
		seen[row.Username] = true

		ids := make([]uint, 0)
		unknown := make([]string, 0)
		for _, name := range strings.Split(row.RoleNames, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if id, ok := roleIDs[name]; ok {
				ids = append(ids, id)
			} else {
				unknown = append(unknown, name)
			}
		}
		if len(unknown) > 0 {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "角色不存在: " + strings.Join(unknown, ",")})
			continue
		}

		hashed, err := utils.HashPassword(row.Password)
		if err != nil {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "密码哈希失败: " + err.Error()})
			continue
		}
		user := &model.User{
			Username:    row.Username,
			Password:    hashed,
			Email:       row.Email,
			Phone:       row.Phone,
			Description: row.Description,
			Status:      row.Status,
			Roles:       buildRoles(ids),
		}
		if err := s.userDAO.Create(user); err != nil {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "创建失败: " + err.Error()})
			continue
		}
		result.Success++
	}
	return result, nil
}
