import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function mountBar(extraProps: Record<string, unknown> = {}) {
  return mount(AccountBulkActionsBar, {
    props: {
      selectedCount: 20,
      filteredSelectionActive: false,
      canSelectAllMatching: false,
      matchingCount: 120,
      editLoading: false,
      ...extraProps
    }
  })
}

describe('AccountBulkActionsBar', () => {
  it('shows select-all-matching entry and emits the action', async () => {
    const wrapper = mountBar({ canSelectAllMatching: true })

    expect(wrapper.text()).toContain('admin.accounts.bulkActions.selected')
    expect(wrapper.text()).toContain('admin.accounts.bulkActions.selectCurrentPage')
    expect(wrapper.text()).toContain('admin.accounts.bulkActions.selectAllMatching')

    const selectAllMatchingButton = wrapper
      .findAll('button')
      .find(button => button.text() === 'admin.accounts.bulkActions.selectAllMatching')

    expect(selectAllMatchingButton).toBeTruthy()
    await selectAllMatchingButton!.trigger('click')

    expect(wrapper.emitted('select-all-matching')).toHaveLength(1)
  })

  it('hides non-edit bulk actions in filtered-selection mode', () => {
    const wrapper = mountBar({
      filteredSelectionActive: true,
      canSelectAllMatching: true
    })

    expect(wrapper.text()).toContain('admin.accounts.bulkActions.selectedMatching')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.selectCurrentPage')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.selectAllMatching')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.delete')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.resetStatus')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.refreshToken')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.enableScheduling')
    expect(wrapper.text()).not.toContain('admin.accounts.bulkActions.disableScheduling')
    expect(wrapper.text()).toContain('admin.accounts.bulkActions.edit')
    expect(wrapper.text()).toContain('admin.accounts.bulkActions.clear')
  })
})
