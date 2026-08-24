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