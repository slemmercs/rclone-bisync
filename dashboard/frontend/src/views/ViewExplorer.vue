<template>
  <div class="space-y-4">
    <!-- Top Explorer Toolbar -->
    <ExplorerToolbar
      :isSwapped="isSwapped"
      :loading="leftLoading || rightLoading"
      :activeProfileId="activeProfile?.id"
      @swapPanes="swapPanes"
      @refreshAll="refreshAll"
      @openCompare="openCompareModal"
    />

    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-2xl overflow-hidden shadow-sm min-h-[580px]">
      <Splitter class="min-h-[580px] bg-carbon-900 text-carbon-100 border-none">
        <!-- Left Pane -->
        <SplitterPanel :size="50" :minSize="30" class="border-r border-carbon-700 relative">
          <ExplorerPane
            pane="left"
            :isLocal="leftIsLocal"
            :remote="leftRemote"
            :path="leftPath"
            :items="leftItems"
            :loading="leftLoading"
            :canary="leftCanary"
            :selectedItem="leftSelected"
            :uploading="leftUploading"
            :isRemoteLocked="isRemoteLocked"
            :remotes="remotesStore.remotes"
            :activeProfile="activeProfile"
            @update:remote="handleRemoteUpdate"
            @navigate="p => handleNavigate('left', p)"
            @refresh="() => refreshPane('left')"
            @touchCanary="() => touchCanary('left')"
            @mkdir="() => openMkdir('left')"
            @upDir="() => upDirectory('left')"
            @uploadFiles="files => handleUploadFiles('left', files)"
            @itemClick="item => handleItemClick('left', item)"
            @itemDblClick="item => handleItemDblClick('left', item)"
            @contextmenu="(e, item) => handleContextMenu(e, 'left', item)"
            @toggleCompare="item => toggleCompare('left', item)"
            @nameClick="item => handleNameClick('left', item)"
            @openOnRemote="item => openOnRemoteDirect(leftRemote, getRelativeItemPath('left', item))"
            @crossPaneDrop="payload => handleCrossPaneDrop(payload, 'left')"
          />
        </SplitterPanel>

        <!-- Right Pane -->
        <SplitterPanel :size="50" :minSize="30" class="relative">
          <ExplorerPane
            pane="right"
            :isLocal="rightIsLocal"
            :remote="rightRemote"
            :path="rightPath"
            :items="rightItems"
            :loading="rightLoading"
            :canary="rightCanary"
            :selectedItem="rightSelected"
            :uploading="rightUploading"
            :isRemoteLocked="isRemoteLocked"
            :remotes="remotesStore.remotes"
            :activeProfile="activeProfile"
            @update:remote="handleRemoteUpdate"
            @navigate="p => handleNavigate('right', p)"
            @refresh="() => refreshPane('right')"
            @touchCanary="() => touchCanary('right')"
            @mkdir="() => openMkdir('right')"
            @upDir="() => upDirectory('right')"
            @uploadFiles="files => handleUploadFiles('right', files)"
            @itemClick="item => handleItemClick('right', item)"
            @itemDblClick="item => handleItemDblClick('right', item)"
            @contextmenu="(e, item) => handleContextMenu(e, 'right', item)"
            @toggleCompare="item => toggleCompare('right', item)"
            @nameClick="item => handleNameClick('right', item)"
            @openOnRemote="item => openOnRemoteDirect(rightRemote, getRelativeItemPath('right', item))"
            @crossPaneDrop="payload => handleCrossPaneDrop(payload, 'right')"
          />
        </SplitterPanel>
      </Splitter>
    </div>

    <!-- Modals & Context Menus -->
    <ModalFilePreview
      v-model:visible="showPreviewModal"
      :preview="previewData"
      :loading="previewLoading"
      :error="previewError"
      :profileId="activeProfile?.id"
    />

    <ModalCompare
      v-model:visible="showCompareModal"
      :comparison="compareData"
      :loading="compareLoading"
      :error="compareError"
    />

    <ModalExplorerMkdir
      v-model:visible="showMkdirModal"
      :remote="mkdirTargetRemote"
      :currentPath="mkdirTargetRemotePath"
      :loading="mkdirLoading"
      :error="mkdirError"
      @create="handleCreateDirectory"
    />

    <ModalExplorerMove
      v-model:visible="showMoveModal"
      :remote="moveTargetRemote"
      :srcPath="moveSrcPath"
      :isDir="moveIsDir"
      :loading="moveLoading"
      :error="moveError"
      @move="handleExecuteMove"
    />

    <ModalExplorerTransfer
      v-model:visible="showTransferModal"
      :sourceRemote="transferSourceRemote"
      :sourcePath="transferSourcePath"
      :destRemote="transferDestRemote"
      :destPath="transferDestPath"
      :itemName="transferItemName"
      :isDir="transferIsDir"
      :loading="transferLoading"
      :error="transferError"
      @confirm="handleExecuteTransfer"
    />

    <ModalConfirm
      v-model:visible="showConfirmModal"
      :title="confirmTitle"
      :message="confirmMessage"
      :target="confirmTarget"
      :loading="confirmLoading"
      @confirm="handleExecuteDelete"
    />

    <ExplorerContextMenu
      :visible="contextMenuVisible"
      :x="contextMenuX"
      :y="contextMenuY"
      :item="contextMenuItem"
      :remote="contextMenuRemote"
      @close="contextMenuVisible = false"
      @action="handleContextAction"
    />

    <ModalUploadDock />
  </div>
</template>

<script setup lang="ts">
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import type { BrowseItem } from '../types/models'
import { isItemPreviewable } from '../types/explorer'
import ExplorerToolbar from '../components/explorer/ExplorerToolbar.vue'
import ExplorerPane from '../components/explorer/ExplorerPane.vue'
import ModalFilePreview from '../components/ModalFilePreview.vue'
import ModalCompare from '../components/ModalCompare.vue'
import ModalExplorerMkdir from '../components/ModalExplorerMkdir.vue'
import ModalExplorerMove from '../components/ModalExplorerMove.vue'
import ModalExplorerTransfer from '../components/explorer/ModalExplorerTransfer.vue'
import ModalConfirm from '../components/ModalConfirm.vue'
import ExplorerContextMenu from '../components/ExplorerContextMenu.vue'
import ModalUploadDock from '../components/ModalUploadDock.vue'
import { useExplorerPanes } from '../composables/useExplorerPanes'
import { useExplorerModals } from '../composables/useExplorerModals'

const panes = useExplorerPanes()
const {
  activeProfile,
  isRemoteLocked,
  remotesStore,
  isSwapped,
  leftPath,
  leftItems,
  leftLoading,
  leftCanary,
  leftSelected,
  leftUploading,
  rightPath,
  rightItems,
  rightLoading,
  rightCanary,
  rightSelected,
  rightUploading,
  leftIsLocal,
  rightIsLocal,
  leftRemote,
  rightRemote,
  swapPanes,
  refreshPane,
  refreshAll,
  handleRemoteUpdate,
  handleNavigate,
  getRelativeItemPath,
  upDirectory,
  touchCanary,
  handleUploadFiles,
} = panes

const modals = useExplorerModals({
  activeProfile,
  refreshAll,
  leftRemote,
  rightRemote,
  leftPath,
  rightPath,
  getRelativeItemPath,
  setSelectedItem: (pane, item) => {
    if (pane === 'left') leftSelected.value = item
    else rightSelected.value = item
  },
})

const {
  showPreviewModal,
  previewData,
  previewLoading,
  previewError,
  showCompareModal,
  compareData,
  compareLoading,
  compareError,
  showMkdirModal,
  mkdirTargetRemote,
  mkdirTargetRemotePath,
  mkdirLoading,
  mkdirError,
  showMoveModal,
  moveTargetRemote,
  moveSrcPath,
  moveIsDir,
  moveLoading,
  moveError,
  showTransferModal,
  transferSourceRemote,
  transferSourcePath,
  transferDestRemote,
  transferDestPath,
  transferItemName,
  transferIsDir,
  transferLoading,
  transferError,
  showConfirmModal,
  confirmTitle,
  confirmMessage,
  confirmTarget,
  confirmLoading,
  contextMenuVisible,
  contextMenuX,
  contextMenuY,
  contextMenuItem,
  contextMenuRemote,
  toggleCompare,
  openCompareModal,
  triggerPreview,
  openOnRemoteDirect,
  openMkdir,
  handleCreateDirectory,
  handleContextMenu,
  handleContextAction,
  handleExecuteMove,
  handleExecuteDelete,
  handleCrossPaneDrop,
  handleExecuteTransfer,
} = modals

function handleItemClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleNavigate(pane, getRelativeItemPath(pane, item))
  } else {
    if (pane === 'left') leftSelected.value = item
    else rightSelected.value = item
  }
}

function handleNameClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleItemClick(pane, item)
    return
  }
  if (pane === 'left') leftSelected.value = item
  else rightSelected.value = item
  if (isItemPreviewable(item)) {
    triggerPreview(pane === 'left' ? leftRemote.value : rightRemote.value, getRelativeItemPath(pane, item), item.Name)
  }
}

function handleItemDblClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleItemClick(pane, item)
  } else if (isItemPreviewable(item)) {
    triggerPreview(pane === 'left' ? leftRemote.value : rightRemote.value, getRelativeItemPath(pane, item), item.Name)
  }
}
</script>
