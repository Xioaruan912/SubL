import request from "@/utils/request";

export function getTemp(){
  return request({
    url: "/api/v1/template/get",
    method: "get",
  });
}

export function AddTemp(data: any){
  return request({
    url: "/api/v1/template/add",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function UpdateTemp(data: any){
  return request({
    url: "/api/v1/template/update",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}
export function DelTemp(data: any){
  return request({
    url: "/api/v1/template/delete",
    method: "post",
    data,
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

export function ValidateTemp(data: any){
  return request({ url: "/api/v1/template/validate", method: "post", data, headers: { "Content-Type": "multipart/form-data" } });
}
export function GetTempVersions(filename: string){
  return request({ url: "/api/v1/template/versions", method: "get", params: { filename } });
}
export function GetTempVersion(id: number){
  return request({ url: "/api/v1/template/version", method: "get", params: { id } });
}
export function RollbackTemp(data: any){
  return request({ url: "/api/v1/template/rollback", method: "post", data, headers: { "Content-Type": "multipart/form-data" } });
}
