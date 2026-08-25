// 节点测试结果 localStorage 持久化工具
// 刷新页面 / 切换页面后，回到节点列表点开卡片仍可显示之前的测试结果

const PREFIX = "sublink_node_test_";
const TTL = 60 * 60 * 1000; // 1 小时过期，避免无限累积

interface StoredData<T> {
  data: T;
  ts: number;
}

function read<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredData<T>;
    // 过期清理
    if (Date.now() - parsed.ts > TTL) {
      localStorage.removeItem(PREFIX + key);
      return null;
    }
    return parsed.data;
  } catch {
    return null;
  }
}

function write<T>(key: string, data: T) {
  try {
    const obj: StoredData<T> = { data, ts: Date.now() };
    localStorage.setItem(PREFIX + key, JSON.stringify(obj));
  } catch { /* 存储满或不可用则忽略 */ }
}

function remove(key: string) {
  try {
    localStorage.removeItem(PREFIX + key);
  } catch { /* ignore */ }
}

// 解锁测试结果
export function saveUnlock(nodeId: number, data: any) {
  write(`unlock_${nodeId}`, data);
}
export function getUnlock(nodeId: number): any {
  return read(`unlock_${nodeId}`);
}
export function clearUnlock(nodeId: number) {
  remove(`unlock_${nodeId}`);
}

// TCP 测试结果
export function saveTcp(nodeId: number, data: any) {
  write(`tcp_${nodeId}`, data);
}
export function getTcp(nodeId: number): any {
  return read(`tcp_${nodeId}`);
}
export function clearTcp(nodeId: number) {
  remove(`tcp_${nodeId}`);
}