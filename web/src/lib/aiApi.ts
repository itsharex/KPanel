import { apiRequest } from '@/lib/api'
import type { AIMessage, AIModel, AIProvider, AIRun, AISession, AIEvolutionProposal, AIMemory, AIProcedure } from '@/types/ai'

export const aiApi = {
  providers: {
    list: () => apiRequest<AIProvider[]>('/ai/providers'),
    create: (body: Partial<AIProvider> & { apiKey?: string }) => apiRequest<AIProvider>('/ai/providers',{method:'POST',body}),
    update: (id:string,body:Partial<AIProvider> & { apiKey?:string;expectedVersion:number }) => apiRequest<AIProvider>(`/ai/providers/${encodeURIComponent(id)}`,{method:'PATCH',body}),
    remove: (id:string) => apiRequest<{deleted:boolean}>(`/ai/providers/${encodeURIComponent(id)}`,{method:'DELETE'}),
    test: (id:string) => apiRequest<{ok:boolean}>(`/ai/providers/${encodeURIComponent(id)}/test`,{method:'POST'}),
    sync: (id:string) => apiRequest<AIModel[]>(`/ai/providers/${encodeURIComponent(id)}/models/sync`,{method:'POST'}),
    addModel: (id:string,body:Partial<AIModel>) => apiRequest<AIModel[]>(`/ai/providers/${encodeURIComponent(id)}/models`,{method:'POST',body}),
  },
  models: (providerId?:string) => apiRequest<AIModel[]>('/ai/models',{query:{providerId}}),
  sessions: {
    list: (search?:string,archived=false) => apiRequest<AISession[]>('/ai/sessions',{query:{search,archived}}),
    create: (providerId:string,modelId:string,title='') => apiRequest<AISession>('/ai/sessions',{method:'POST',body:{providerId,modelId,title}}),
    update: (id:string,body:Partial<AISession>) => apiRequest<AISession>(`/ai/sessions/${encodeURIComponent(id)}`,{method:'PATCH',body}),
    remove: (id:string) => apiRequest<{deleted:boolean}>(`/ai/sessions/${encodeURIComponent(id)}`,{method:'DELETE'}),
    messages: (id:string,cursor?:string) => apiRequest<{items:AIMessage[];nextCursor?:string}>(`/ai/sessions/${encodeURIComponent(id)}/messages`,{query:{cursor}}),
    send: (id:string,content:string) => apiRequest<{runId:string}>(`/ai/sessions/${encodeURIComponent(id)}/messages`,{method:'POST',body:{content}}),
  },
  runs: {
    get: (id:string) => apiRequest<AIRun>(`/ai/runs/${encodeURIComponent(id)}`),
    decision: (id:string,toolCallId:string,approve:boolean) => apiRequest<{accepted:boolean}>(`/ai/runs/${encodeURIComponent(id)}/decision`,{method:'POST',body:{toolCallId,approve}}),
    cancel: (id:string) => apiRequest<{cancelled:boolean}>(`/ai/runs/${encodeURIComponent(id)}/cancel`,{method:'POST'}),
    retry: (id:string) => apiRequest<{runId:string}>(`/ai/runs/${encodeURIComponent(id)}/retry`,{method:'POST'}),
    propose: (id:string) => apiRequest<{created:boolean}>(`/ai/runs/${encodeURIComponent(id)}/evolution`,{method:'POST'}),
  },
  evolution: {
    proposals: () => apiRequest<AIEvolutionProposal[]>('/ai/evolution/proposals'),
    approve: (id:string) => apiRequest<void>(`/ai/evolution/proposals/${encodeURIComponent(id)}/approve`,{method:'POST'}),
    reject: (id:string) => apiRequest<void>(`/ai/evolution/proposals/${encodeURIComponent(id)}/reject`,{method:'POST'}),
    memories: () => apiRequest<AIMemory[]>('/ai/memories'),
    procedures: () => apiRequest<AIProcedure[]>('/ai/procedures'),
	updateMemory: (id:string,body:Partial<AIMemory>) => apiRequest<AIMemory>(`/ai/memories/${encodeURIComponent(id)}`,{method:'PATCH',body}),
	removeMemory: (id:string) => apiRequest<void>(`/ai/memories/${encodeURIComponent(id)}`,{method:'DELETE'}),
	updateProcedure: (id:string,body:Partial<AIProcedure>&{rollbackToVersion?:number}) => apiRequest<AIProcedure>(`/ai/procedures/${encodeURIComponent(id)}`,{method:'PATCH',body}),
	removeProcedure: (id:string) => apiRequest<void>(`/ai/procedures/${encodeURIComponent(id)}`,{method:'DELETE'}),
  },
}

export function runEventURL(runId:string):string {
  const base=(import.meta.env.VITE_API_BASE_URL||'/api/v1').replace(/\/+$/,'')
  return `${base}/ai/runs/${encodeURIComponent(runId)}/events`
}
