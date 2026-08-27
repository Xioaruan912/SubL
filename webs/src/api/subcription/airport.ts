import request from "@/utils/request";

export function getAirports() {
  return request({
    url: "/api/v1/airport/list",
    method: "get",
  });
}

export function getAirportDetail(id: number) {
  return request({
    url: "/api/v1/airport/detail",
    method: "get",
    params: { id },
  });
}

export function AddAirport(data: any) {
  return request({
    url: "/api/v1/airport/add",
    method: "post",
    data,
  });
}

export function DelAirport(data: any) {
  return request({
    url: "/api/v1/airport/delete",
    method: "delete",
    params: data,
  });
}

export function UpdateAirport(data: any) {
  return request({
    url: "/api/v1/airport/update",
    method: "post",
    data,
  });
}

export function SyncAirport(data: any) {
  return request({
    url: "/api/v1/airport/sync",
    method: "post",
    data,
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
  });
}
