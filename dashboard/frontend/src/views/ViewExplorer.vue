<template>
  <div class="space-y-4">
    <!-- Header Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3 bg-carbon-900 p-3.5 rounded-2xl border border-carbon-700 shadow-md">
      <div class="flex items-center gap-3">
        <div class="h-8 w-8 rounded-lg bg-carbon-800 border border-carbon-700 flex items-center justify-center text-electric-400">
          <i class="pi pi-folder text-sm"></i>
        </div>
        <div>
          <h2 class="text-sm font-bold text-white tracking-tight flex items-center gap-2">
            <span>Dual-Pane Directory Explorer</span>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-800 text-carbon-400 border border-carbon-700">
              {{ isSwapped ? 'Remote ⇄ Local' : 'Local ⇄ Remote' }}
            </span>
          </h2>
          <p class="text-[11px] text-carbon-400">Direct side-by-side inspection, multi-file checksum comparisons, and cross-backend operations.</p>
        </div>
      </div>

      <!-- Global Explorer Action Buttons -->
      <div class="flex flex-wrap items-center gap-2">
        <!-- Compare Selection Pill Button -->
        <div v-if="explorerStore.compareSelection.length > 0" class="flex items-center gap-1.5 bg-carbon-950 p-1 rounded-xl border border-carbon-700">
          <button
            @click="openCompareModal"
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-all flex items-center gap-1.5 cursor-pointer shadow-sm"
            :class="explorerStore.compareSelection.length >= 2
              ? 'bg-indigo-600 hover:bg-indigo-500 text-white animate-pulse'
              : 'bg-carbon-800 hover:bg-carbon-700 text-carbon-300'"
            :title="explorerStore.compareSelection.length >= 2 ? 'Run cryptographic checksum comparison' : 'Select at least 2 files to compare'"
          >
            <i class="pi pi-sliders-v text-xs"></i>
            <span>Compare ({{ explorerStore.compareSelection.length }})</span>
          </button>
          <button
            @click="explorerStore.clearCompareSelection()"
            class="px-2 py-1 bg-carbon-800 hover:bg-carbon-700 text-carbon-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
            title="Clear compare selection"
          >
            ✕
          </button>
        </div>

        <!-- Upload Dock Launcher Button -->
        <button
          @click="explorerStore.openUploadDockModal(activeProfile?.id)"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          :title="'Upload Dock: ' + (explorerStore.uploadDock.pending_count || 0) + ' pending'"
        >
          <i class="pi pi-upload text-xs text-electric-400"></i>
          <span>Uploads ({{ explorerStore.uploadDock.pending_count || 0 }})</span>
          <span
            v-if="explorerStore.uploadDock.is_sync_active"
            class="text-[9px] px-1 py-0.2 rounded bg-amber-950 text-amber-400 border border-amber-700 font-mono uppercase"
          >
            Deferred
          </span>
        </button>

        <!-- Swap Panes Button -->
        <button
          @click="swapPanes"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Swap Left and Right Panes"
        >
          <i class="pi pi-arrows-h text-xs text-electric-400"></i>
          <span>Swap Panes</span>
        </button>

        <!-- Refresh Explorer Button -->
        <button
          @click="refreshAll"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Refresh both storage panes"
        >
          <i :class="(leftLoading || rightLoading) ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
          <span>Refresh</span>
        </button>

        <!-- Toggle Log Dock Button -->
        <button
          @click="logsStore.toggleDock()"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          :class="logsStore.isDockOpen ? 'text-electric-400 border-electric-500' : ''"
          title="Toggle Streaming Log Dock"
        >
          <i class="pi pi-terminal text-xs"></i>
          <span>Log Dock</span>
        </button>
      </div>
    </div>

    <!-- Hidden File Upload Inputs -->
    <input type="file" ref="leftFileInput" class="hidden" @change="handleFileUpload('left', $event)" />
    <input type="file" ref="rightFileInput" class="hidden" @change="handleFileUpload('right', $event)" />

    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-2xl overflow-hidden shadow-sm min-h-[580px]">
      <Splitter class="min-h-[580px] bg-carbon-900 text-carbon-100 border-none">
        <!-- Left Pane -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col border-r border-carbon-700">
          <!-- Pane Header -->
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i :class="leftIsLocal ? 'pi pi-desktop text-indigo-400' : 'pi pi-cloud text-emerald-400'"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">
                {{ leftIsLocal ? 'Local Storage' : leftRemote }}
              </span>
              <!-- Canary Guard Pill -->
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="leftCanary ? 'bg-emerald-950 text-emerald-400 border border-emerald-700' : 'bg-amber-950 text-amber-400 border border-amber-700'"
                :title="leftCanary ? 'Canary Active: RCLONE_TEST verified' : 'Canary Missing: Click Touch Canary to provision'"
              >
                <i :class="leftCanary ? 'pi pi-check' : 'pi pi-exclamation-triangle'" class="text-[9px]"></i>
                {{ leftCanary ? 'Canary OK' : 'No Canary' }}
              </span>
              <button
                v-if="!leftCanary"
                @click="touchCanary('left')"
                class="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/60 hover:bg-amber-800 text-amber-200 border border-amber-700 cursor-pointer"
                title="Create RCLONE_TEST canary file"
              >
                Touch Canary
              </button>
            </div>

            <!-- Pane Action Icons -->
            <div class="flex items-center gap-1">
              <button
                @click="triggerUploadPicker('left')"
                :disabled="leftUploading"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Upload file to this directory"
              >
                <i :class="leftUploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" class="text-xs"></i>
              </button>
              <button
                @click="openMkdir('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Create new folder"
              >
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button
                @click="upDirectory('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Navigate up"
              >
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button
                @click="refreshPane('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Refresh directory"
              >
                <i :class="leftLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Left Breadcrumb Location Bar -->
          <div class="mb-2 bg-carbon-950/80 px-3 py-1.5 rounded-xl border border-carbon-800 flex items-center justify-between gap-2 text-xs font-mono text-slate-300 min-w-0 shrink-0">
            <div class="flex items-center gap-1 min-w-0 overflow-x-auto py-0.5 no-scrollbar flex-1" :title="getDisplayPath('left')">
              <i class="pi pi-folder text-slate-500 text-xs shrink-0 mr-1"></i>
              <template v-for="(crumb, idx) in getBreadcrumbs('left')" :key="crumb.path + crumb.name">
                <div class="flex items-center shrink-0 min-w-0">
                  <span v-if="idx > 0" class="text-carbon-600 px-1 select-none">/</span>
                  <button
                    type="button"
                    @click="navigateToCrumb('left', crumb)"
                    class="transition-colors truncate max-w-[140px] sm:max-w-[200px] cursor-pointer"
                    :class="[
                      idx === getBreadcrumbs('left').length - 1 ? 'text-electric-400 font-bold' : 'text-slate-400 hover:text-white hover:underline',
                      crumb.isEllipsis ? 'text-slate-500 hover:text-slate-300 font-bold px-0.5' : '',
                    ]"
                    :title="crumb.title || crumb.name"
                  >
                    {{ crumb.name }}
                  </button>
                </div>
              </template>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="text-[10px] text-carbon-500 font-sans">{{ leftItems.length }} item(s)</span>
              <button
                type="button"
                @click="copyPanePath('left')"
                class="p-1 hover:bg-carbon-800 text-slate-400 hover:text-white rounded transition-colors cursor-pointer"
                :class="leftCopied ? 'text-emerald-400' : ''"
                :title="leftCopied ? 'Copied!' : 'Copy Path'"
              >
                <i :class="leftCopied ? 'pi pi-check text-xs' : 'pi pi-copy text-xs'"></i>
              </button>
            </div>
          </div>

          <!-- Left Column Sort & Filter Header -->
          <div class="px-3 py-1.5 bg-carbon-950/90 border-b border-carbon-800 text-[10px] uppercase font-bold text-slate-500 flex items-center justify-between gap-2 select-none tracking-wider shrink-0 font-sans rounded-t-lg">
            <div class="flex items-center gap-2 flex-1 min-w-0">
              <div class="relative w-28 sm:w-36 shrink-0">
                <input
                  v-model="leftSearch"
                  type="text"
                  placeholder="Filter..."
                  class="w-full bg-carbon-900 border border-carbon-800 rounded px-2 py-0.5 text-[10px] text-white placeholder-slate-500 focus:outline-none focus:border-electric-500 transition-colors font-sans normal-case"
                />
                <button
                  v-if="leftSearch"
                  @click="leftSearch = ''"
                  class="absolute inset-y-0 right-0 pr-1.5 flex items-center text-slate-400 hover:text-white text-[10px] cursor-pointer"
                >✕</button>
              </div>
              <span class="text-carbon-700">|</span>
              <button
                type="button"
                @click="toggleSort('left', 'name')"
                class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort alphabetically by Name"
              >
                <span>Name</span>
                <span v-if="leftSortBy === 'name'" class="text-electric-400">{{ leftSortAsc ? '▲' : '▼' }}</span>
              </button>
              <span class="text-carbon-700">|</span>
              <button
                type="button"
                @click="toggleSort('left', 'type')"
                class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by File Type"
              >
                <span>Type</span>
                <span v-if="leftSortBy === 'type'" class="text-electric-400">{{ leftSortAsc ? '▲' : '▼' }}</span>
              </button>
            </div>
            <div class="flex items-center gap-2.5 shrink-0">
              <button
                type="button"
                @click="toggleSort('left', 'mtime')"
                class="w-24 text-left hidden sm:flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by Last Modified Date"
              >
                <span>Modified</span>
                <span v-if="leftSortBy === 'mtime'" class="text-electric-400">{{ leftSortAsc ? '▲' : '▼' }}</span>
              </button>
              <button
                type="button"
                @click="toggleSort('left', 'size')"
                class="w-16 text-right flex items-center justify-end gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by File Size"
              >
                <span>Size</span>
                <span v-if="leftSortBy === 'size'" class="text-electric-400">{{ leftSortAsc ? '▲' : '▼' }}</span>
              </button>
              <div class="w-4 shrink-0"></div>
            </div>
          </div>

          <!-- Left Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-800/40 text-xs font-mono">
            <div v-if="leftLoading" class="py-16 text-center text-carbon-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-400 mb-2"></i>
              <div class="text-[11px]">Reading directory entries...</div>
            </div>
            <div v-else-if="getSortedItems('left').length === 0" class="py-16 text-center text-carbon-500">
              {{ leftSearch ? 'No matching items found.' : 'Directory is empty.' }}
            </div>
            <div
              v-else
              v-for="item in getSortedItems('left')"
              :key="item.Name"
              @click="handleItemClick('left', item)"
              @dblclick="handleItemDblClick('left', item)"
              @contextmenu.prevent="handleContextMenu($event, 'left', item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="[
                isRowSelected('left', item) ? 'bg-carbon-800 text-electric-400 font-semibold' : 'hover:bg-carbon-800/60 text-carbon-200',
                explorerStore.highlightName === item.Name ? 'ring-2 ring-electric-400 bg-electric-950/50 text-electric-300 font-bold animate-pulse' : ''
              ]"
            >
              <div class="flex items-center gap-2.5 truncate flex-1 min-w-0">
                <!-- Checkbox for multi-file comparison -->
                <input
                  v-if="!item.IsDir && item.Name !== 'RCLONE_TEST'"
                  type="checkbox"
                  :checked="isItemInCompare('left', item)"
                  @click.stop="toggleCompare('left', item)"
                  class="rounded bg-carbon-900 border-carbon-700 text-electric-500 cursor-pointer shrink-0"
                  title="Select for checksum comparison"
                />
                <span v-else class="w-3.5 shrink-0"></span>

                <i
                  class="text-xs shrink-0"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-carbon-400'"
                ></i>
                <span
                  class="truncate"
                  :class="isItemPreviewable(item) ? 'hover:text-electric-300 hover:underline' : ''"
                  @click.stop="handleNameClick('left', item)"
                >{{ item.Name }}</span>
              </div>

              <div class="flex items-center gap-2.5 text-[11px] text-carbon-400 shrink-0">
                <span v-if="item.ModTime" class="hidden sm:inline text-carbon-500 font-sans text-[10px] w-24 truncate">
                  {{ formatShortDate(item.ModTime) }}
                </span>
                <span v-if="!item.IsDir" class="w-16 text-right">{{ formatBytes(item.Size) }}</span>
                <span v-else class="w-16 text-right text-carbon-500 text-[10px]">DIR</span>

                <!-- Quick Open on Remote Icon for Cloud Remotes -->
                <button
                  v-if="!leftIsLocal && !item.IsDir"
                  @click.stop="openOnRemoteDirect(leftRemote, getRelativeItemPath('left', item))"
                  class="opacity-0 group-hover:opacity-100 hover:text-electric-400 p-0.5 transition-opacity cursor-pointer"
                  title="Open on remote in new tab"
                >
                  <i class="pi pi-external-link text-xs"></i>
                </button>
                <span v-else class="w-4 shrink-0"></span>
              </div>
            </div>
          </div>
        </SplitterPanel>

        <!-- Right Pane -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col">
          <!-- Pane Header -->
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i :class="rightIsLocal ? 'pi pi-desktop text-indigo-400' : 'pi pi-cloud text-emerald-400'"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">
                {{ rightIsLocal ? 'Local Storage' : rightRemote }}
              </span>
              <!-- Canary Guard Pill -->
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="rightCanary ? 'bg-emerald-950 text-emerald-400 border border-emerald-700' : 'bg-amber-950 text-amber-400 border border-amber-700'"
                :title="rightCanary ? 'Canary Active: RCLONE_TEST verified' : 'Canary Missing: Click Touch Canary to provision'"
              >
                <i :class="rightCanary ? 'pi pi-check' : 'pi pi-exclamation-triangle'" class="text-[9px]"></i>
                {{ rightCanary ? 'Canary OK' : 'No Canary' }}
              </span>
              <button
                v-if="!rightCanary"
                @click="touchCanary('right')"
                class="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/60 hover:bg-amber-800 text-amber-200 border border-amber-700 cursor-pointer"
                title="Create RCLONE_TEST canary file"
              >
                Touch Canary
              </button>
            </div>

            <!-- Pane Action Icons -->
            <div class="flex items-center gap-1">
              <button
                @click="triggerUploadPicker('right')"
                :disabled="rightUploading"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Upload file to this directory"
              >
                <i :class="rightUploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" class="text-xs"></i>
              </button>
              <button
                @click="openMkdir('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Create new folder"
              >
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button
                @click="upDirectory('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Navigate up"
              >
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button
                @click="refreshPane('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Refresh directory"
              >
                <i :class="rightLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Right Breadcrumb Location Bar -->
          <div class="mb-2 bg-carbon-950/80 px-3 py-1.5 rounded-xl border border-carbon-800 flex items-center justify-between gap-2 text-xs font-mono text-slate-300 min-w-0 shrink-0">
            <div class="flex items-center gap-1 min-w-0 overflow-x-auto py-0.5 no-scrollbar flex-1" :title="getDisplayPath('right')">
              <i class="pi pi-folder text-slate-500 text-xs shrink-0 mr-1"></i>
              <template v-for="(crumb, idx) in getBreadcrumbs('right')" :key="crumb.path + crumb.name">
                <div class="flex items-center shrink-0 min-w-0">
                  <span v-if="idx > 0" class="text-carbon-600 px-1 select-none">/</span>
                  <button
                    type="button"
                    @click="navigateToCrumb('right', crumb)"
                    class="transition-colors truncate max-w-[140px] sm:max-w-[200px] cursor-pointer"
                    :class="[
                      idx === getBreadcrumbs('right').length - 1 ? 'text-emerald-400 font-bold' : 'text-slate-400 hover:text-white hover:underline',
                      crumb.isEllipsis ? 'text-slate-500 hover:text-slate-300 font-bold px-0.5' : '',
                    ]"
                    :title="crumb.title || crumb.name"
                  >
                    {{ crumb.name }}
                  </button>
                </div>
              </template>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="text-[10px] text-carbon-500 font-sans">{{ rightItems.length }} item(s)</span>
              <button
                type="button"
                @click="copyPanePath('right')"
                class="p-1 hover:bg-carbon-800 text-slate-400 hover:text-white rounded transition-colors cursor-pointer"
                :class="rightCopied ? 'text-emerald-400' : ''"
                :title="rightCopied ? 'Copied!' : 'Copy Path'"
              >
                <i :class="rightCopied ? 'pi pi-check text-xs' : 'pi pi-copy text-xs'"></i>
              </button>
            </div>
          </div>

          <!-- Right Column Sort & Filter Header -->
          <div class="px-3 py-1.5 bg-carbon-950/90 border-b border-carbon-800 text-[10px] uppercase font-bold text-slate-500 flex items-center justify-between gap-2 select-none tracking-wider shrink-0 font-sans rounded-t-lg">
            <div class="flex items-center gap-2 flex-1 min-w-0">
              <div class="relative w-28 sm:w-36 shrink-0">
                <input
                  v-model="rightSearch"
                  type="text"
                  placeholder="Filter..."
                  class="w-full bg-carbon-900 border border-carbon-800 rounded px-2 py-0.5 text-[10px] text-white placeholder-slate-500 focus:outline-none focus:border-electric-500 transition-colors font-sans normal-case"
                />
                <button
                  v-if="rightSearch"
                  @click="rightSearch = ''"
                  class="absolute inset-y-0 right-0 pr-1.5 flex items-center text-slate-400 hover:text-white text-[10px] cursor-pointer"
                >✕</button>
              </div>
              <span class="text-carbon-700">|</span>
              <button
                type="button"
                @click="toggleSort('right', 'name')"
                class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort alphabetically by Name"
              >
                <span>Name</span>
                <span v-if="rightSortBy === 'name'" class="text-electric-400">{{ rightSortAsc ? '▲' : '▼' }}</span>
              </button>
              <span class="text-carbon-700">|</span>
              <button
                type="button"
                @click="toggleSort('right', 'type')"
                class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by File Type"
              >
                <span>Type</span>
                <span v-if="rightSortBy === 'type'" class="text-electric-400">{{ rightSortAsc ? '▲' : '▼' }}</span>
              </button>
            </div>
            <div class="flex items-center gap-2.5 shrink-0">
              <button
                type="button"
                @click="toggleSort('right', 'mtime')"
                class="w-24 text-left hidden sm:flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by Last Modified Date"
              >
                <span>Modified</span>
                <span v-if="rightSortBy === 'mtime'" class="text-electric-400">{{ rightSortAsc ? '▲' : '▼' }}</span>
              </button>
              <button
                type="button"
                @click="toggleSort('right', 'size')"
                class="w-16 text-right flex items-center justify-end gap-1 hover:text-slate-200 transition-colors cursor-pointer"
                title="Sort by File Size"
              >
                <span>Size</span>
                <span v-if="rightSortBy === 'size'" class="text-electric-400">{{ rightSortAsc ? '▲' : '▼' }}</span>
              </button>
              <div class="w-4 shrink-0"></div>
            </div>
          </div>

          <!-- Right Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-800/40 text-xs font-mono">
            <div v-if="rightLoading" class="py-16 text-center text-carbon-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-400 mb-2"></i>
              <div class="text-[11px]">Reading directory entries...</div>
            </div>
            <div v-else-if="getSortedItems('right').length === 0" class="py-16 text-center text-carbon-500">
              {{ rightSearch ? 'No matching items found.' : 'Directory is empty.' }}
            </div>
            <div
              v-else
              v-for="item in getSortedItems('right')"
              :key="item.Name"
              @click="handleItemClick('right', item)"
              @dblclick="handleItemDblClick('right', item)"
              @contextmenu.prevent="handleContextMenu($event, 'right', item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="[
                isRowSelected('right', item) ? 'bg-carbon-800 text-electric-400 font-semibold' : 'hover:bg-carbon-800/60 text-carbon-200',
                explorerStore.highlightName === item.Name ? 'ring-2 ring-electric-400 bg-electric-950/50 text-electric-300 font-bold animate-pulse' : ''
              ]"
            >
              <div class="flex items-center gap-2.5 truncate flex-1 min-w-0">
                <!-- Checkbox for multi-file comparison -->
                <input
                  v-if="!item.IsDir && item.Name !== 'RCLONE_TEST'"
                  type="checkbox"
                  :checked="isItemInCompare('right', item)"
                  @click.stop="toggleCompare('right', item)"
                  class="rounded bg-carbon-900 border-carbon-700 text-electric-500 cursor-pointer shrink-0"
                  title="Select for checksum comparison"
                />
                <span v-else class="w-3.5 shrink-0"></span>

                <i
                  class="text-xs shrink-0"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-carbon-400'"
                ></i>
                <span
                  class="truncate"
                  :class="isItemPreviewable(item) ? 'hover:text-electric-300 hover:underline' : ''"
                  @click.stop="handleNameClick('right', item)"
                >{{ item.Name }}</span>
              </div>

              <div class="flex items-center gap-2.5 text-[11px] text-carbon-400 shrink-0">
                <span v-if="item.ModTime" class="hidden sm:inline text-carbon-500 font-sans text-[10px] w-24 truncate">
                  {{ formatShortDate(item.ModTime) }}
                </span>
                <span v-if="!item.IsDir" class="w-16 text-right">{{ formatBytes(item.Size) }}</span>
                <span v-else class="w-16 text-right text-carbon-500 text-[10px]">DIR</span>

                <!-- Quick Open on Remote Icon for Cloud Remotes -->
                <button
                  v-if="!rightIsLocal && !item.IsDir"
                  @click.stop="openOnRemoteDirect(rightRemote, getRelativeItemPath('right', item))"
                  class="opacity-0 group-hover:opacity-100 hover:text-electric-400 p-0.5 transition-opacity cursor-pointer"
                  title="Open on remote in new tab"
                >
                  <i class="pi pi-external-link text-xs"></i>
                </button>
                <span v-else class="w-4 shrink-0"></span>
              </div>
            </div>
          </div>
        </SplitterPanel>
      </Splitter>
    </div>

    <!-- Modals -->
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
import { ref, computed, watch } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import { useProfilesStore } from '../stores/profiles'
import { useExplorerStore } from '../stores/explorer'
import { useLogsStore } from '../stores/logs'
import type { BrowseItem } from '../types/models'
import { isItemPreviewable, type FilePreviewResponse, type FileCompareResponse } from '../types/explorer'
import ModalFilePreview from '../components/ModalFilePreview.vue'
import ModalCompare from '../components/ModalCompare.vue'
import ModalExplorerMkdir from '../components/ModalExplorerMkdir.vue'
import ModalExplorerMove from '../components/ModalExplorerMove.vue'
import ModalConfirm from '../components/ModalConfirm.vue'
import ExplorerContextMenu from '../components/ExplorerContextMenu.vue'
import ModalUploadDock from '../components/ModalUploadDock.vue'

const profilesStore = useProfilesStore()
const explorerStore = useExplorerStore()
const logsStore = useLogsStore()

// Pane Swap State
const isSwapped = ref(false)

// Left Pane State (defaults to Local)
const leftPath = ref('')
const leftItems = ref<BrowseItem[]>([])
const leftLoading = ref(false)
const leftCanary = ref(false)
const leftSelected = ref<BrowseItem | null>(null)
const leftUploading = ref(false)
const leftFileInput = ref<HTMLInputElement | null>(null)

// Right Pane State (defaults to Remote)
const rightRemoteName = ref('')
const rightPath = ref('')
const rightItems = ref<BrowseItem[]>([])
const rightLoading = ref(false)
const rightCanary = ref(false)
const rightSelected = ref<BrowseItem | null>(null)
const rightUploading = ref(false)
const rightFileInput = ref<HTMLInputElement | null>(null)

type SortField = 'name' | 'type' | 'mtime' | 'size'

const leftSortBy = ref<SortField>('name')
const leftSortAsc = ref(true)
const leftSearch = ref('')

const rightSortBy = ref<SortField>('name')
const rightSortAsc = ref(true)
const rightSearch = ref('')

function toggleSort(pane: 'left' | 'right', field: SortField) {
  if (pane === 'left') {
    if (leftSortBy.value === field) {
      leftSortAsc.value = !leftSortAsc.value
    } else {
      leftSortBy.value = field
      leftSortAsc.value = (field === 'name' || field === 'type')
    }
  } else {
    if (rightSortBy.value === field) {
      rightSortAsc.value = !rightSortAsc.value
    } else {
      rightSortBy.value = field
      rightSortAsc.value = (field === 'name' || field === 'type')
    }
  }
}

function getSortedItems(pane: 'left' | 'right'): BrowseItem[] {
  const items = pane === 'left' ? leftItems.value : rightItems.value
  const query = (pane === 'left' ? leftSearch.value : rightSearch.value).toLowerCase().trim()
  const sortBy = pane === 'left' ? leftSortBy.value : rightSortBy.value
  const asc = pane === 'left' ? leftSortAsc.value : rightSortAsc.value

  let filtered = items
  if (query) {
    filtered = filtered.filter(item => (item.Name || '').toLowerCase().includes(query))
  }

  return [...filtered].sort((a, b) => {
    // Folders always sorted first
    if (a.IsDir !== b.IsDir) {
      return a.IsDir ? -1 : 1
    }

    let res = 0
    if (sortBy === 'size') {
      const sizeA = a.IsDir ? -1 : (a.Size ?? 0)
      const sizeB = b.IsDir ? -1 : (b.Size ?? 0)
      res = sizeA - sizeB
    } else if (sortBy === 'mtime') {
      const timeA = a.ModTime ? new Date(a.ModTime).getTime() : 0
      const timeB = b.ModTime ? new Date(b.ModTime).getTime() : 0
      res = timeA - timeB
    } else if (sortBy === 'type') {
      const extA = a.IsDir ? '' : (a.Name.split('.').pop() || '').toLowerCase()
      const extB = b.IsDir ? '' : (b.Name.split('.').pop() || '').toLowerCase()
      res = extA.localeCompare(extB)
      if (res === 0) {
        res = (a.Name || '').localeCompare(b.Name || '', undefined, { numeric: true, sensitivity: 'base' })
      }
    } else {
      res = (a.Name || '').localeCompare(b.Name || '', undefined, { numeric: true, sensitivity: 'base' })
    }

    return asc ? res : -res
  })
}

function formatShortDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

const activeProfile = computed(() => profilesStore.activeProfile)

const leftIsLocal = computed(() => !isSwapped.value)
const rightIsLocal = computed(() => isSwapped.value)

const leftRemote = computed(() => {
  return isSwapped.value ? rightRemoteName.value : 'local'
})

const rightRemote = computed(() => {
  return isSwapped.value ? 'local' : rightRemoteName.value
})

// Modal states
const showPreviewModal = ref(false)
const previewData = ref<FilePreviewResponse | null>(null)
const previewLoading = ref(false)
const previewError = ref<string | null>(null)

const showCompareModal = ref(false)
const compareData = ref<FileCompareResponse | null>(null)
const compareLoading = ref(false)
const compareError = ref<string | null>(null)

const showMkdirModal = ref(false)
const mkdirTargetRemote = ref('local')
const mkdirTargetRemotePath = ref('')
const mkdirLoading = ref(false)
const mkdirError = ref<string | null>(null)

const showMoveModal = ref(false)
const moveTargetRemote = ref('local')
const moveSrcPath = ref('')
const moveIsDir = ref(false)
const moveLoading = ref(false)
const moveError = ref<string | null>(null)

const showConfirmModal = ref(false)
const confirmTitle = ref('')
const confirmMessage = ref('')
const confirmTarget = ref('')
const deleteRemote = ref('local')
const deletePath = ref('')
const deleteIsDir = ref(false)
const confirmLoading = ref(false)

// Context Menu State
const contextMenuVisible = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuItem = ref<BrowseItem | null>(null)
const contextMenuRemote = ref('local')

// Watch active profile to load directories
watch(
  () => activeProfile.value,
  (profile) => {
    if (profile) {
      leftPath.value = ''
      rightRemoteName.value = (profile.remote || 'local').replace(/:$/, '')
      rightPath.value = (profile.remote_path || '').replace(/^\/+/, '')
      refreshAll()
      explorerStore.fetchUploadDockStatus(profile.id)
    }
  },
  { immediate: true }
)

function swapPanes() {
  isSwapped.value = !isSwapped.value
  const tempPath = leftPath.value
  leftPath.value = rightPath.value
  rightPath.value = tempPath

  const tempItems = leftItems.value
  leftItems.value = rightItems.value
  rightItems.value = tempItems

  const tempCanary = leftCanary.value
  leftCanary.value = rightCanary.value
  rightCanary.value = tempCanary
}

async function refreshPane(pane: 'left' | 'right') {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = pane === 'left' ? leftPath.value : rightPath.value
  const isLocal = pane === 'left' ? leftIsLocal.value : rightIsLocal.value

  if (pane === 'left') {
    leftLoading.value = true
    leftSelected.value = null
  } else {
    rightLoading.value = true
    rightSelected.value = null
  }

  try {
    await explorerStore.browse(remote, path, activeProfile.value?.id)
    if (pane === 'left') {
      leftItems.value = explorerStore.items
      if (!path) {
        leftCanary.value = explorerStore.hasCanary
      }
    } else {
      rightItems.value = explorerStore.items
      if (!path) {
        rightCanary.value = explorerStore.hasCanary
      }
    }
  } catch (e) {
    if (pane === 'left') leftItems.value = []
    else rightItems.value = []
  } finally {
    if (pane === 'left') leftLoading.value = false
    else rightLoading.value = false
  }
}

async function refreshAll() {
  await Promise.all([refreshPane('left'), refreshPane('right')])
}

function getDisplayPath(pane: 'left' | 'right'): string {
  const isLocal = pane === 'left' ? leftIsLocal.value : rightIsLocal.value
  const relPath = pane === 'left' ? leftPath.value : rightPath.value
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value

  if (isLocal) {
    const root = activeProfile.value?.local_path || 'local'
    return relPath ? `${root}/${relPath}` : root
  }
  return relPath ? `${remote}:${relPath}` : `${remote}:/`
}

interface BreadcrumbItem {
  name: string
  path: string
  title?: string
  isEllipsis?: boolean
}

function getBreadcrumbs(pane: 'left' | 'right'): BreadcrumbItem[] {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const isLocal = pane === 'left' ? leftIsLocal.value : rightIsLocal.value
  const currentPath = pane === 'left' ? leftPath.value : rightPath.value

  const rootLabel = isLocal ? '~' : `${remote}:`
  const allCrumbs: BreadcrumbItem[] = [{ name: rootLabel, path: '' }]
  if (!currentPath) return allCrumbs

  const parts = currentPath.split('/').filter(Boolean)
  let cumulative = ''
  for (const part of parts) {
    cumulative = cumulative ? `${cumulative}/${part}` : part
    allCrumbs.push({ name: part, path: cumulative })
  }

  if (allCrumbs.length <= 4) {
    return allCrumbs
  }

  // Middle collapse deep paths: root / … / parent / current
  const rootCrumb = allCrumbs[0]
  const parentCrumb = allCrumbs[allCrumbs.length - 2]
  const currentCrumb = allCrumbs[allCrumbs.length - 1]
  const middleCrumbs = allCrumbs.slice(1, allCrumbs.length - 2)
  const hiddenPath = middleCrumbs.map((c) => c.name).join('/')

  const ellipsisCrumb: BreadcrumbItem = {
    name: '…',
    path: middleCrumbs[middleCrumbs.length - 1].path,
    title: `Hidden folders: /${hiddenPath}`,
    isEllipsis: true,
  }

  return [rootCrumb, ellipsisCrumb, parentCrumb, currentCrumb]
}

function navigateToCrumb(pane: 'left' | 'right', crumb: BreadcrumbItem) {
  if (pane === 'left') {
    leftPath.value = crumb.path
    refreshPane('left')
  } else {
    rightPath.value = crumb.path
    refreshPane('right')
  }
}

const leftCopied = ref(false)
const rightCopied = ref(false)

function copyPanePath(pane: 'left' | 'right') {
  const path = getDisplayPath(pane)
  navigator.clipboard.writeText(path).then(() => {
    if (pane === 'left') {
      leftCopied.value = true
      setTimeout(() => {
        leftCopied.value = false
      }, 1500)
    } else {
      rightCopied.value = true
      setTimeout(() => {
        rightCopied.value = false
      }, 1500)
    }
  }).catch(() => {})
}

function getRelativeItemPath(pane: 'left' | 'right', item: BrowseItem): string {
  const currentPath = pane === 'left' ? leftPath.value : rightPath.value
  return currentPath ? `${currentPath.replace(/\/+$/, '')}/${item.Name}` : item.Name
}

function isRowSelected(pane: 'left' | 'right', item: BrowseItem): boolean {
  const selected = pane === 'left' ? leftSelected.value : rightSelected.value
  return selected?.Name === item.Name
}

function isItemInCompare(pane: 'left' | 'right', item: BrowseItem): boolean {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = getRelativeItemPath(pane, item)
  return explorerStore.isItemInCompare(remote, path)
}

function toggleCompare(pane: 'left' | 'right', item: BrowseItem) {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = getRelativeItemPath(pane, item)
  explorerStore.toggleCompareItem({
    remote,
    path,
    name: item.Name,
    size: item.Size || 0,
    pane,
  })
}

function handleItemClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    if (pane === 'left') {
      leftPath.value = getRelativeItemPath('left', item)
      refreshPane('left')
    } else {
      rightPath.value = getRelativeItemPath('right', item)
      refreshPane('right')
    }
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

function upDirectory(pane: 'left' | 'right') {
  const current = pane === 'left' ? leftPath.value : rightPath.value
  const parts = current.replace(/\/+$/, '').split('/').filter(Boolean)
  if (parts.length > 0) {
    parts.pop()
    const next = parts.join('/')
    if (pane === 'left') {
      leftPath.value = next
      refreshPane('left')
    } else {
      rightPath.value = next
      refreshPane('right')
    }
  }
}

async function touchCanary(pane: 'left' | 'right') {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  try {
    await explorerStore.touchCanary(remote, '', activeProfile.value?.id)
    if (pane === 'left') leftCanary.value = true
    else rightCanary.value = true
    refreshPane(pane)
  } catch (e: any) {
    alert(e.message || 'Failed to touch canary')
  }
}

function triggerUploadPicker(pane: 'left' | 'right') {
  if (pane === 'left' && leftFileInput.value) {
    leftFileInput.value.click()
  } else if (pane === 'right' && rightFileInput.value) {
    rightFileInput.value.click()
  }
}

async function handleFileUpload(pane: 'left' | 'right', e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = pane === 'left' ? leftPath.value : rightPath.value

  if (pane === 'left') leftUploading.value = true
  else rightUploading.value = true

  try {
    await explorerStore.uploadFile(file, remote, path, activeProfile.value?.id)
    refreshPane(pane)
    explorerStore.fetchUploadDockStatus(activeProfile.value?.id)
  } catch (err: any) {
    alert(err.message || 'File upload failed')
  } finally {
    if (pane === 'left') leftUploading.value = false
    else rightUploading.value = false
    target.value = ''
  }
}

async function openOnRemoteDirect(remote: string, path: string) {
  try {
    const data = await explorerStore.getPublicLink(remote, path, activeProfile.value?.id)
    if (data && data.supported && data.url) {
      window.open(data.url, '_blank')
    } else {
      alert('Cloud web links are unsupported for this storage provider.')
    }
  } catch (err: any) {
    alert(err.message || 'Failed to resolve cloud web link')
  }
}

async function triggerPreview(remote: string, path: string, name: string) {
  if (!isItemPreviewable({ Name: name, IsDir: false })) {
    return
  }
  previewLoading.value = true
  previewError.value = null
  previewData.value = null
  showPreviewModal.value = true

  try {
    previewData.value = await explorerStore.previewFile(remote, path, activeProfile.value?.id)
  } catch (e: any) {
    previewError.value = e.message || 'Failed to load preview'
  } finally {
    previewLoading.value = false
  }
}

function openCompareModal() {
  if (explorerStore.compareSelection.length < 2) return

  compareLoading.value = true
  compareError.value = null
  compareData.value = null
  showCompareModal.value = true

  const items = explorerStore.compareSelection.map((i) => ({
    remote: i.remote,
    path: i.path,
    profile_id: activeProfile.value?.id,
  }))

  explorerStore
    .compareFiles(items)
    .then((res) => {
      compareData.value = res
    })
    .catch((e: any) => {
      compareError.value = e.message || 'Comparison failed'
    })
    .finally(() => {
      compareLoading.value = false
    })
}

function openMkdir(pane: 'left' | 'right') {
  mkdirTargetRemote.value = pane === 'left' ? leftRemote.value : rightRemote.value
  mkdirTargetRemotePath.value = pane === 'left' ? leftPath.value : rightPath.value
  mkdirError.value = null
  showMkdirModal.value = true
}

async function handleCreateDirectory(dirName: string) {
  const base = mkdirTargetRemotePath.value
  const fullPath = base ? `${base.replace(/\/+$/, '')}/${dirName}` : dirName
  mkdirLoading.value = true
  mkdirError.value = null

  try {
    await explorerStore.createDirectory(mkdirTargetRemote.value, fullPath, activeProfile.value?.id)
    showMkdirModal.value = false
    refreshAll()
  } catch (e: any) {
    mkdirError.value = e.message || 'Failed to create directory'
  } finally {
    mkdirLoading.value = false
  }
}

function handleContextMenu(e: MouseEvent, pane: 'left' | 'right', item: BrowseItem) {
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuItem.value = item
  contextMenuRemote.value = pane === 'left' ? leftRemote.value : rightRemote.value
  contextMenuVisible.value = true

  if (pane === 'left') leftSelected.value = item
  else rightSelected.value = item
}

function handleContextAction(act: 'preview' | 'compare' | 'download' | 'open_on_remote' | 'move' | 'copy_path' | 'delete') {
  if (!contextMenuItem.value) return
  const item = contextMenuItem.value
  const remote = contextMenuRemote.value
  const pane = remote === leftRemote.value ? 'left' : 'right'
  const fullPath = getRelativeItemPath(pane, item)

  switch (act) {
    case 'preview':
      if (isItemPreviewable(item)) {
        triggerPreview(remote, fullPath, item.Name)
      }
      break
    case 'compare':
      toggleCompare(pane, item)
      break
    case 'download': {
      const url = explorerStore.getDownloadUrl(remote, fullPath, activeProfile.value?.id)
      const a = document.createElement('a')
      a.href = url
      a.download = item.Name
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      break
    }
    case 'open_on_remote':
      openOnRemoteDirect(remote, fullPath)
      break
    case 'move':
      moveTargetRemote.value = remote
      moveSrcPath.value = fullPath
      moveIsDir.value = item.IsDir
      moveError.value = null
      showMoveModal.value = true
      break
    case 'copy_path':
      navigator.clipboard.writeText(fullPath)
      break
    case 'delete':
      confirmTitle.value = item.IsDir ? 'Confirm Directory Purge' : 'Confirm File Deletion'
      confirmMessage.value = `Are you sure you want to permanently delete "${item.Name}"?`
      confirmTarget.value = `${remote}:${fullPath}`
      deleteRemote.value = remote
      deletePath.value = fullPath
      deleteIsDir.value = item.IsDir
      showConfirmModal.value = true
      break
  }
}

async function handleExecuteMove(dstPath: string) {
  moveLoading.value = true
  moveError.value = null
  try {
    await explorerStore.moveItem({
      src_remote: moveTargetRemote.value,
      src_path: moveSrcPath.value,
      dst_remote: moveTargetRemote.value,
      dst_path: dstPath,
      is_dir: moveIsDir.value,
      profile_id: activeProfile.value?.id,
    })
    showMoveModal.value = false
    refreshAll()
  } catch (e: any) {
    moveError.value = e.message || 'Move failed'
  } finally {
    moveLoading.value = false
  }
}

async function handleExecuteDelete() {
  confirmLoading.value = true
  try {
    await explorerStore.deleteItem(deleteRemote.value, deletePath.value, deleteIsDir.value, activeProfile.value?.id)
    showConfirmModal.value = false
    refreshAll()
  } catch (e: any) {
    alert(e.message || 'Deletion failed')
  } finally {
    confirmLoading.value = false
  }
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes < 0) return '--'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}
watch(
  () => explorerStore.targetJumpPath,
  async (target) => {
    if (!target) return
    leftPath.value = target.dir
    rightPath.value = target.dir
    await Promise.allSettled([refreshPane('left'), refreshPane('right')])
  }
)

</script>
