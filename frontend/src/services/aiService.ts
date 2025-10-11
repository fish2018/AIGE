import api from '@/utils/api'

export interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface ChatRequest {
  provider_id: number
  model_id: string
  messages: Message[]
  stream: boolean
}

export interface ChatResponse {
  content: string
  finishReason?: string
}

export interface TestConnectionRequest {
  provider_id: number
  model_id: string
  api_type?: string
}

export interface TestConnectionResponse {
  success: boolean
  message: string
  result?: any
}

export class AIService {
  static async chat(request: ChatRequest): Promise<ChatResponse> {
    return await api.post<ChatResponse>('/admin/ai/chat', request)
  }

  static async chatStream(
    request: ChatRequest,
    onChunk: (content: string) => void,
    onError: (error: Error) => void,
    onComplete: () => void
  ): Promise<void> {
    //console.log('[AIService] chatStream 开始, request:', request)
    
    try {
      const token = localStorage.getItem('token')
      const baseURL = import.meta.env.VITE_API_BASE_URL || 'https://game.252035.xyz'
      
      //console.log('[AIService] 准备发送请求到:', `${baseURL}/api/admin/ai/chat`)
      
      const response = await fetch(`${baseURL}/api/admin/ai/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ ...request, stream: true })
      })

      //console.log('[AIService] 收到响应, status:', response.status, 'ok:', response.ok)

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`HTTP ${response.status}: ${errorText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('无法获取响应流')
      }

      //console.log('[AIService] 开始读取流数据...')

      const decoder = new TextDecoder()
      let buffer = ''
      let chunkCount = 0

      while (true) {
        const { done, value } = await reader.read()
        
        if (done) {
          //console.log('[AIService] 流读取完成, 总共处理', chunkCount, '个chunk')
          onComplete()
          break
        }

        const decoded = decoder.decode(value, { stream: true })
        //console.log('[AIService] 收到原始数据:', decoded)
        
        buffer += decoded
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        //console.log('[AIService] 分割后的行数:', lines.length)

        for (const line of lines) {
          //console.log('[AIService] 处理行:', JSON.stringify(line))
          
          if (!line.trim()) {
            //console.log('[AIService] 跳过空行')
            continue
          }
          
          if (line.startsWith('event:')) {
            //console.log('[AIService] 跳过event行')
            continue
          }
          
          if (line.startsWith('data:')) {
            const data = line.slice(5).trim()
            //console.log('[AIService] 提取data:', data)
            
            if (!data) {
              //console.log('[AIService] data为空，跳过')
              continue
            }
            
            try {
              const chunk = JSON.parse(data)
              //console.log('[AIService] 解析chunk成功:', chunk)
              
              // 先处理content（即使done为true，也可能包含最后的内容）
              if (chunk.content !== undefined && chunk.content !== '') {
                chunkCount++
                //console.log('[AIService] 调用onChunk, content:', JSON.stringify(chunk.content))
                onChunk(chunk.content)
              }
              
              // 再检查是否结束
              if (chunk.done) {
                //console.log('[AIService] 收到done信号，结束，总共处理', chunkCount, '个chunk')
                onComplete()
                return
              }
            } catch (e) {
              console.error('[AIService] 解析SSE数据失败:', e, data)
            }
          } else {
            //console.log('[AIService] 行不以data:开头:', line)
          }
        }
      }
    } catch (error) {
      console.error('[AIService] chatStream错误:', error)
      onError(error as Error)
    }
  }

  static async testConnection(request: TestConnectionRequest): Promise<TestConnectionResponse> {
    return await api.post<TestConnectionResponse>('/admin/ai/test', request)
  }
}
