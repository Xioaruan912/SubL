import request from "@/utils/request";

export function getCollections() {
  return request({ url: "/api/v1/collections/list", method: "get" });
}
export function addCollection(data: any) {
  return request({ url: "/api/v1/collections/add", method: "post", data });
}
export function updateCollection(data: any) {
  return request({ url: "/api/v1/collections/update", method: "post", data });
}
export function deleteCollection(id: number) {
  return request({ url: "/api/v1/collections/delete", method: "delete", params: { id } });
}
export function resetCollectionToken(id: number) {
  return request({
    url: "/api/v1/collections/reset-token",
    method: "post",
    data: new URLSearchParams({ id: String(id) }),
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
  });
}
