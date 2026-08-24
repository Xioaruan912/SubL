import request from "@/utils/request";
export function getSubTotal(){
  return request({
    url: "/api/v1/total/sub",
    method: "get",
  });
}
export function getNodeTotal(){
    return request({
      url: "/api/v1/total/node",
      method: "get",
    });
  }
export function getNodeMap(){
  return request({
    url: "/api/v1/nodes/map",
    method: "get",
  });
}
export function getNodePing(){
  return request({
    url: "/api/v1/nodes/ping",
    method: "get",
  });
}