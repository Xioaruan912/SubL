import request from '@/utils/request'

export function getTasks(limit = 100){ return request({ url:'/api/v1/tasks/list', method:'get', params:{ limit } }) }
export function cancelTask(id:number){ return request({ url:'/api/v1/tasks/cancel', method:'post', params:{ id } }) }
export function retryTask(id:number){ return request({ url:'/api/v1/tasks/retry', method:'post', params:{ id } }) }
