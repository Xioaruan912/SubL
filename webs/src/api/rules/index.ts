import request from "@/utils/request";

export const getRuleSources = () => request({ url: "/api/v1/rules/sources", method: "get" });

export const getRuleCatalog = (params: any) => request({
  url: "/api/v1/rules/catalog",
  method: "get",
  params,
});

export const getRulePreview = (id: string) => request({
  url: "/api/v1/rules/preview",
  method: "get",
  params: { id },
});
export const getRuleUpdateImpact = (id:string) => request({ url:'/api/v1/rules/update-impact', method:'get', params:{ id } });
export const applyRuleUpdate = (id:string) => request({ url:'/api/v1/rules/apply-update', method:'post', params:{ id } });
export const getRuleSnapshots = (id:string) => request({ url:'/api/v1/rules/snapshots', method:'get', params:{ id } });
export const rollbackRule = (id:string, snapshotId:number) => request({ url:'/api/v1/rules/rollback', method:'post', params:{ id, snapshotId } });

export const getRuleTemplateGroups = (template: string) => request({
  url: "/api/v1/rules/template-groups",
  method: "get",
  params: { template },
});

export function syncRuleCatalog(source = "") {
  const data = new FormData();
  if (source) data.append("source", source);
  return request({
    url: "/api/v1/rules/sync",
    method: "post",
    data,
    headers: { "Content-Type": "multipart/form-data" },
  });
}

export const applyRulesToTemplate = (data: any) => request({
  url: "/api/v1/rules/import",
  method: "post",
  data,
});
