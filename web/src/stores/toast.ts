import { readonly, ref } from 'vue'

export type ToastTone = 'success' | 'danger' | 'info'

export interface Toast {
  id: number
  title: string
  message?: string
  tone: ToastTone
}

const items = ref<Toast[]>([])
let nextId = 1

function remove(id: number): void {
  items.value = items.value.filter((item) => item.id !== id)
}

function show(title: string, options: { message?: string; tone?: ToastTone; duration?: number } = {}): void {
  const item: Toast = {
    id: nextId++,
    title,
    message: options.message,
    tone: options.tone || 'info',
  }
  items.value.push(item)
  window.setTimeout(() => remove(item.id), options.duration || 4200)
}

export function useToast() {
  return {
    items: readonly(items),
    show,
    success: (title: string, message?: string) => show(title, { message, tone: 'success' }),
    danger: (title: string, message?: string) => show(title, { message, tone: 'danger', duration: 6500 }),
    remove,
  }
}
