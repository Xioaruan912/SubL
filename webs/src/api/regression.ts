import request from '@/utils/request'

export const getRegressionCases = () => request({ url:'/api/v1/regressions/list', method:'get' })
export const saveRegressionCase = (data:any) => request({ url:'/api/v1/regressions/save', method:'post', data })
export const deleteRegressionCase = (id:number) => request({ url:'/api/v1/regressions/delete', method:'delete', params:{ id } })
export const evaluateRegression = (data:any) => request({ url:'/api/v1/regressions/evaluate', method:'post', data, headers:{ 'Content-Type':'multipart/form-data' } })
export const compareRegression = (data:any) => request({ url:'/api/v1/regressions/compare', method:'post', data, headers:{ 'Content-Type':'multipart/form-data' } })
