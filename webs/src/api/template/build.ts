import request from "@/utils/request";

// 通过表单构建 clash 模板并保存
export function BuildClashTemplate(data: any){
  return request({
    url: "/api/v1/template/build",
    method: "post",
    data,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

// 获取默认 mihomo 配置（用于预填表单）
export function GetDefaultTemplate(){
  return request({
    url: "/api/v1/template/default",
    method: "get",
  });
}