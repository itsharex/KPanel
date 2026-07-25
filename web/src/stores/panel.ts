import { computed, reactive } from 'vue'
import type { AgentStatus } from '@/types/api'

interface PanelState {
  agent?: AgentStatus
  lastOverviewAt?: string
}

const state = reactive<PanelState>({})

function setAgent(agent?: AgentStatus): void {
  state.agent = agent
}

export function usePanelState() {
  return {
    state,
    setAgent,
    isOffline: computed(() => state.agent?.connected === false),
    isReadOnly: computed(
      () => Boolean(state.agent && (!state.agent.connected || state.agent.readOnly || !state.agent.compatible)),
    ),
  }
}
