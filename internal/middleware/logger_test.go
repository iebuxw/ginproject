package middleware

import "testing"

func TestMaskSensitiveParams(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "登录密码脱敏",
			in:   `{"username":"admin","password":"123456"}`,
			want: `{"password":"***","username":"admin"}`,
		},
		{
			name: "修改密码字段脱敏",
			in:   `{"old_password":"1","new_password":"2"}`,
			want: `{"new_password":"***","old_password":"***"}`,
		},
		{
			name: "嵌套结构中的密码脱敏",
			in:   `{"user":{"password":"secret","email":"a@b.com"},"note":"password 提示语不应受影响"}`,
			want: `{"note":"password 提示语不应受影响","user":{"email":"a@b.com","password":"***"}}`,
		},
		{
			name: "无敏感字段保持原样",
			in:   `{"username":"admin","email":"a@b.com"}`,
			want: `{"email":"a@b.com","username":"admin"}`,
		},
		{
			name: "非 JSON 回退正则",
			in:   `password=hunter2&username=admin`,
			want: `password=***&username=admin`,
		},
		{
			name: "空字符串",
			in:   ``,
			want: ``,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maskSensitiveParams(tc.in); got != tc.want {
				t.Errorf("maskSensitiveParams(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
