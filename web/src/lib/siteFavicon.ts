import { ref, watch, type Ref } from 'vue'

export function createSiteFaviconFailureState(
  siteId: () => string,
  refreshKey: () => number,
): Ref<boolean> {
  const failed = ref(false)
  watch([siteId, refreshKey], () => {
    failed.value = false
  })
  return failed
}
