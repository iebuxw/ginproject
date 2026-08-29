package service

import (
	"sort"
	"testing"
)

func TestExpandTargets(t *testing.T) {
	cases := []struct {
		name    string
		target  SendTarget
		want    []uint
		wantErr bool
	}{
		{
			name:    "全员广播",
			target:  SendTarget{TargetType: 1, UserIDs: []uint{1, 2, 3}},
			want:    []uint{1, 2, 3},
		},
		{
			name:    "指定用户去重",
			target:  SendTarget{TargetType: 3, UserIDs: []uint{5, 3, 5, 1}},
			want:    []uint{1, 3, 5},
		},
		{
			name:    "指定用户为空报错",
			target:  SendTarget{TargetType: 3, UserIDs: []uint{}},
			wantErr: true,
		},
		{
			name:    "角色用户为空报错",
			target:  SendTarget{TargetType: 2, UserIDs: []uint{}},
			wantErr: true,
		},
		{
			name:    "非法 target_type",
			target:  SendTarget{TargetType: 9, UserIDs: []uint{1}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandTargets(tc.target)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际返回 %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错: %v", err)
			}
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
