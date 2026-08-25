package node

import "testing"

func TestChinaTargetsCount(t *testing.T) {
	if len(chinaTargets) == 0 {
		t.Fatal("chinaTargets 为空")
	}
	t.Logf("全量目标数: %d", len(chinaTargets))
	// 默认去重后每省每运营商 1 条
	def := defaultChinaTargets()
	t.Logf("默认去重后数: %d", len(def))
	if len(def) == 0 {
		t.Fatal("默认去重后为空")
	}
	// 验证无重复 省|运营商
	seen := map[string]bool{}
	for _, d := range def {
		k := d.Province + "|" + d.ISP
		if seen[k] {
			t.Fatalf("存在重复省+运营商: %s", k)
		}
		seen[k] = true
	}
	// 验证省份覆盖
	provs := map[string]bool{}
	for _, d := range def {
		provs[d.Province] = true
	}
	t.Logf("覆盖省份数: %d", len(provs))
}

func TestFilterChinaTargets(t *testing.T) {
	// 按运营商筛选
	dx := FilterChinaTargets(nil, []string{"电信"})
	for _, d := range dx {
		if d.ISP != "电信" {
			t.Fatalf("筛选失败: %s", d.ISP)
		}
	}
	// 按省份筛选
	zj := FilterChinaTargets([]string{"浙江"}, nil)
	for _, d := range zj {
		if d.Province != "浙江" {
			t.Fatalf("筛选失败: %s", d.Province)
		}
	}
	t.Logf("电信目标数: %d, 浙江目标数: %d", len(dx), len(zj))
}