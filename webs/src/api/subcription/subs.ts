import request from "@/utils/request";
export function getSubs(){
  return request({
    url: "/api/v1/subcription/get",
    method: "get",
  });
}

export function getSubPreviewNodes(id: number){
  return request({
    url: "/api/v1/subcription/preview-nodes",
    method: "get",
    params: { id },
  });
}
export function subscriptionEgressPlan(id: number){
  return request({ url: "/api/v1/subcription/egress-plan", method: "post", params: { id } });
}
export function subscriptionRuleExplain(data: any){
  return request({ url: "/api/v1/subcription/rule-explain", method: "post", data });
}
export function previewSubPipeline(data: any){
  return request({ url: "/api/v1/subcription/pipeline/preview", method: "post", data, headers: { "Content-Type": "multipart/form-data" } });
}
export function startSubscriptionBuild(data:any){ return request({ url:'/api/v1/subcription/build-task', method:'post', data }); }
export function getSubscriptionArtifacts(id:number, client = ''){ return request({ url:'/api/v1/subcription/artifacts', method:'get', params:{ id, client } }); }
export function rollbackSubscriptionArtifact(artifactId:number){ return request({ url:'/api/v1/subcription/artifacts/rollback', method:'post', params:{ artifactId } }); }
export function safePublishSubscription(data:any){ return request({ url:'/api/v1/tasks/safe-publish', method:'post', data }); }

export function AddSub(data: any){
  return request({
    url: "/api/v1/subcription/add",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function DelSub(data: any){
  return request({
    url: "/api/v1/subcription/delete",
    method: "delete",
    params: data,
  });
}

export function UpdateSub(data: any){
  return request({
    url: "/api/v1/subcription/update",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

export function ResetToken(data: any){
  return request({
    url: "/api/v1/subcription/reset-token",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

export function SetExpire(data: any){
  return request({
    url: "/api/v1/subcription/set-expire",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
