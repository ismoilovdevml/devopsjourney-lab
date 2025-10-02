import { ref } from 'vue'
import axios, { type AxiosRequestConfig, type CancelTokenSource } from 'axios'

export interface UseApiOptions extends AxiosRequestConfig {
  maxRetries?: number
  retryDelay?: number
  retryableStatuses?: number[]
}

export function useApi<T>(url: string, options: UseApiOptions = {}) {
  const data = ref<T | null>(null)
  const error = ref<Error | null>(null)
  const loading = ref(false)

  const {
    maxRetries = 3,
    retryDelay = 1000,
    retryableStatuses = [408, 429, 500, 502, 503, 504],
    ...axiosOptions
  } = options

  let cancelToken: CancelTokenSource | null = null
  let retryCount = 0

  const execute = async (): Promise<T | undefined> => {
    loading.value = true
    error.value = null

    // Cancel previous request
    if (cancelToken) {
      cancelToken.cancel('New request initiated')
    }

    cancelToken = axios.CancelToken.source()

    try {
      const response = await axios.request<T>({
        ...axiosOptions,
        url,
        cancelToken: cancelToken.token,
      })

      data.value = response.data
      retryCount = 0
      return response.data
    } catch (err: any) {
      // Check if request was cancelled
      if (axios.isCancel(err)) {
        console.log('Request canceled:', err.message)
        return
      }

      // Retry logic for retryable errors
      const shouldRetry =
        retryCount < maxRetries &&
        (retryableStatuses.includes(err.response?.status) ||
          err.code === 'ECONNABORTED' ||
          err.message === 'Network Error')

      if (shouldRetry) {
        retryCount++
        const delay = retryDelay * Math.pow(2, retryCount - 1) // Exponential backoff

        console.log(
          `Retrying request (${retryCount}/${maxRetries}) after ${delay}ms...`
        )

        await new Promise((resolve) => setTimeout(resolve, delay))
        return execute()
      }

      error.value = err
      throw err
    } finally {
      loading.value = false
    }
  }

  const cancel = () => {
    if (cancelToken) {
      cancelToken.cancel('Request cancelled by user')
    }
  }

  const reset = () => {
    data.value = null
    error.value = null
    loading.value = false
    retryCount = 0
  }

  return {
    data,
    error,
    loading,
    execute,
    cancel,
    reset,
  }
}
