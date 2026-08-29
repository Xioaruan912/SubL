package node

import (
	"context"
	"net"
	"testing"
	"time"
)

type blockingContextDialer struct{}

func (blockingContextDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

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

func TestTCPPingViaHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if got := tcpPingVia(ctx, blockingContextDialer{}, "127.0.0.1:1", time.Second); got != -1 {
		t.Fatalf("取消后的结果 = %d，期望 -1", got)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("取消未及时生效，耗时 %v", elapsed)
	}
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
