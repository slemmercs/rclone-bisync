<template>
  <div class="space-y-4 font-mono text-xs">
    <!-- Fallback if no provider or options -->
    <div
      v-if="!provider || (!provider.options?.length && !flavorOption)"
      class="p-4 rounded-xl bg-carbon-surface border border-carbon-border text-slate-400 font-sans text-xs"
    >
      <div class="flex items-center gap-2 text-slate-300 font-semibold mb-1">
        <i class="pi pi-info-circle text-electric-blue text-sm"></i>
        <span>Standard Provider Configuration</span>
      </div>
      <p class="text-[11px] leading-relaxed">
        This provider uses default baseline settings. Direct deployment will establish initial connectivity or prompt for authentication.
      </p>
    </div>

    <div v-else class="space-y-4">
      <!-- Top-Level Provider Flavor / Sub-Provider Selector -->
      <div
        v-if="flavorOption"
        class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-2.5"
      >
        <div class="flex items-center justify-between">
          <label class="text-slate-200 font-semibold text-xs flex items-center gap-2">
            <i class="pi pi-server text-electric-blue text-xs"></i>
            <span>Provider Backend Flavor</span>
            <span class="text-[10px] text-electric-blue font-normal">(Preset)</span>
          </label>
          <span
            v-if="selectedFlavor"
            class="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-electric-blue/10 text-electric-blue border border-electric-blue/20"
          >
            {{ selectedFlavor }}
          </span>
        </div>

        <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
          Select target cloud or object storage preset to calibrate region, signature version, and endpoint parameters.
        </p>

        <select
          :value="selectedFlavor"
          @change="onFlavorChange(($event.target as HTMLSelectElement).value)"
          class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-2 text-slate-200 focus:outline-none focus:border-electric-blue font-mono text-xs cursor-pointer"
        >
          <option
            v-for="ex in getOptExamples(flavorOption)"
            :key="ex.value"
            :value="ex.value"
          >
            {{ ex.value }} — {{ ex.help || ex.value }}
          </option>
        </select>
      </div>

      <!-- Standard / Required Configuration Fields -->
      <div
        v-if="standardOptions.length > 0"
        class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3.5"
      >
        <div class="flex items-center justify-between border-b border-carbon-border/60 pb-2">
          <label class="text-slate-200 font-semibold text-xs flex items-center gap-2">
            <i class="pi pi-sliders-h text-emerald-400 text-xs"></i>
            <span>Configuration Parameters</span>
          </label>
          <span class="text-[10px] text-slate-500">
            {{ standardOptions.length }} setting{{ standardOptions.length === 1 ? '' : 's' }}
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
          <div
            v-for="opt in standardOptions"
            :key="getOptName(opt)"
            :class="isOptFullWidth(opt) ? 'sm:col-span-2' : ''"
            class="space-y-1"
          >
            <!-- Boolean Field -->
            <div v-if="getOptType(opt) === 'bool'" class="pt-1">
              <label class="flex items-start gap-2.5 cursor-pointer select-none">
                <input
                  type="checkbox"
                  :checked="getBoolValue(opt)"
                  @change="onCheckboxChange(getOptName(opt), ($event.target as HTMLInputElement).checked)"
                  class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-4 h-4 mt-0.5 cursor-pointer"
                />
                <div class="space-y-0.5">
                  <div class="text-xs text-white font-semibold flex items-center gap-1.5">
                    <span>{{ formatLabel(getOptName(opt)) }}</span>
                    <span v-if="isOptRequired(opt)" class="text-rose-400 text-xs">*</span>
                  </div>
                  <p v-if="getOptHelp(opt)" class="text-[11px] text-slate-400 font-sans leading-relaxed">
                    {{ getOptHelp(opt) }}
                  </p>
                </div>
              </label>
            </div>

            <!-- Password Field -->
            <div v-else-if="isOptPassword(opt)">
              <div class="flex items-center justify-between mb-1">
                <label class="text-slate-300 font-semibold text-[11px] flex items-center gap-1">
                  <span>{{ formatLabel(getOptName(opt)) }}</span>
                  <span v-if="isOptRequired(opt)" class="text-rose-400">*</span>
                </label>
                <button
                  type="button"
                  @click="togglePassword(getOptName(opt))"
                  class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans flex items-center gap-1"
                >
                  <i :class="showPasswords[getOptName(opt)] ? 'pi pi-eye-slash' : 'pi pi-eye'" class="text-[10px]"></i>
                  <span>{{ showPasswords[getOptName(opt)] ? 'Hide' : 'Show' }}</span>
                </button>
              </div>
              <input
                :type="showPasswords[getOptName(opt)] ? 'text' : 'password'"
                :value="getFieldValue(getOptName(opt))"
                @input="onFieldInput(getOptName(opt), ($event.target as HTMLInputElement).value)"
                :placeholder="getPlaceholder(opt)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono text-xs"
              />
              <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                {{ getOptHelp(opt) }}
              </p>
            </div>

            <!-- Select Dropdown with Examples -->
            <div v-else-if="getOptExamples(opt).length > 0">
              <label class="block text-slate-300 font-semibold mb-1 text-[11px] flex items-center gap-1">
                <span>{{ formatLabel(getOptName(opt)) }}</span>
                <span v-if="isOptRequired(opt)" class="text-rose-400">*</span>
              </label>
              <select
                :value="getFieldValue(getOptName(opt))"
                @change="onFieldInput(getOptName(opt), ($event.target as HTMLSelectElement).value)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue font-mono text-xs cursor-pointer"
              >
                <option value="">{{ getPlaceholder(opt) || '-- Select option --' }}</option>
                <option
                  v-for="ex in getOptExamples(opt)"
                  :key="ex.value"
                  :value="ex.value"
                >
                  {{ ex.value }} {{ ex.help ? '— ' + ex.help : '' }}
                </option>
              </select>
              <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                {{ getOptHelp(opt) }}
              </p>
            </div>

            <!-- Standard Text / Number Input -->
            <div v-else>
              <label class="block text-slate-300 font-semibold mb-1 text-[11px] flex items-center gap-1">
                <span>{{ formatLabel(getOptName(opt)) }}</span>
                <span v-if="isOptRequired(opt)" class="text-rose-400">*</span>
              </label>
              <input
                :type="getOptType(opt) === 'int' ? 'number' : 'text'"
                :value="getFieldValue(getOptName(opt))"
                @input="onFieldInput(getOptName(opt), ($event.target as HTMLInputElement).value)"
                :placeholder="getPlaceholder(opt)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono text-xs"
              />
              <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                {{ getOptHelp(opt) }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Collapsible Advanced Settings Accordion -->
      <div v-if="advancedOptions.length > 0" class="rounded-xl border border-carbon-border overflow-hidden">
        <button
          type="button"
          @click="showAdvanced = !showAdvanced"
          class="w-full flex items-center justify-between p-3.5 bg-carbon-surface hover:bg-carbon-elevated/70 text-left transition-colors cursor-pointer select-none"
        >
          <div class="flex items-center gap-2">
            <i class="pi pi-cog text-slate-400 text-xs"></i>
            <span class="text-xs font-bold text-slate-200">Advanced Settings</span>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-carbon-canvas border border-carbon-border text-slate-400">
              {{ advancedOptions.length }}
            </span>
          </div>
          <div class="flex items-center gap-1.5 text-slate-400 text-xs">
            <span class="text-[11px]">{{ showAdvanced ? 'Hide' : 'Show' }}</span>
            <i :class="showAdvanced ? 'pi pi-chevron-up' : 'pi pi-chevron-down'" class="text-[10px]"></i>
          </div>
        </button>

        <div v-if="showAdvanced" class="p-3.5 bg-carbon-canvas border-t border-carbon-border space-y-3.5">
          <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
            Fine-tune low-level backend parameters, timeout thresholds, chunk sizes, and protocol flags.
          </p>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <div
              v-for="opt in advancedOptions"
              :key="getOptName(opt)"
              :class="isOptFullWidth(opt) ? 'sm:col-span-2' : ''"
              class="space-y-1"
            >
              <!-- Boolean Field -->
              <div v-if="getOptType(opt) === 'bool'" class="pt-1">
                <label class="flex items-start gap-2.5 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    :checked="getBoolValue(opt)"
                    @change="onCheckboxChange(getOptName(opt), ($event.target as HTMLInputElement).checked)"
                    class="rounded bg-carbon-surface border-carbon-border text-electric-blue focus:ring-0 w-4 h-4 mt-0.5 cursor-pointer"
                  />
                  <div class="space-y-0.5">
                    <div class="text-xs text-white font-semibold flex items-center gap-1.5">
                      <span>{{ formatLabel(getOptName(opt)) }}</span>
                    </div>
                    <p v-if="getOptHelp(opt)" class="text-[11px] text-slate-400 font-sans leading-relaxed">
                      {{ getOptHelp(opt) }}
                    </p>
                  </div>
                </label>
              </div>

              <!-- Password Field -->
              <div v-else-if="isOptPassword(opt)">
                <div class="flex items-center justify-between mb-1">
                  <label class="text-slate-300 font-semibold text-[11px]">
                    {{ formatLabel(getOptName(opt)) }}
                  </label>
                  <button
                    type="button"
                    @click="togglePassword(getOptName(opt))"
                    class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans flex items-center gap-1"
                  >
                    <i :class="showPasswords[getOptName(opt)] ? 'pi pi-eye-slash' : 'pi pi-eye'" class="text-[10px]"></i>
                    <span>{{ showPasswords[getOptName(opt)] ? 'Hide' : 'Show' }}</span>
                  </button>
                </div>
                <input
                  :type="showPasswords[getOptName(opt)] ? 'text' : 'password'"
                  :value="getFieldValue(getOptName(opt))"
                  @input="onFieldInput(getOptName(opt), ($event.target as HTMLInputElement).value)"
                  :placeholder="getPlaceholder(opt)"
                  class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono text-xs"
                />
                <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                  {{ getOptHelp(opt) }}
                </p>
              </div>

              <!-- Select Dropdown with Examples -->
              <div v-else-if="getOptExamples(opt).length > 0">
                <label class="block text-slate-300 font-semibold mb-1 text-[11px]">
                  {{ formatLabel(getOptName(opt)) }}
                </label>
                <select
                  :value="getFieldValue(getOptName(opt))"
                  @change="onFieldInput(getOptName(opt), ($event.target as HTMLSelectElement).value)"
                  class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue font-mono text-xs cursor-pointer"
                >
                  <option value="">{{ getPlaceholder(opt) || '-- Select option --' }}</option>
                  <option
                    v-for="ex in getOptExamples(opt)"
                    :key="ex.value"
                    :value="ex.value"
                  >
                    {{ ex.value }} {{ ex.help ? '— ' + ex.help : '' }}
                  </option>
                </select>
                <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                  {{ getOptHelp(opt) }}
                </p>
              </div>

              <!-- Standard Text / Number Input -->
              <div v-else>
                <label class="block text-slate-300 font-semibold mb-1 text-[11px]">
                  {{ formatLabel(getOptName(opt)) }}
                </label>
                <input
                  :type="getOptType(opt) === 'int' ? 'number' : 'text'"
                  :value="getFieldValue(getOptName(opt))"
                  @input="onFieldInput(getOptName(opt), ($event.target as HTMLInputElement).value)"
                  :placeholder="getPlaceholder(opt)"
                  class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono text-xs"
                />
                <p v-if="getOptHelp(opt)" class="text-[10px] text-slate-400 font-sans mt-1">
                  {{ getOptHelp(opt) }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { ProviderCatalogItem, ProviderOption } from '../types/dedupe'

interface Props {
  provider: ProviderCatalogItem | null
  modelValue?: Record<string, any>
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({}),
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, any>): void
}>()

const showAdvanced = ref<boolean>(false)
const showPasswords = ref<Record<string, boolean>>({})
const selectedFlavor = ref<string>('')

// Normalization Helpers for ProviderOption
function getOptName(opt: ProviderOption): string {
  return opt.name || opt.Name || ''
}

function getOptHelp(opt: ProviderOption): string {
  return opt.help || opt.Help || ''
}

function getOptType(opt: ProviderOption): string {
  return (opt.type || opt.Type || 'string').toLowerCase()
}

function getOptHide(opt: ProviderOption): number {
  return opt.hide ?? opt.Hide ?? 0
}

function isOptAdvanced(opt: ProviderOption): boolean {
  return !!(opt.advanced ?? opt.Advanced)
}

function isOptRequired(opt: ProviderOption): boolean {
  return !!(opt.required ?? opt.Required)
}

function isOptPassword(opt: ProviderOption): boolean {
  return !!((opt.is_password ?? opt.IsPassword) || getOptType(opt) === 'password')
}

function getOptProvider(opt: ProviderOption): string {
  return opt.provider || opt.Provider || ''
}

function getOptDefault(opt: ProviderOption): any {
  return opt.default !== undefined ? opt.default : opt.Default
}

interface ExampleItem {
  value: string
  help: string
}

function getOptExamples(opt: ProviderOption): ExampleItem[] {
  const raw = opt.examples || opt.Examples || []
  return raw.map((e) => ({
    value: String((e as any).value ?? (e as any).Value ?? ''),
    help: String((e as any).help ?? (e as any).Help ?? ''),
  }))
}

function isOptFullWidth(opt: ProviderOption): boolean {
  const type = getOptType(opt)
  const name = getOptName(opt).toLowerCase()
  return (
    type === 'bool' ||
    name.includes('endpoint') ||
    name.includes('url') ||
    name.includes('description') ||
    getOptHelp(opt).length > 80
  )
}

function formatLabel(name: string): string {
  if (!name) return ''
  return name
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase())
}

function getPlaceholder(opt: ProviderOption): string {
  const def = getOptDefault(opt)
  if (def !== undefined && def !== null && def !== '') {
    return `Default: ${def}`
  }
  const examples = getOptExamples(opt)
  if (examples.length > 0) {
    return `e.g. ${examples[0].value}`
  }
  return ''
}

// Find top-level flavor option (e.g. "provider" option in S3)
const flavorOption = computed<ProviderOption | null>(() => {
  if (!props.provider?.options) return null
  return (
    props.provider.options.find((opt) => {
      const name = getOptName(opt).toLowerCase()
      const examples = getOptExamples(opt)
      return name === 'provider' && examples.length > 0
    }) || null
  )
})
// Initialize and watch Flavor selection
watch(
  () => [props.provider, flavorOption.value],
  () => {
    if (flavorOption.value) {
      const examples = getOptExamples(flavorOption.value)
      const currentVal = props.modelValue['provider']
      if (currentVal) {
        selectedFlavor.value = String(currentVal)
      } else {
        const def = getOptDefault(flavorOption.value)
        const initialFlavor = def ? String(def) : (examples[0]?.value || 'AWS')
        selectedFlavor.value = initialFlavor
        updateField('provider', initialFlavor)
      }
    } else {
      selectedFlavor.value = ''
    }
  },
  { immediate: true }
)

function onFlavorChange(val: string) {
  selectedFlavor.value = val
  updateField('provider', val)
}

// Dynamic visibility filtering
function isOptionVisible(opt: ProviderOption): boolean {
  const name = getOptName(opt)
  if (!name) return false

  // Suppress configurator internal state flags: (opt.Hide & 3) !== 0
  const hide = getOptHide(opt)
  if ((hide & 3) !== 0) return false

  const internalKeys = ['config_fs_advanced', 'config_is_local', 'token']
  if (internalKeys.includes(name.toLowerCase())) return false

  // If this is the flavor option itself, we render it at the top separately
  if (flavorOption.value && getOptName(flavorOption.value) === name) {
    return false
  }

  // Flavor filtering logic (e.g. S3 AWS, Minio, Ceph, etc.)
  const providerFilter = getOptProvider(opt)
  if (providerFilter && selectedFlavor.value) {
    const tokens = providerFilter
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const negativeTokens = tokens
      .filter((t) => t.startsWith('!'))
      .map((t) => t.slice(1).toLowerCase())
    const positiveTokens = tokens
      .filter((t) => !t.startsWith('!'))
      .map((t) => t.toLowerCase())
    const current = selectedFlavor.value.toLowerCase()

    // If negated match, hide
    if (negativeTokens.includes(current)) {
      return false
    }

    // If positive tokens specified, must match one
    if (positiveTokens.length > 0 && !positiveTokens.includes(current)) {
      return false
    }
  }

  return true
}

const visibleOptions = computed<ProviderOption[]>(() => {
  if (!props.provider?.options) return []
  return props.provider.options.filter(isOptionVisible)
})

const standardOptions = computed<ProviderOption[]>(() => {
  return visibleOptions.value.filter((opt) => !isOptAdvanced(opt) || isOptRequired(opt))
})

const advancedOptions = computed<ProviderOption[]>(() => {
  return visibleOptions.value.filter((opt) => isOptAdvanced(opt) && !isOptRequired(opt))
})

// Field Getters and Mutators
function getFieldValue(name: string): any {
  return props.modelValue[name] ?? ''
}

function getBoolValue(opt: ProviderOption): boolean {
  const name = getOptName(opt)
  if (props.modelValue[name] !== undefined) {
    return Boolean(props.modelValue[name])
  }
  const def = getOptDefault(opt)
  return def === true || def === 'true'
}

function togglePassword(name: string) {
  showPasswords.value[name] = !showPasswords.value[name]
}

function updateField(key: string, value: any) {
  const updated = { ...props.modelValue, [key]: value }
  emit('update:modelValue', updated)
}

function onFieldInput(key: string, value: any) {
  updateField(key, value)
}

function onCheckboxChange(key: string, checked: boolean) {
  updateField(key, checked)
}
</script>
