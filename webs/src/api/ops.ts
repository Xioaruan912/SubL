import request from '@/utils/request'

export const getApiTokens = () => request({ url:'/api/v1/tokens/list', method:'get' })
export const createApiToken = (data:any) => request({ url:'/api/v1/tokens/create', method:'post', data })
export const revokeApiToken = (id:number) => request({ url:'/api/v1/tokens/revoke', method:'post', params:{ id } })
export const inspectSafeBackup = () => request({ url:'/api/v1/ops/backup/inspect', method:'get' })
export const exportSafeBackup = () => request({ url:'/api/v1/ops/backup/export', method:'get' })
export const importSafeBackup = (data:any) => request({ url:'/api/v1/ops/backup/import', method:'post', data })
