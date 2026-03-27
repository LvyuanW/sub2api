<template>
  <div v-if="selectedCount > 0" class="mb-4 flex items-center justify-between p-3 bg-primary-50 rounded-lg dark:bg-primary-900/20">
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{
          filteredSelectionActive
            ? t('admin.accounts.bulkActions.selectedMatching', { count: selectedCount })
            : t('admin.accounts.bulkActions.selected', { count: selectedCount })
        }}
      </span>
      <button
        v-if="!filteredSelectionActive"
        @click="$emit('select-page')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span v-if="!filteredSelectionActive && canSelectAllMatching" class="text-gray-300 dark:text-primary-800">•</span>
      <button
        v-if="!filteredSelectionActive && canSelectAllMatching"
        @click="$emit('select-all-matching')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectAllMatching', { count: matchingCount }) }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">•</span>
      <button
        @click="$emit('clear')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
    </div>
    <div class="flex gap-2">
      <template v-if="!filteredSelectionActive">
        <button @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
        <button @click="$emit('reset-status')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button @click="$emit('refresh-token')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
      </template>
      <button @click="$emit('edit')" :disabled="editLoading" class="btn btn-primary btn-sm disabled:cursor-not-allowed disabled:opacity-60">
        {{ t('admin.accounts.bulkActions.edit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps({
  selectedCount: { type: Number, required: true },
  filteredSelectionActive: { type: Boolean, default: false },
  canSelectAllMatching: { type: Boolean, default: false },
  matchingCount: { type: Number, default: 0 },
  editLoading: { type: Boolean, default: false }
})

defineEmits(['delete', 'edit', 'clear', 'select-page', 'select-all-matching', 'toggle-schedulable', 'reset-status', 'refresh-token'])

const { t } = useI18n()
</script>
