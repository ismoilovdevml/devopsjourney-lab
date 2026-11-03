import { ApiError, errorMessage, isApiStatus } from './api'

/** Uzbek message for a failed lab start/recreate, covering the documented 409/429/503 cases. */
export function sessionStartMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.status === 409 && err.body?.error === 'Task unavailable') {
    return "Bu vazifa ushbu serverda o'chirilgan: u xavfsiz sandbox (gVisor) talab qiladi."
  }
  if (isApiStatus(err, 409)) return 'Bu vazifa hali qulflangan. Avval oldingi vazifalarni yakunlang.'
  if (isApiStatus(err, 429)) return "Laboratoriya juda tez-tez ishga tushirildi. Bir daqiqadan so'ng qayta urinib ko'ring."
  if (isApiStatus(err, 503)) return "Laboratoriyada bo'sh joy qolmadi. Bir necha daqiqadan so'ng qayta urinib ko'ring."
  return errorMessage(err, fallback)
}
