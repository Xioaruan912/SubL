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