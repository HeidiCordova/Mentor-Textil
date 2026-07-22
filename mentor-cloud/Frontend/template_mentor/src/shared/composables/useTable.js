import { ref, computed } from 'vue'

export function useTable(options = {}) {
  const items = ref([])
  const currentPage = ref(1)
  const perPage = ref(options.perPage || 10)
  const total = ref(0)
  const sortBy = ref(options.sortBy || null)
  const sortOrder = ref(options.sortOrder || 'asc')
  const searchQuery = ref('')

  const totalPages = computed(() => Math.ceil(total.value / perPage.value))
  const hasNextPage = computed(() => currentPage.value < totalPages.value)
  const hasPrevPage = computed(() => currentPage.value > 1)

  function setItems(data, totalCount) {
    items.value = data
    total.value = totalCount || data.length
  }

  function setPage(page) {
    if (page >= 1 && page <= totalPages.value) {
      currentPage.value = page
    }
  }

  function nextPage() {
    if (hasNextPage.value) {
      currentPage.value++
    }
  }

  function prevPage() {
    if (hasPrevPage.value) {
      currentPage.value--
    }
  }

  function sort(column) {
    if (sortBy.value === column) {
      sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortBy.value = column
      sortOrder.value = 'asc'
    }
  }

  function search(query) {
    searchQuery.value = query
    currentPage.value = 1
  }

  function reset() {
    currentPage.value = 1
    sortBy.value = options.sortBy || null
    sortOrder.value = options.sortOrder || 'asc'
    searchQuery.value = ''
  }

  return {
    items,
    currentPage,
    perPage,
    total,
    totalPages,
    hasNextPage,
    hasPrevPage,
    sortBy,
    sortOrder,
    searchQuery,
    setItems,
    setPage,
    nextPage,
    prevPage,
    sort,
    search,
    reset
  }
}
