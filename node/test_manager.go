package node

import (
	"context"
	"sync"
	"time"
)

// TestSession 当前测试会话状态
type TestSession struct {
	NodeName  string    `json:"nodeName"`
	NodeID    int       `json:"nodeId"`
	Type      string    `json:"type"` // unlock / tcp
	StartedAt time.Time `json:"startedAt"`
}

// testManager 管理全局测试会话
type testManager struct {
	mu     sync.Mutex
	session *TestSession
	cancel context.CancelFunc
	busy   bool
}

var tm = &testManager{}

// BeginTest 开始一个测试会话。
// 若已有测试进行中返回 false + 当前会话信息；否则记录会话并返回可取消的 ctx。
// baseCtx 是请求的 context（客户端断开时也会取消）。
func BeginTest(nodeName string, nodeID int, testType string, baseCtx context.Context) (context.Context, context.CancelFunc, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.busy {
		// 已有测试：返回当前会话供前端提示
		return nil, nil, false
	}
	ctx, cancel := context.WithCancel(baseCtx)
	tm.session = &TestSession{
		NodeName: nodeName, NodeID: nodeID, Type: testType, StartedAt: time.Now(),
	}
	tm.cancel = cancel
	tm.busy = true
	return ctx, cancel, true
}

// EndTest 结束测试会话（defer 调用），释放锁。
func EndTest() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.session = nil
	tm.cancel = nil
	tm.busy = false
}

// GetTestStatus 返回当前测试会话，无测试时返回 nil。
func GetTestStatus() *TestSession {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.busy || tm.session == nil {
		return nil
	}
	sess := *tm.session
	return &sess
}

// CancelTest 主动停止当前测试（调用 cancel 使测试 ctx 取消，从而提前结束并释放锁）。
// 返回是否成功停止。
func CancelTest() bool {
	tm.mu.Lock()
	if !tm.busy || tm.cancel == nil {
		tm.mu.Unlock()
		return false
	}
	cancel := tm.cancel
	tm.mu.Unlock()
	cancel()
	return true
}