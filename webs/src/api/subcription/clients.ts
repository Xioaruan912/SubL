import request from "@/utils/request";

export function getClientList(){
  return request({
    url: "/api/v1/clients/list",
    method: "get",
  });
}

export function checkClient(){
  return request({
    url: "/api/v1/clients/check",
    method: "post",
  });
}