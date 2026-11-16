<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Chart } from 'chart.js/auto'
import { useTelemetryStore } from '../stores/telemetry'

const telemetryStore = useTelemetryStore()
const canvasRef = ref<HTMLCanvasElement | null>(null)

// Non-reactive reference outside Vue proxy system to prevent recursive proxy loops
let chartInstance: Chart | null = null

const history = ref<{ time: string; speed: number; latency: number }[]>([])

function getSpeedMBps(): number {
  const bytesPerSec = telemetryStore.transferStats?.speed || 0
  return Number((bytesPerSec / (1024 * 1024)).toFixed(2))
}

function updateChartData() {
  if (!chartInstance) return

  const now = new Date().toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
  const speed = getSpeedMBps()
  const latency = telemetryStore.telemetry?.daemon_health?.response_time_ms || 0

  history.value.push({ time: now, speed, latency })
  if (history.value.length > 20) {
    history.value.shift()
  }

  chartInstance.data.labels = history.value.map((h) => h.time)
  chartInstance.data.datasets[0].data = history.value.map((h) => h.speed)
  chartInstance.data.datasets[1].data = history.value.map((h) => h.latency)
  chartInstance.update('none')
}

onMounted(() => {
  if (!canvasRef.value) return

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  // Electric Blue Gradient
  const gradient = ctx.createLinearGradient(0, 0, 0, 180)
  gradient.addColorStop(0, 'rgba(14, 165, 233, 0.35)')
  gradient.addColorStop(1, 'rgba(14, 165, 233, 0.0)')

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        {
          label: 'Throughput (MB/s)',
          data: [],
          borderColor: '#0EA5E9',
          backgroundColor: gradient,
          fill: true,
          tension: 0.3,
          borderWidth: 2,
          pointRadius: 2,
          pointBackgroundColor: '#0EA5E9',
          yAxisID: 'y',
        },
        {
          label: 'Latency (ms)',
          data: [],
          borderColor: '#A855F7',
          backgroundColor: 'transparent',
          fill: false,
          tension: 0.3,
          borderWidth: 1.5,
          borderDash: [4, 4],
          pointRadius: 0,
          yAxisID: 'y1',
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: {
        legend: {
          display: true,
          labels: {
            color: '#94A3B8',
            font: { size: 10, family: 'monospace' },
            boxWidth: 12,
          },
        },
        tooltip: {
          mode: 'index',
          intersect: false,
          backgroundColor: '#111318',
          borderColor: '#262A36',
          borderWidth: 1,
          titleColor: '#F8FAFC',
          bodyColor: '#CBD5E1',
          bodyFont: { family: 'monospace', size: 11 },
        },
      },
      scales: {
        x: {
          grid: { color: '#191C24' },
          ticks: { color: '#64748B', font: { size: 9, family: 'monospace' }, maxTicksLimit: 6 },
        },
        y: {
          type: 'linear',
          display: true,
          position: 'left',
          grid: { color: '#191C24' },
          ticks: { color: '#0EA5E9', font: { size: 9, family: 'monospace' } },
          min: 0,
        },
        y1: {
          type: 'linear',
          display: true,
          position: 'right',
          grid: { drawOnChartArea: false },
          ticks: { color: '#A855F7', font: { size: 9, family: 'monospace' } },
          min: 0,
        },
      },
    },
  })

  // Seed initial entry
  updateChartData()
})

watch(
  () => [telemetryStore.transferStats?.speed, telemetryStore.telemetry?.daemon_health?.response_time_ms],
  () => {
    updateChartData()
  }
)

onUnmounted(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>

<template>
  <div class="h-44 w-full">
    <canvas ref="canvasRef"></canvas>
  </div>
</template>
