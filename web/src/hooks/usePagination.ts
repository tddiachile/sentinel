import { useState } from 'react'

interface PaginationState {
  page: number
  pageSize: number
}

export function usePagination(initialPageSize = 20) {
  const [pagination, setPagination] = useState<PaginationState>({
    page: 1,
    pageSize: initialPageSize,
  })

  const goToPage = (page: number) => {
    setPagination((prev) => ({ ...prev, page: Math.max(1, page) }))
  }

  const reset = () => {
    setPagination({ page: 1, pageSize: initialPageSize })
  }

  return {
    page: pagination.page,
    pageSize: pagination.pageSize,
    goToPage,
    reset,
  }
}
