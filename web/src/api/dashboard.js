import request from './request'

export function getServerInfo() {
  return request.get('/dashboard/server-info')
}
