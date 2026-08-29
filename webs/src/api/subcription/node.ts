import request from "@/utils/request";
export function getNodes(){
  return request({
    url: "/api/v1/nodes/get",
    method: "get",
  });
}

// 节点概览（含国家/延迟/分组）
export function getNodeOverview(){
  return request({
    url: "/api/v1/nodes/overview",
    method: "get",
  });
}

export function getNodeQualityHistory(id: number, hours = 24){
  return request({
    url: "/api/v1/nodes/quality/history",
    method: "get",
    params: { id, hours },
  });
}

export function getNodeQualitySummary(){
  return request({
    url: "/api/v1/nodes/quality/summary",
    method: "get",
  });
}
export function getNodeRecommendations(){ return request({ url: "/api/v1/nodes/recommendations", method: "get" }); }
export function getNodeHealthEvents(limit = 30){ return request({ url: "/api/v1/nodes/health/events", method: "get", params: { limit } }); }
export function getAlertSetting(){ return request({ url: "/api/v1/nodes/alerts", method: "get" }); }
export function updateAlertSetting(data: any){ return request({ url: "/api/v1/nodes/alerts", method: "post", data, headers: { "Content-Type": "multipart/form-data" } }); }

// 节点解锁测试
export function UnlockTest(data: any){
  return request({
    url: "/api/v1/nodes/unlock",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function EgressTest(data: any){ return request({ url: "/api/v1/nodes/egress", method: "post", data, headers: { "Content-Type": "multipart/form-data" } }); }

// 中国各地延迟测试
export function ChinaPingTest(data: any){
  return request({
    url: "/api/v1/nodes/chinaping",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

// 当前测试状态
export function GetTestStatus(){
  return request({
    url: "/api/v1/nodes/test/status",
    method: "get",
  });
}
// 停止当前测试
export function CancelTest(){
  return request({
    url: "/api/v1/nodes/test/cancel",
    method: "post",
  });
}

export function AddNodes(data: any){
  return request({
    url: "/api/v1/nodes/add",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function UpdateNode(data: any){
  return request({
    url: "/api/v1/nodes/update",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function DelNode(data: any){
  return request({
    url: "/api/v1/nodes/delete",
    method: "delete",
    params: data,
  });
}
// 获取全部分组
export function GetGroup(){
  return request({
    url: "/api/v1/nodes/group/get",
    method: "get",
  });
}
// 获取分组完整信息（含 ID/节点数）
export function GetGroupFull(){
  return request({
    url: "/api/v1/nodes/group/full",
    method: "get",
  });
}
// 设置关联分组
export function SetGroup(data: any){
  return request({
    url: "/api/v1/nodes/group/set",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
// 解除节点与分组绑定
export function UnbindGroup(data: any){
  return request({
    url: "/api/v1/nodes/group/unbind",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
// 删除分组

export function DelGroup(data: any){
  return request({
    url: "/api/v1/nodes/group/delete",
    method: "delete",
    params: data,
  });
}
