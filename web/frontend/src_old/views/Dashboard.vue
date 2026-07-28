<template>
    <div class="dashboard-view">
        <div class="header-section">
            <h1>Gateway Dashboard</h1>
            <div class="status-indicator" :class="{ connected: isConnected }">
                {{ isConnected ? '● Live' : '○ Reconnecting' }}
            </div>
        </div>

        <!-- Global Stats -->
        <div class="stats-row">
            <div class="stat-card">
                <h3>Poller Read Ops</h3>
                <div class="stat-val">{{ (pollerStats.read_ops || 0).toLocaleString() }} <span class="unit">ops</span></div>
            </div>
            <div class="stat-card">
                <h3>Poller Write Ops</h3>
                <div class="stat-val">{{ (pollerStats.write_ops || 0).toLocaleString() }} <span class="unit">ops</span></div>
            </div>
            <div class="stat-card success">
                <h3>Avg Read RTT</h3>
                <div class="stat-val">{{ pollerStats.read_latency_ms || 0 }} <span class="unit">ms</span></div>
            </div>
            <div class="stat-card error">
                <h3>Avg Write RTT</h3>
                <div class="stat-val">{{ pollerStats.write_latency_ms || 0 }} <span class="unit">ms</span></div>
            </div>
        </div>

        <!-- Manual Read/Write Tester Panel -->
        <div class="tester-panel glass-card">
            <div class="panel-header">
                <h2>Manual Tag Tester</h2>
                <span class="badge">Direct PLC Connection</span>
            </div>
            <div class="tester-controls">
                <select class="input-3d" v-model="testPlcId" @change="filterTagsForPlc">
                    <option value="">Select PLC...</option>
                    <option v-for="plc in plcs" :key="plc.id" :value="plc.id">{{ plc.ip_address }} ({{ plc.facility_name }})</option>
                </select>

                <select class="input-3d" v-model="testTagId">
                    <option value="">Select Tag Address...</option>
                    <option v-for="tag in filteredTags" :key="tag.id" :value="tag.id">
                        {{ tag.tag_name }} [{{ tag.device }}{{ tag.offset }}]
                    </option>
                </select>

                <input type="number" class="input-3d value-input" v-model="testValue" placeholder="Value (0)">

                <button class="btn-3d" @click="manualRead" :disabled="!testTagId">Read</button>
                <button class="btn-3d write" @click="manualWrite" :disabled="!testTagId">Write</button>
            </div>
            <div class="tester-result" v-if="testResult">
                <span>Value: <strong>{{ testResult.value }}</strong></span>
                <span class="rtt">RTT: {{ testResult.rtt_ms }} ms</span>
            </div>
        </div>

        <!-- ECharts Trend -->
        <div class="charts-row">
            <div class="chart-container glass-card">
                <div class="chart-header">
                    <h2>Latency Trends</h2>
                    <div class="plc-selector">
                        <button class="plc-chip" :class="{ active: selectedChartPlc === 'Global' }" @click="selectedChartPlc = 'Global'">Global</button>
                        <button class="plc-chip" :class="{ active: selectedChartPlc === 'Manual' }" @click="selectedChartPlc = 'Manual'">Manual Tests</button>
                    </div>
                </div>
                <v-chart class="chart" :option="rttChartOption" autoresize />
            </div>
        </div>

        <!-- Tag Monitoring Grid -->
        <div class="section-header">
            <h2>Live Facility Monitoring</h2>
        </div>
        <div class="facilities-grid">
            <div class="facility-card glass-card" v-for="(tags, facility) in tagsByFacility" :key="facility">
                <h2>{{ facility }}</h2>
                <div class="tag-list">
                    <div class="tag-item" v-for="tag in tags" :key="tag.id">
                        <span class="tag-name">{{ tag.tag_name }} [{{tag.device}}{{tag.offset}}]</span>
                        <span class="tag-value" :class="{ updated: tag.isUpdated }">{{ tag.value }}</span>
                    </div>
                    <div v-if="tags.length === 0" class="no-tags">No tags mapped.</div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, TitleComponent, LegendComponent, ToolboxComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, TitleComponent, LegendComponent, ToolboxComponent])

const health = ref([])
const pollerStats = ref({})
const isConnected = ref(false)
let ws = null
let pollInterval = null

// Real-time tagging state
const allTags = ref([])
const plcs = ref([])
const tagsByFacility = ref({})

// Manual Tester State
const testPlcId = ref("")
const testTagId = ref("")
const testValue = ref(0)
const filteredTags = ref([])
const testResult = ref(null)

// ECharts State
const selectedChartPlc = ref('Global')
const globalHistory = ref({ times: [], readRTT: [], writeRTT: [] })
const manualHistory = ref({ times: [], rtt: [] })

const rttChartOption = computed(() => {
    let times, series, legendData, colors, subtext

    if (selectedChartPlc.value === 'Global') {
        times = globalHistory.value.times
        legendData = ['Global Read RTT', 'Global Write RTT']
        colors = ['#10b981', '#38bdf8']
        subtext = 'Poller Aggregated Average'
        series = [
            { name: 'Global Read RTT', type: 'line', data: globalHistory.value.readRTT, smooth: true, lineStyle: { width: 3 } },
            { name: 'Global Write RTT', type: 'line', data: globalHistory.value.writeRTT, smooth: true, lineStyle: { width: 3 } }
        ]
    } else {
        times = manualHistory.value.times
        legendData = ['Manual RTT']
        colors = ['#f59e0b']
        subtext = 'Manual Read/Write Tests'
        series = [
            { name: 'Manual RTT', type: 'line', data: manualHistory.value.rtt, smooth: true, symbolSize: 8, lineStyle: { width: 3, type: 'dashed' } }
        ]
    }

    return {
        color: colors,
        backgroundColor: 'transparent',
        title: {
            text: 'Latency Trend (ms)',
            subtext: subtext,
            textStyle: { color: '#f8fafc' },
            subtextStyle: { color: '#94a3b8' }
        },
        tooltip: { trigger: 'axis' },
        grid: { top: 60, bottom: 40, left: '5%', right: '4%', containLabel: true },
        legend: {
            data: legendData,
            textStyle: { color: '#f8fafc' },
            top: 0
        },
        xAxis: [
            {
                type: 'category',
                boundaryGap: false,
                axisLine: { lineStyle: { color: '#334155' } },
                axisLabel: { color: '#94a3b8' },
                data: times
            }
        ],
        yAxis: [
            {
                type: 'value',
                axisLabel: { formatter: '{value} ms', color: '#94a3b8' },
                splitLine: { lineStyle: { color: '#334155' } },
                min: 0,
                max: function(value) { return Math.max(10, Math.ceil(value.max * 1.2)) }
            }
        ],
        series: series
    }
})

function pushGlobalHistory(stats) {
    const timeStr = new Date().toLocaleTimeString([], { hour12: false })
    globalHistory.value.times.push(timeStr)
    globalHistory.value.readRTT.push(stats.read_latency_ms || 0)
    globalHistory.value.writeRTT.push(stats.write_latency_ms || 0)

    if (globalHistory.value.times.length > 50) {
        globalHistory.value.times.shift()
        globalHistory.value.readRTT.shift()
        globalHistory.value.writeRTT.shift()
    }
}

function pushManualHistory(rtt) {
    const timeStr = new Date().toLocaleTimeString([], { hour12: false })
    manualHistory.value.times.push(timeStr)
    manualHistory.value.rtt.push(rtt)

    if (manualHistory.value.times.length > 50) {
        manualHistory.value.times.shift()
        manualHistory.value.rtt.shift()
    }
}

async function fetchPlcsAndTags() {
    try {
        const pRes = await fetch('/api/plcs')
        const pData = await pRes.json()
        plcs.value = pData.data || []

        const tRes = await fetch('/api/tags')
        const tData = await tRes.json()
        allTags.value = tData.data || []
        
        // Group by facility
        const grouped = {}
        allTags.value.forEach(tag => {
            if (!grouped[tag.facility]) grouped[tag.facility] = []
            grouped[tag.facility].push({ ...tag, value: 0, isUpdated: false })
        })
        tagsByFacility.value = grouped
    } catch (e) {
        console.error(e)
    }
}

function filterTagsForPlc() {
    testTagId.value = ""
    testResult.value = null
    if (!testPlcId.value) {
        filteredTags.value = []
        return
    }
    filteredTags.value = allTags.value.filter(t => t.plc_id == testPlcId.value)
}

async function manualRead() {
    const tag = allTags.value.find(t => t.id == testTagId.value)
    if (!tag) return
    try {
        const res = await fetch(`/api/plcs/${testPlcId.value}/manual-read`, {
            method: 'POST',
            headers: {'Content-Type':'application/json'},
            body: JSON.stringify({
                device: tag.device,
                offset: tag.offset,
                is_bit: tag.tag_type === 'bit'
            })
        })
        const data = await res.json()
        testResult.value = { value: data.value, rtt_ms: data.rtt_ms }
        pushManualHistory(data.rtt_ms)
        selectedChartPlc.value = 'Manual'
    } catch (e) {
        console.error(e)
    }
}

async function manualWrite() {
    const tag = allTags.value.find(t => t.id == testTagId.value)
    if (!tag) return
    try {
        const res = await fetch(`/api/plcs/${testPlcId.value}/manual-write`, {
            method: 'POST',
            headers: {'Content-Type':'application/json'},
            body: JSON.stringify({
                device: tag.device,
                offset: tag.offset,
                is_bit: tag.tag_type === 'bit',
                value: parseInt(testValue.value, 10) || 0
            })
        })
        const data = await res.json()
        testResult.value = { value: 'Written ' + testValue.value, rtt_ms: data.rtt_ms }
        pushManualHistory(data.rtt_ms)
        selectedChartPlc.value = 'Manual'
    } catch (e) {
        console.error(e)
    }
}

async function fetchStats() {
    try {
        const res = await fetch('/api/poller-stats')
        const data = await res.json()
        pollerStats.value = data
        pushGlobalHistory(data)
    } catch (err) {
        console.error('Stats err:', err)
    }
}

function connectWS() {
    ws = new WebSocket(`ws://${window.location.host}/ws`)
    ws.onopen = () => { isConnected.value = true }
    ws.onclose = () => { isConnected.value = false; setTimeout(connectWS, 3000) }
    ws.onmessage = (msg) => {
        const data = JSON.parse(msg.data)
        const fac = tagsByFacility.value[data.facility]
        if (fac) {
            const tag = fac.find(t => t.tag_name === data.tag_name)
            if (tag) {
                tag.value = data.value
                tag.isUpdated = true
                setTimeout(() => { tag.isUpdated = false }, 400) // Neon flash effect
            }
        }
    }
}

onMounted(() => {
    fetchPlcsAndTags()
    fetchStats()
    pollInterval = setInterval(() => {
        fetchStats()
    }, 2000)
    connectWS()
})

onUnmounted(() => {
    clearInterval(pollInterval)
    if (ws) ws.close()
})
</script>

<style scoped>
.dashboard-view {
  padding: 24px;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-section h1 {
  color: var(--text-primary);
  font-weight: 800;
  margin: 0;
  font-size: 1.8rem;
}

.status-indicator {
  padding: 8px 16px;
  border-radius: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-weight: 600;
  transition: all 0.3s ease;
}
.status-indicator.connected {
  background: rgba(16, 185, 129, 0.1);
  border-color: var(--accent-neon);
  color: var(--accent-neon);
  box-shadow: 0 0 15px rgba(16, 185, 129, 0.2);
}

.section-header {
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 8px;
}
.section-header h2 {
  color: var(--text-primary);
  font-weight: 700;
  margin: 0;
}

/* Glass Cards */
.glass-card {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

/* Stats Row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.4);
}
.stat-card h3 {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 1px;
  font-weight: 600;
}
.stat-val {
  font-size: 2.2rem;
  font-weight: 800;
  color: var(--text-primary);
  margin-top: 8px;
}
.stat-val .unit {
  font-size: 1rem;
  color: var(--text-secondary);
  font-weight: 500;
}
.stat-card.success .stat-val { color: var(--accent-neon); }
.stat-card.error .stat-val { color: var(--accent-blue); }

/* Tester Panel */
.tester-panel {
  padding: 24px;
  margin-bottom: 30px;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.panel-header h2 {
  margin: 0;
  font-size: 1.4rem;
  color: var(--accent-neon);
  text-shadow: 0 0 10px rgba(16, 185, 129, 0.3);
}
.badge {
  background: rgba(16, 185, 129, 0.15);
  color: var(--accent-neon);
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: bold;
  border: 1px solid var(--accent-neon);
}
.tester-controls {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  align-items: center;
}
.value-input {
  width: 120px;
}
.tester-result {
  margin-top: 20px;
  padding: 16px;
  background: rgba(0,0,0,0.2);
  border-radius: 8px;
  border-left: 4px solid var(--accent-neon);
  display: flex;
  gap: 24px;
  font-size: 1.1rem;
}
.tester-result strong {
  color: var(--accent-neon);
}
.tester-result .rtt {
  color: var(--accent-blue);
  font-family: 'JetBrains Mono', monospace;
  font-weight: bold;
}

/* Charts */
.charts-row {
  margin-bottom: 30px;
}
.chart-container {
  padding: 24px;
}
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.chart-header h2 {
  margin: 0;
  font-size: 1.3rem;
  color: var(--text-primary);
  font-weight: 700;
}
.plc-selector {
  display: flex;
  gap: 8px;
}
.plc-chip {
  padding: 6px 14px;
  border-radius: 20px;
  background: var(--bg-main);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}
.plc-chip:hover {
  background: var(--border-color);
  color: var(--text-primary);
}
.plc-chip.active {
  background: rgba(16, 185, 129, 0.15);
  color: var(--accent-neon);
  border-color: var(--accent-neon);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.2);
}
.chart {
  height: 400px;
  width: 100%;
}

/* Tags Grid */
.facilities-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 24px;
}
.facility-card {
  padding: 24px;
}
.facility-card h2 {
  margin-top: 0;
  margin-bottom: 20px;
  color: var(--text-primary);
  text-transform: capitalize;
  font-size: 1.3rem;
  font-weight: 800;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 12px;
}
.tag-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tag-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-main);
  border: 1px solid var(--border-color);
  padding: 14px 18px;
  border-radius: 12px;
  transition: all 0.3s ease;
}
.tag-item:hover {
  border-color: var(--accent-neon);
  box-shadow: 0 0 10px rgba(16, 185, 129, 0.1);
}
.tag-name {
  color: var(--text-secondary);
  font-size: 0.95rem;
  font-weight: 600;
}
.tag-value {
  font-family: 'JetBrains Mono', monospace;
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--accent-blue);
  transition: all 0.2s ease;
}
.tag-value.updated {
  color: var(--accent-neon);
  transform: scale(1.15);
  text-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
}
.no-tags {
  color: var(--text-secondary);
  font-style: italic;
  text-align: center;
  padding: 24px 0;
  font-weight: 500;
}
</style>
