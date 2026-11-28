package vector

import (
	"testing"
)

func TestWedge(t *testing.T) {
	t.Parallel()

	mat, ok := CalculateWedge([]int{1, 2, 3}, []int{4, 5, 6})
	if !ok {
		t.Fatal("CalculateWedge 返回 false, want true")
	}
	if len(mat) != 3 {
		t.Fatalf("CalculateWedge 行数 = %d, want 3", len(mat))
	}

	// 外积矩阵为反对称矩阵：M[i][j] == -M[j][i]，对角线为 0
	for i := range mat {
		if mat[i][i] != 0 {
			t.Errorf("mat[%d][%d] = %d, want 0", i, i, mat[i][i])
		}
		for j := range mat[i] {
			if mat[i][j] != -mat[j][i] {
				t.Errorf("mat[%d][%d] = %d, mat[%d][%d] = %d, want 相反数", i, j, mat[i][j], j, i, mat[j][i])
			}
		}
	}
}

func TestWedgeInvalidShape(t *testing.T) {
	t.Parallel()

	if _, ok := CalculateWedge([]int{}, []int{}); ok {
		t.Error("空切片 CalculateWedge 返回 true, want false")
	}
	if _, ok := CalculateWedge([]int{1, 2}, []int{1}); ok {
		t.Error("长度不一致 CalculateWedge 返回 true, want false")
	}
}
