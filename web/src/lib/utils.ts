import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { format, formatDistanceToNow, parseISO } from 'date-fns'
import { es } from 'date-fns/locale'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '-'
  try {
    return format(parseISO(dateStr), 'dd/MM/yyyy HH:mm', { locale: es })
  } catch {
    return dateStr
  }
}

export function formatDateRelative(dateStr: string | null | undefined): string {
  if (!dateStr) return '-'
  try {
    return formatDistanceToNow(parseISO(dateStr), { addSuffix: true, locale: es })
  } catch {
    return dateStr
  }
}

export function getErrorMessage(code: string): string {
  const messages: Record<string, string> = {
    INVALID_CREDENTIALS: 'Usuario o contrasena incorrectos.',
    ACCOUNT_LOCKED: 'La cuenta esta bloqueada por multiples intentos fallidos.',
    ACCOUNT_INACTIVE: 'La cuenta esta desactivada. Contacte al administrador.',
    APPLICATION_NOT_FOUND: 'Clave de aplicacion invalida.',
    TOKEN_INVALID: 'Sesion invalida. Por favor inicie sesion nuevamente.',
    TOKEN_EXPIRED: 'La sesion ha expirado. Por favor inicie sesion nuevamente.',
    TOKEN_REVOKED: 'La sesion fue revocada. Por favor inicie sesion nuevamente.',
    VALIDATION_ERROR: 'Los datos ingresados no son validos.',
    INVALID_CLIENT_TYPE: 'Tipo de cliente invalido.',
    FORBIDDEN: 'No tiene permisos para realizar esta operacion.',
    INTERNAL_ERROR: 'Error interno del servidor. Intente nuevamente.',
    PASSWORD_REUSED: 'La contrasena ya fue usada anteriormente (ultimas 5).',
  }
  return messages[code] ?? `Error: ${code}`
}

export function truncate(str: string, length: number): string {
  if (str.length <= length) return str
  return str.slice(0, length) + '...'
}
