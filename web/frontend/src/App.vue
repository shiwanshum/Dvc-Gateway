<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import Papa from 'papaparse'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  GridComponent,
  MarkAreaComponent,
  VisualMapComponent
} from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  GridComponent,
  MarkAreaComponent,
  VisualMapComponent
])

const API = 'http://' + window.location.hostname + ':6080/api'

// Navigation
const isSidebarCollapsed = ref(false)
const tab = ref('dashboard')
const globalSearch = ref('')
const apiHealthy = ref(false)

// Toast
const toasts = ref([])
function toast(msg, type = 'info') {
    const icons = { success: 'checkmark-circle', error: 'alert-circle', info: 'information-circle' }
    toasts.value.push({ msg, type, icon: icons[type] || icons.info })
    setTimeout(() => toasts.value.shift(), 4000)
}

// Loading flags
const loading = reactive({ robots: false, tags: false, plcs: false })

// ===================== DASHBOARD =====================
const health = ref([])
const tagCount = ref(0)
const pollerStats = ref({ read_latency_ms: 0, write_latency_ms: 0, read_ops: 0, write_ops: 0, error_count: 0 })
const showErrorSidebar = ref(false)
const errorLog = ref([])
let errorPollInterval = null
const mcSimulator = computed(() => health.value.find(p => p.ip_address === '192.169.4.101'))

const otherPLCs = computed(() => health.value.filter(p => p.ip_address !== '192.169.4.101'))
const onlineCount = computed(() => health.value.filter(p => p.status === 'online').length)
const offlineCount = computed(() => health.value.filter(p => p.status === 'offline').length)

// Chart Logic
const selectedChartPlc = ref(['Global'])
const rttHistoryLimit = ref(50)
const historyLimitOptions = [10, 30, 50, 100, 300, 500, 1000, 1500, 2000, 2500, 3000, 5000, 7500, 10000]
const rttTimeRange = ref(11000)

function toggleChartPlc(id) {
    const idx = selectedChartPlc.value.indexOf(id)
    if (idx > -1) {
        if (selectedChartPlc.value.length > 1) {
            selectedChartPlc.value.splice(idx, 1)
        }
    } else {
        selectedChartPlc.value.push(id)
    }
}

function chipStyle(id) {
    const isActive = selectedChartPlc.value.includes(id)
    if (isActive) {
        return {
            background: 'linear-gradient(135deg, var(--accent-blue), var(--accent-orange))',
            color: '#fff',
            boxShadow: '0 4px 12px rgba(56,189,248,0.35)',
            transform: 'translateY(-2px)'
        }
    } else {
        return {
            background: 'rgba(255,255,255,0.05)',
            color: 'var(--text-secondary)',
            boxShadow: 'none',
            border: '1px solid rgba(255,255,255,0.1)',
            transform: 'translateY(0)'
        }
    }
}
const timeRangeOptions = [
    { label: '11 Seconds', value: 11000 },
    { label: '1 Minute', value: 60000 },
    { label: '5 Minutes', value: 300000 },
    { label: '1 Hour', value: 3600000 },
    { label: '24 Hours', value: 86400000 }
]

const rttHistory = ref([]) // array of { time: Date, read: number, write: number, plc_id: string }

const rttChartOption = computed(() => {
    const now = Date.now()
    const colors = ['#5470C6', '#EE6666', '#91CC75', '#FAC858', '#73C0DE', '#3BA272', '#FC8452', '#9A60B4']
    
    let series = []
    let legendData = []
    let times = []
    let colorIdx = 0

    selectedChartPlc.value.forEach(selId => {
        let hist = rttHistory.value.filter(h => h.plc_id === selId && (now - h.time.getTime() <= rttTimeRange.value))
        hist = hist.slice(-rttHistoryLimit.value)

        if (times.length === 0 && hist.length > 0) {
            times = hist.map(h => {
                const d = h.time
                return `${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}:${d.getSeconds().toString().padStart(2,'0')}`
            })
        }

        let label = selId === 'Global' ? 'Global' : (health.value.find(p => p.id === selId)?.ip_address || selId)

        legendData.push(`${label} Read`)
        legendData.push(`${label} Write`)

        series.push({
            name: `${label} Read`,
            type: 'line',
            smooth: true,
            emphasis: { focus: 'series' },
            itemStyle: { color: colors[colorIdx % colors.length] },
            data: hist.map(h => h.read)
        })

        series.push({
            name: `${label} Write`,
            type: 'line',
            smooth: true,
            emphasis: { focus: 'series' },
            itemStyle: { color: colors[(colorIdx+1) % colors.length] },
            data: hist.map(h => h.write)
        })

        colorIdx += 2
    })

    return {
        color: colors,
        title: {
            text: 'Read/Write RTT Trend',
            subtext: selectedChartPlc.value.includes('Global') ? 'Global Poller RTT' : 'PLC RTT',
            textStyle: { color: '#ccc' }
        },
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'cross' }
        },
        toolbox: {
            show: true,
            feature: { saveAsImage: {} }
        },
        grid: {
            top: 70, bottom: 50, left: '3%', right: '4%', containLabel: true
        },
        legend: {
            data: legendData,
            textStyle: { color: '#ccc' },
            type: 'scroll',
            top: 25
        },
        xAxis: [
            {
                type: 'category',
                boundaryGap: false,
                axisTick: { alignWithLabel: true },
                axisLine: { lineStyle: { color: '#aaa' } },
                data: times
            }
        ],
        yAxis: [
            {
                type: 'value',
                axisLabel: { formatter: '{value} ms', color: '#aaa' },
                splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } },
                min: 0,
                max: function(value) {
                    return Math.max(10, Math.ceil(value.max * 1.2))
                }
            }
        ],
        series: series
    }
})

function trimHistory() {
    const maxEntries = 10000 // Keep up to max possible in memory
    if (rttHistory.value.length > maxEntries * 10) { // arbitrary safe buffer for multiple PLCs
        rttHistory.value = rttHistory.value.slice(-maxEntries * 5)
    }
}

async function fetchPLCHealth() {
    try {
        const res = await fetch(API + '/health/plcs')
        if (res.ok) {
            const data = await res.json()
            health.value = data
            apiHealthy.value = true
            const now = new Date()
            data.forEach(p => {
                rttHistory.value.push({ time: now, read: p.latency_ms || 0, write: 0, plc_id: p.id, ops: p.tags_count || 0 })
            })
            trimHistory()
        }
    } catch (_) { apiHealthy.value = false }
}

const rttMetrics = computed(() => {
    const now = Date.now()
    return selectedChartPlc.value.map(selId => {
        let hist = rttHistory.value.filter(h => h.plc_id === selId && (now - h.time.getTime() <= rttTimeRange.value))
        hist = hist.slice(-rttHistoryLimit.value)
        
        let label = selId === 'Global' ? 'Global Stats' : (health.value.find(p => p.id === selId)?.ip_address || selId)
        
        if (hist.length === 0) {
            return { label, read: { min: 0, max: 0, avg: 0 }, write: { min: 0, max: 0, avg: 0 }, ops: 0 }
        }
        
        const reads = hist.map(h => h.read)
        const writes = hist.map(h => h.write)
        
        return {
            label,
            read: {
                min: Math.min(...reads).toFixed(1),
                max: Math.max(...reads).toFixed(1),
                avg: (reads.reduce((a,b)=>a+b, 0) / reads.length).toFixed(1)
            },
            write: {
                min: Math.min(...writes).toFixed(1),
                max: Math.max(...writes).toFixed(1),
                avg: (writes.reduce((a,b)=>a+b, 0) / writes.length).toFixed(1)
            },
            ops: hist[hist.length-1]?.ops || 0
        }
    })
})

async function fetchTagCount() {
    try {
        const res = await fetch(API + '/tags?per_page=1')
        if (res.ok) {
            const data = await res.json()
            tagCount.value = data.total || 0
        }
    } catch (_) {}
}

async function fetchPollerStats() {
    try {
        const res = await fetch(API + '/poller-stats')
        if (res.ok) {
            const data = await res.json()
            pollerStats.value = data
            rttHistory.value.push({ time: new Date(), read: data.read_latency_ms, write: data.write_latency_ms, plc_id: 'Global', ops: data.read_ops || 0 })
            trimHistory()
        }
    } catch (_) {}
}

async function fetchErrorLog() {
    try {
        const res = await fetch(API + '/poller-errors')
        if (res.ok) errorLog.value = await res.json()
    } catch (_) {}
}

function toggleErrorSidebar() {
    showErrorSidebar.value = !showErrorSidebar.value
    if (showErrorSidebar.value) {
        fetchErrorLog()
        errorPollInterval = setInterval(fetchErrorLog, 2000)
    } else if (errorPollInterval) {
        clearInterval(errorPollInterval)
        errorPollInterval = null
    }
}

function dismissError(i) {
    errorLog.value.splice(i, 1)
}

async function clearAllErrors() {
    errorLog.value = []
    try { await fetch(`${API}/poller-errors`, { method: 'DELETE' }) } catch (_) {}
}

async function toggleRegisterById(tagId, currentVal, tagAddr) {
    const newVal = currentVal == 1 ? 0 : 1
    try {
        await fetch(`${API}/tags/${tagId}/write?value=${newVal}`, { method: 'PUT' })
        const tag = tags.value.find(t => t.id === tagId)
        if (tag) tag.value = newVal
    } catch (_) {}
}

function getRobotStatus(robot) {
    if (robot.plc_id) {
        const plc = health.value.find(p => p.id === robot.plc_id)
        if (plc) {
            return {
                class: plc.status,
                text: plc.status === 'online' ? 'Connected' : (plc.status === 'slow' ? 'Slow' : 'Offline')
            }
        }
    }
    return { class: 'offline', text: 'Unknown' }
}

function getRobotStats(robot) {
    if (robot.plc_id) {
        const plc = health.value.find(p => p.id === robot.plc_id)
        if (plc && plc.status === 'online') {
            return { text: 'Active', class: 'success-text' }
        }
    }
    return { text: 'Idle', class: 'muted-text' }
}

// ===================== PLCS =====================
const plcs = ref([])
const plcSearch = ref('')
const showPlcModal = ref(false)
const editingPlc = ref(null)
const plcForm = reactive({
    id: '', name: '', make: '', series: '', ip_address: '',
    protocol: '', read_port: 0
})
const scannedPorts = ref([])
const scanningPorts = ref(false)
const detectedProtocol = ref('')
const isScanningPorts = ref(false)

function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = Math.random() * 16 | 0
        return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16)
    })
}

const famousMakes = [
    'Mitsubishi', 'Siemens', 'Rockwell', 'Omron', 'Keyence',
    'Panasonic', 'Schneider', 'Beckhoff', 'Bosch Rexroth', 'ABB', 'Generic'
]

const makeSeriesMap = {
    'Mitsubishi': ['iQ-R', 'iQ-F', 'Q Series', 'FX Series', 'L Series', 'A Series'],
    'Siemens': ['S7-1200', 'S7-1500', 'S7-300', 'S7-400', 'S7-200', 'LOGO!'],
    'Rockwell': ['ControlLogix', 'CompactLogix', 'Micro800', 'PLC-5', 'SLC 500'],
    'Omron': ['NX', 'NJ', 'CP1', 'CJ2', 'CS1', 'NJ/NX'],
    'Keyence': ['KV-7000', 'KV-5000', 'KV-3000', 'KV-Nano'],
    'Panasonic': ['FP7', 'FP-XH', 'FP0R', 'FP Sigma'],
    'Schneider': ['Modicon M580', 'Modicon M340', 'Modicon M221', 'Modicon Quantum'],
    'Beckhoff': ['CX20xx', 'CX51xx', 'CX70xx', 'TwinCAT'],
    'Bosch Rexroth': ['IndraLogic XM', 'IndraLogic XLC'],
    'ABB': ['AC500', 'AC500-S', 'AC500-eCo'],
    'Generic': ['Other', 'Bench']
}

const seriesOptions = computed(() => {
    return makeSeriesMap[plcForm.make] || ['Other']
})

const readPortOptions = computed(() => {
    return scannedPorts.value
})

const protocolMap = {
    'Mitsubishi': 'MC Protocol',
    'Siemens': 'S7',
    'Rockwell': 'Ethernet/IP',
    'Omron': 'FINS',
    'Keyence': 'KV Protocol',
    'Panasonic': 'MEWTOCOL',
    'Schneider': 'Modbus TCP',
    'Beckhoff': 'ADS',
    'Bosch Rexroth': 'Sercos III',
    'ABB': 'Modbus TCP',
    'Generic': 'Modbus/Bench'
}

const filteredPlcs = computed(() => {
    if (!plcSearch.value) return plcs.value
    const q = plcSearch.value.toLowerCase()
    return plcs.value.filter(p =>
        p.name?.toLowerCase().includes(q) ||
        p.make?.toLowerCase().includes(q) ||
        p.ip_address?.includes(q)
    )
})

async function fetchPLCs() {
    loading.plcs = true
    try {
        const res = await fetch(API + '/plcs')
        if (res.ok) { const d = await res.json(); plcs.value = d.data || d || [] }
    } catch (_) {}
    finally { loading.plcs = false }
}

function openAddPlc() {
    editingPlc.value = null
    Object.assign(plcForm, {
        id: generateUUID(), name: '', make: '', series: '', ip_address: '',
        protocol: '', read_port: 0
    })
    scannedPorts.value = []
    detectedProtocol.value = ''
    showPlcModal.value = true
}

function openEditPlc(p) {
    editingPlc.value = p
    Object.assign(plcForm, {
        id: p.id, name: p.facility_name || '', make: p.maker || '',
        series: p.driver || '', ip_address: p.ip_address,
        protocol: p.comtype || '', read_port: p.port || 0
    })
    detectedProtocol.value = p.comtype || ''
    scannedPorts.value = []
    showPlcModal.value = true
}

async function autoDetectPlc() {
    if (!plcForm.ip_address) {
        toast('Please enter an IP address first', 'warning')
        return
    }
    toast('Detecting protocol...', 'info')
    try {
        const res = await fetch(API + '/autodetect?ip=' + plcForm.ip_address)
        if (res.ok) {
            const data = await res.json()
            if (data.detected) {
                plcForm.make = data.make || ''
                plcForm.series = data.series || ''
                plcForm.protocol = data.protocol || ''
                detectedProtocol.value = data.protocol || ''
                plcForm.read_port = data.read_port || 0
                toast('Detected: ' + data.make + ' - ' + data.protocol, 'success')
            } else {
                toast('No known PLC detected at ' + plcForm.ip_address, 'warning')
            }
        } else {
            toast('Auto-detect failed or not found', 'error')
        }
    } catch (_) {
        toast('Auto-detect network error', 'error')
    }
}

async function scanOpenPorts() {
    if (!plcForm.ip_address) {
        toast('Please enter an IP address first', 'warning')
        return
    }
    scanningPorts.value = true
    scannedPorts.value = []
    toast('Scanning ports on ' + plcForm.ip_address + '...', 'info')
    try {
        const make = plcForm.make || ''
        const res = await fetch(API + '/plcs/scan-ports?ip=' + plcForm.ip_address + '&make=' + encodeURIComponent(make))
        if (res.ok) {
            const data = await res.json()
            scannedPorts.value = data.open_ports || []
            if (scannedPorts.value.length > 0) {
                toast('Found ' + scannedPorts.value.length + ' open port(s)', 'success')
                if (!plcForm.read_port && scannedPorts.value.length > 0) {
                    plcForm.read_port = scannedPorts.value[0]
                }
            } else {
                toast('No open ports found in common ranges', 'warning')
            }
        } else {
            toast('Port scan failed', 'error')
        }
    } catch (_) {
        toast('Port scan network error', 'error')
    } finally {
        scanningPorts.value = false
    }
}

async function savePlc() {
    if (!plcForm.name || !plcForm.ip_address) { toast('Name and IP are required', 'error'); return; }
    const url = editingPlc.value
        ? API + '/plcs/' + editingPlc.value.id
        : API + '/plcs'
    const method = editingPlc.value ? 'PUT' : 'POST'

    plcForm.read_port = parseInt(plcForm.read_port) || 0

    const body = {
        facility_name: plcForm.name,
        maker: plcForm.make,
        driver: plcForm.series || plcForm.make,
        ip_address: plcForm.ip_address,
        comtype: plcForm.protocol || detectedProtocol.value,
        port: plcForm.read_port
    }

    try {
        const res = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        })
        if (res.ok) {
            toast(editingPlc.value ? 'PLC updated' : 'PLC created', 'success')
            showPlcModal.value = false
            await fetchPLCs()
            await fetchPLCHealth()
        } else {
            const err = await res.json()
            toast(err.error || 'Failed to save PLC', 'error')
        }
    } catch (_) { toast('Network error', 'error') }
}

async function scanAllPLCPorts() {
    isScanningPorts.value = true
    toast('Scanning PLC ports...', 'info')
    
    // Convert to parallel promises
    const promises = plcs.value.map(async (plc) => {
        try {
            const res = await fetch(`${API}/plcs/${plc.id}/scan`, {
                method: 'POST',
                headers: { 'Authorization': 'Bearer ' + token.value }
            })
            if (res.ok) {
                const data = await res.json()
                plc.openPorts = data.ports || []
            } else {
                plc.openPorts = []
            }
        } catch (e) {
            plc.openPorts = []
        }
    })
    
    await Promise.all(promises)
    isScanningPorts.value = false
    toast('Port scan complete', 'success')
}

async function scanSinglePLC(plc) {
    plc.isScanning = true
    try {
        const res = await fetch(`${API}/plcs/${plc.id}/scan`, {
            method: 'POST',
            headers: { 'Authorization': 'Bearer ' + token.value }
        })
        if (res.ok) {
            const data = await res.json()
            plc.openPorts = data.ports || []
            toast('Ports scanned for ' + plc.ip_address, 'success')
        } else {
            plc.openPorts = []
            toast('Scan failed for ' + plc.ip_address, 'error')
        }
    } catch (e) {
        plc.openPorts = []
        toast('Network error during scan', 'error')
    } finally {
        plc.isScanning = false
    }
}

async function quickUpdatePort(plc, newPort) {
    try {
        const payload = { ...plc, port: parseInt(newPort) }
        const res = await fetch(`${API}/plcs/${plc.id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token.value },
            body: JSON.stringify(payload)
        })
        if (res.ok) {
            toast(`Port updated to ${newPort}`, 'success')
            plc.port = parseInt(newPort)
            if (plc.openPorts) {
                plc.openPorts = [...plc.openPorts]
            }
        } else {
            toast('Failed to update port', 'error')
        }
    } catch (e) {
        toast('Network error updating port', 'error')
    }
}

const isPortInUse = (portStr) => {
    const p = parseInt(portStr)
    return plcs.value.some(plc => plc.port === p)
}

async function deletePlc(p) {
    if (!confirm('Delete PLC "' + p.name + '"?')) return
    try {
        const res = await fetch(API + '/plcs/' + p.id, { method: 'DELETE' })
        if (res.ok) {
            toast('PLC deleted', 'success')
            await fetchPLCs()
            await fetchPLCHealth()
        } else {
            const err = await res.json()
            toast(err.error || 'Delete failed', 'error')
        }
    } catch (_) { toast('Delete failed', 'error') }
}

function protocolForMake(make) {
    return protocolMap[make] || ''
}

function onMakeChange() {
    if (!plcForm.protocol || plcForm.protocol === detectedProtocol.value || !detectedProtocol.value) {
        plcForm.protocol = protocolForMake(plcForm.make)
    }
    plcForm.series = ''
    if (plcForm.ip_address) {
        scanOpenPorts()
    }
}

// ===================== ROBOTS =====================
const robots = ref([])
const robotSearch = ref('')
const showRobotModal = ref(false)
const editingRobot = ref(null)
const robotForm = reactive({ name: '', plc_id: '', ip_address: '', model_id: '' })

const plcOptions = computed(() => {
    const opts = []
    for (const p of health.value) {
        opts.push({ id: p.id, name: p.name, ip_address: p.ip_address, type: p.type })
    }
    return opts
})

const filteredRobots = computed(() => {
    if (!robotSearch.value) return robots.value
    const q = robotSearch.value.toLowerCase()
    return robots.value.filter(r =>
        r.name?.toLowerCase().includes(q) ||
        r.ip_address?.includes(q) ||
        r.plc_id?.toLowerCase().includes(q) ||
        r.model_id?.toLowerCase().includes(q)
    )
})

async function fetchRobots() {
    loading.robots = true
    try {
        const res = await fetch(API + '/robots')
        if (res.ok) { const d = await res.json(); robots.value = d.data || d || [] }
    } catch (_) {}
    finally { loading.robots = false }
}

function openAddRobot() {
    editingRobot.value = null
    Object.assign(robotForm, { name: '', plc_id: '', ip_address: '', model_id: '' })
    showRobotModal.value = true
}

function openEditRobot(r) {
    editingRobot.value = r
    Object.assign(robotForm, { name: r.name, plc_id: r.plc_id, ip_address: r.ip_address, model_id: r.model_id || '' })
    showRobotModal.value = true
}

async function saveRobot() {
    if (!robotForm.name) { toast('Robot name is required', 'error'); return; }
    const url = editingRobot.value
        ? API + '/robots/' + editingRobot.value.id
        : API + '/robots'
    const method = editingRobot.value ? 'PUT' : 'POST'
    try {
        const res = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(robotForm)
        })
        if (res.ok) {
            toast(editingRobot.value ? 'Robot updated' : 'Robot created', 'success')
            showRobotModal.value = false
            await fetchRobots()
        } else {
            const err = await res.json()
            toast(err.error || 'Failed to save robot', 'error')
        }
    } catch (_) { toast('Network error', 'error') }
}

async function deleteRobot(r) {
    if (!confirm('Delete robot "' + r.name + '"?')) return
    try {
        const res = await fetch(API + '/robots/' + r.id, { method: 'DELETE' })
        if (res.ok) {
            toast('Robot deleted', 'success')
            await fetchRobots()
        } else {
            const err = await res.json()
            toast(err.error || 'Delete failed', 'error')
        }
    } catch (_) { toast('Delete failed', 'error') }
}

// ===================== ROBOT MODELS =====================
const robotModels = ref([])
const showModelModal = ref(false)
const modelForm = reactive({ manufacturer: '', name: '' })

// ===================== COLUMN VISIBILITY =====================
const plcColDefs = [
	{ key: 'facility_name', label: 'Facility Name', def: true },
	{ key: 'driver', label: 'Driver', def: true },
	{ key: 'ip_address', label: 'IP Address', def: true },
	{ key: 'comtype', label: 'Com Type', def: false },
	{ key: 'rack', label: 'Rack', def: false },
	{ key: 'slot', label: 'Slot', def: false },
	{ key: 'port', label: 'Port', def: true },
	{ key: 'alarm_port', label: 'Alarm Port', def: false },
	{ key: 'maker', label: 'Maker', def: true },
]
const plcVisibleCols = ref(plcColDefs.filter(c => c.def).map(c => c.key))
const plcShowCols = ref(false)

const robotColDefs = [
    { key: 'id', label: 'ID', def: false },
    { key: 'name', label: 'Name', def: true },
    { key: 'plc_id', label: 'PLC ID', def: true },
    { key: 'ip_address', label: 'IP Address', def: true },
    { key: 'model_id', label: 'Model ID', def: false },
]
const robotVisibleCols = ref(robotColDefs.filter(c => c.def).map(c => c.key))
const robotShowCols = ref(false)

const tagColDefs = [
	{ key: 'id', label: 'ID', def: false },
	{ key: 'tag_address', label: 'Tag Address', def: true },
	{ key: 'plc_ip', label: 'PLC IP', def: true },
	{ key: 'tag_name', label: 'Tag Name', def: true },
	{ key: 'fac_name', label: 'Facility', def: false },
	{ key: 'robot_id', label: 'Robot ID', def: false },
	{ key: 'plc_id', label: 'PLC ID', def: false },
	{ key: 'data_type', label: 'Data Type', def: true },
	{ key: 'comment', label: 'Comment', def: true },
	{ key: 'action', label: 'Action', def: false },
	{ key: 'screen', label: 'Screen', def: false },
	{ key: 'svg_element', label: 'SVG', def: false },
	{ key: 'true_condition_color', label: 'True Color', def: false },
	{ key: 'false_condition_color', label: 'False Color', def: false },
	{ key: 'blinking', label: 'Blinking', def: false },
	{ key: 'refresh_rate', label: 'Refresh Rate', def: false },
	{ key: 'value', label: 'Value', def: true },
]
const tagVisibleCols = ref(tagColDefs.filter(c => c.def).map(c => c.key))
const tagShowCols = ref(false)

function toggleCol(arr, key) {
	const i = arr.indexOf(key)
	if (i >= 0) arr.splice(i, 1)
	else arr.push(key)
}

async function fetchRobotModels() {
    try {
        const res = await fetch(API + '/robot-models')
        if (res.ok) { const d = await res.json(); robotModels.value = d.data || d || [] }
    } catch (_) {}
}

function openAddModel() {
    modelForm.manufacturer = ''
    modelForm.name = ''
    showModelModal.value = true
    fetchRobotModels()
}

async function saveModel() {
    if (!modelForm.manufacturer || !modelForm.name) {
        toast('Manufacturer and model name required', 'error')
        return
    }
    try {
        const res = await fetch(API + '/robot-models', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(modelForm)
        })
        if (res.ok) {
            toast('Model added', 'success')
            modelForm.manufacturer = ''
            modelForm.name = ''
            await fetchRobotModels()
        }
    } catch (_) { toast('Failed to add model', 'error') }
}

async function deleteModel(m) {
    if (!confirm('Delete model "' + m.name + '"?')) return
    try {
        const res = await fetch(API + '/robot-models/' + m.id, { method: 'DELETE' })
        if (res.ok) {
            toast('Model deleted', 'success')
            await fetchRobotModels()
        } else {
            const err = await res.json()
            toast(err.error || 'Delete failed', 'error')
        }
    } catch (_) { toast('Delete failed', 'error') }
}

// ===================== TAGS =====================
const tags = ref([])
const tagSearch = ref('')
const tagPage = ref(1)
const tagPerPage = 50
const tagTotal = ref(0)
const tagSortBy = ref('id')
const tagSortOrder = ref('asc')
const tagMaxPage = computed(() => Math.max(1, Math.ceil(tagTotal.value / tagPerPage)))
const exportURL = ref(API + '/tags/export')

const mitsubishiDataTypes = [
    'bit', 'word', 'dword', 'int', 'dint', 'real', 'string', 'timer', 'counter'
]

const showTagModal = ref(false)
const editingTag = ref(null)
const selectedRobotId = ref('')
const tagForm = reactive({
    tag_address: '', data_type: '', plc_ip: '', tag_name: '', fac_name: '',
    robot_id: '', plc_id: '', comment: '', action: '', screen: '',
    svg_element: false, true_condition_color: '', false_condition_color: '',
    blinking: false, refresh_rate: 0
})

function onRobotChange() {
    const r = robots.value.find(r => r.id === selectedRobotId.value)
    if (r) {
        tagForm.robot_id = r.id
        tagForm.plc_id = r.plc_id
        tagForm.plc_ip = r.plc?.ip_address || ''
    } else {
        tagForm.robot_id = ''
        tagForm.plc_id = ''
        tagForm.plc_ip = ''
    }
}

let searchTimer = null
function debounceSearchTags() {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
        tagPage.value = 1
        fetchTags()
    }, 300)
}

async function fetchTags(background = false) {
    if (!background) loading.tags = true
    try {
        const params = new URLSearchParams({ 
            page: tagPage.value.toString(), 
            per_page: tagPerPage.toString(),
            sort_by: tagSortBy.value,
            order: tagSortOrder.value
        })
        if (tagSearch.value) params.set('search', tagSearch.value)
        const res = await fetch(API + '/tags?' + params.toString())
        if (res.ok) {
            const data = await res.json()
            tags.value = data.data || []
            tagTotal.value = data.total || 0
        }
    } catch (_) {}
    finally { if (!background) loading.tags = false }
}

function toggleTagSort(col) {
    if (tagSortBy.value === col) {
        tagSortOrder.value = tagSortOrder.value === 'asc' ? 'desc' : 'asc'
    } else {
        tagSortBy.value = col
        tagSortOrder.value = 'asc'
    }
    fetchTags()
}

function openAddTag() {
    editingTag.value = null
    selectedRobotId.value = ''
    Object.assign(tagForm, {
        tag_address: '', data_type: '', plc_ip: '', tag_name: '', fac_name: '',
        robot_id: '', plc_id: '', comment: '', action: '', screen: '',
        svg_element: false, true_condition_color: '', false_condition_color: '',
        blinking: false, refresh_rate: 0
    })
    showTagModal.value = true
}

function openEditTag(t) {
    editingTag.value = t
    selectedRobotId.value = t.robot_id || ''
    Object.assign(tagForm, {
        tag_address: t.tag_address,
        data_type: t.data_type || '',
        plc_ip: t.plc_ip || '',
        tag_name: t.tag_name || '',
        fac_name: t.fac_name || '',
        robot_id: t.robot_id || '',
        plc_id: t.plc_id || '',
        comment: t.comment || '',
        action: t.action || '',
        screen: t.screen || '',
        svg_element: !!t.svg_element,
        true_condition_color: t.true_condition_color || '',
        false_condition_color: t.false_condition_color || '',
        blinking: !!t.blinking,
        refresh_rate: t.refresh_rate || 0
    })
    showTagModal.value = true
}

async function saveTag() {
    if (!tagForm.tag_address) { toast('Tag address is required', 'error'); return; }
    const url = editingTag.value
        ? API + '/tags/' + editingTag.value.id
        : API + '/tags'
    const method = editingTag.value ? 'PUT' : 'POST'
    try {
        const res = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(tagForm)
        })
        if (res.ok) {
            toast(editingTag.value ? 'Tag updated' : 'Tag created', 'success')
            showTagModal.value = false
            await fetchTags()
        } else {
            const err = await res.json()
            toast(err.error || 'Failed to save tag', 'error')
        }
    } catch (_) { toast('Network error', 'error') }
}

async function deleteTag(t) {
    if (!confirm('Delete tag ' + t.tag_address + '?')) return
    try {
        const res = await fetch(API + '/tags/' + t.id, { method: 'DELETE' })
        if (res.ok) {
            toast('Tag deleted', 'success')
            await fetchTags()
        } else {
            const err = await res.json()
            toast(err.error || 'Delete failed', 'error')
        }
    } catch (_) { toast('Delete failed', 'error') }
}

async function deleteAllTags() {
    if (!confirm('Delete ALL ' + tagTotal.value + ' tags? This cannot be undone.')) return
    try {
        const res = await fetch(API + '/tags', { method: 'DELETE' })
        if (res.ok) {
            const data = await res.json()
            toast('Deleted ' + (data.deleted || 0) + ' tags', 'success')
            tagPage.value = 1
            await fetchTags()
        } else {
            const err = await res.json()
            toast(err.error || 'Delete all failed', 'error')
        }
    } catch (_) { toast('Delete all failed', 'error') }
}

// ===================== CSV IMPORT =====================
const showImportModal = ref(false)
const importFile = ref(null)
const importPreview = ref([])
const csvInput = ref(null)

function openImportCSV() {
    importFile.value = null
    importPreview.value = []
    showImportModal.value = true
}

function onFileSelect(e) {
    const file = e.target.files[0]
    if (!file) return
    importFile.value = file

    const reader = new FileReader()
    reader.onload = function(evt) {
        const csv = evt.target.result
        const parsed = Papa.parse(csv, { header: true, skipEmptyLines: true })
        if (parsed.errors.length) {
            toast('CSV parse error: ' + parsed.errors[0].message, 'error')
            return
        }
        importPreview.value = parsed.data.slice(0, 200)
    }
    reader.readAsText(file)
}

async function confirmImport() {
    if (!importFile.value) return
    const formData = new FormData()
    formData.append('file', importFile.value)
    try {
        const res = await fetch(API + '/tags/import', { method: 'POST', body: formData })
        if (res.ok) {
            const data = await res.json()
            toast('Imported ' + data.count + ' tags' + (data.errors?.length ? ' (' + data.errors.length + ' errors)' : ''), 'success')
            showImportModal.value = false
            await fetchTags()
        } else {
            const err = await res.json()
            toast(err.error || 'Import failed', 'error')
        }
    } catch (_) { toast('Import network error', 'error') }
}

// ===================== AUTH =====================
const user = ref({ name: 'Temp Admin', role: 'superadmin', email: 'admin@local' })
const token = ref('temp-bypass-token')
const showAuthModal = ref(false)
const authTab = ref('login')
const authForm = reactive({ email: '', password: '', name: '' })
const authError = ref('')

const isLoggedIn = computed(() => !!token.value)
const isSuperadmin = computed(() => user.value?.role === 'superadmin')
const isCoAdmin = computed(() => user.value?.role === 'co-admin')
const isAdmin = computed(() => isSuperadmin.value || isCoAdmin.value)

function loadUser() {
    if (token.value === 'temp-bypass-token') return
    if (!token.value) return
    fetch(API + '/auth/profile', { headers: { Authorization: 'Bearer ' + token.value } })
        .then(r => {
            if (!r.ok) {
                logout(true)
                return null
            }
            return r.json()
        })
        .then(d => {
            if (d) {
                user.value = d.user
                fetchPermissions()
            }
        })
        .catch(() => { logout(true) })
}

async function handleAuth() {
    authError.value = ''
    try {
        const res = await fetch(API + '/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email: authForm.email, password: authForm.password })
        })
        const data = await res.json()
        if (!res.ok) { authError.value = data.error || 'Authentication failed'; return }
        token.value = data.token
        user.value = data.user
        localStorage.setItem('token', data.token)
        showAuthModal.value = false
        await fetchPermissions()
        toast('Welcome ' + data.user.name, 'success')
    } catch (_) { authError.value = 'Network error' }
}

const permittedModules = ref([])

async function fetchPermissions() {
    if (!token.value || token.value === 'temp-bypass-token') return
    try {
        const res = await fetch(API + '/auth/my-permissions', {
            headers: { Authorization: 'Bearer ' + token.value }
        })
        if (res.ok) permittedModules.value = await res.json()
    } catch (_) {}
}

function hasModule(name) {
    if (!isLoggedIn.value) return false
    if (token.value === 'temp-bypass-token') return true
    return permittedModules.value.some(m => m.name === name)
}

function logout(silent = false) {
    // Temp bypass: Prevent actual logout
    toast('Logout disabled temporarily', 'info')
}

function openAuth(tab = 'login') {
    authTab.value = tab
    authForm.email = ''
    authForm.password = ''
    authForm.name = ''
    authError.value = ''
    showAuthModal.value = true
}

// ===================== USER MANAGEMENT (Admin) =====================
const users = ref([])
const roles = ref([])
const adminLoading = ref(false)

async function fetchUsers() {
    if (!isAdmin.value) return
    adminLoading.value = true
    try {
        const res = await fetch(API + '/admin/users', { headers: { Authorization: 'Bearer ' + token.value } })
        if (res.ok) users.value = await res.json()
    } catch (_) {}
    adminLoading.value = false
}

async function fetchRoles() {
    try {
        const res = await fetch(API + '/admin/roles', { headers: { Authorization: 'Bearer ' + token.value } })
        if (res.ok) roles.value = await res.json()
    } catch (_) {}
}

async function createUserByAdmin() {
    if (!authForm.email || !authForm.password || !authForm.name) {
        authError.value = 'All fields required'; return
    }
    authError.value = ''
    try {
        const res = await fetch(API + '/admin/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value },
            body: JSON.stringify({ email: authForm.email, password: authForm.password, name: authForm.name, role_id: authForm.role_id })
        })
        const data = await res.json()
        if (!res.ok) { authError.value = data.error || 'Failed'; return }
        await fetchUsers()
        showAuthModal.value = false
        toast('User created', 'success')
    } catch (_) { authError.value = 'Network error' }
}

// ===================== PERMISSIONS (Admin) =====================
const modules = ref([])
const rolePerms = ref([])
const adminSubTab = ref('users')

async function fetchModules() {
    try {
        const res = await fetch(API + '/admin/modules', {
            headers: { Authorization: 'Bearer ' + token.value }
        })
        if (res.ok) modules.value = await res.json()
    } catch (_) {}
}

async function fetchRolePermissions(roleId) {
    try {
        const res = await fetch(API + '/admin/permissions/' + roleId, {
            headers: { Authorization: 'Bearer ' + token.value }
        })
        if (res.ok) rolePerms.value = await res.json()
    } catch (_) {}
}

function hasModulePerm(moduleId) {
    return rolePerms.value.some(p => p.module_id === moduleId && p.can_access)
}

function rolePermsByRole(roleId, moduleId) {
    const p = rolePerms.value.find(p => p.role_id === roleId && p.module_id === moduleId)
    return p ? p.can_access : false
}

async function fetchAllRolePerms() {
    rolePerms.value = []
    for (const r of roles.value) {
        try {
            const res = await fetch(API + '/admin/permissions/' + r.id, {
                headers: { Authorization: 'Bearer ' + token.value }
            })
            if (res.ok) {
                const perms = await res.json()
                rolePerms.value.push(...perms)
            }
        } catch (_) {}
    }
}

async function toggleModulePerm(roleId, moduleId, canAccess) {
    try {
        await fetch(API + '/admin/permissions', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value },
            body: JSON.stringify({ role_id: roleId, module_id: moduleId, can_access: canAccess })
        })
        await fetchRolePermissions(roleId)
        toast('Permission updated', 'success')
    } catch (_) { toast('Failed to update permission', 'error') }
}

async function updateUserRole(userId, roleId) {
    try {
        await fetch(API + '/admin/users/' + userId, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value },
            body: JSON.stringify({ role_id: roleId })
        })
        await fetchUsers()
        toast('User role updated', 'success')
    } catch (_) { toast('Failed to update role', 'error') }
}

async function toggleUserActive(userId, active) {
    try {
        await fetch(API + '/admin/users/' + userId, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value },
            body: JSON.stringify({ active: !active })
        })
        await fetchUsers()
    } catch (_) {}
}

async function deleteUserById(userId) {
    if (!confirm('Delete this user?')) return
    try {
        await fetch(API + '/admin/users/' + userId, { method: 'DELETE', headers: { Authorization: 'Bearer ' + token.value } })
        await fetchUsers()
        toast('User deleted', 'success')
    } catch (_) { toast('Failed to delete user', 'error') }
}

// ===================== GENERAL =====================
function openSwagger() {
    window.open('/swagger/index.html', '_blank')
}

function refreshAll() {
    fetchPLCHealth()
    fetchTagCount()
    fetchPollerStats()
    fetchPLCs()
    fetchRobots()
    fetchTags()
    toast('Refreshed all data', 'info')
}

// ===================== LIFECYCLE =====================
const starCanvas = ref(null)
let starAnimId = null

function initStars() {
    const canvas = starCanvas.value
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight

    const stars = Array.from({ length: 200 }, () => ({
        x: Math.random() * canvas.width,
        y: Math.random() * canvas.height,
        r: Math.random() * 1.5 + 0.5,
        dx: (Math.random() - 0.5) * 0.3,
        dy: (Math.random() - 0.5) * 0.3
    }))

    function draw() {
        ctx.clearRect(0, 0, canvas.width, canvas.height)
        ctx.fillStyle = 'rgba(255,255,255,0.8)'
        stars.forEach(s => {
            s.x += s.dx
            s.y += s.dy
            if (s.x < 0 || s.x > canvas.width) s.dx *= -1
            if (s.y < 0 || s.y > canvas.height) s.dy *= -1
            ctx.beginPath()
            ctx.arc(s.x, s.y, s.r, 0, Math.PI * 2)
            ctx.fill()
        })
        starAnimId = requestAnimationFrame(draw)
    }
    draw()
}

onMounted(() => {
    if (!isLoggedIn.value) initStars()
    loadUser()
    fetchPLCHealth()
    fetchTagCount()
    fetchPollerStats()
    fetchPLCs()
    fetchRobots()
    fetchRobotModels()
    fetchTags()
    setInterval(fetchPollerStats, 1000)
    setInterval(fetchPLCHealth, 1000)
    setInterval(() => { if (tab.value === 'tags') fetchTags(true) }, 3000)
    window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
    if (starAnimId) cancelAnimationFrame(starAnimId)
    window.removeEventListener('resize', handleResize)
})

watch(globalSearch, (val) => {
    if (tab.value === 'plcs') plcSearch.value = val;
    if (tab.value === 'robots') robotSearch.value = val;
    if (tab.value === 'tags') { tagSearch.value = val; debounceSearchTags(); }
})
watch(tab, () => {
    globalSearch.value = '';
})

watch(isLoggedIn, (v) => {
    if (!v) nextTick(() => initStars())
    else { if (starAnimId) { cancelAnimationFrame(starAnimId); starAnimId = null } }
})

function handleResize() {
    const canvas = starCanvas.value
    if (canvas) {
        canvas.width = window.innerWidth
        canvas.height = window.innerHeight
    }
}

// ===================== BULK STRESS CHART LOGIC =====================
const bulkTestRunning = ref(false)
const bulkTestStatus = ref('')
const bulkTestStats = ref(null)
const showBulkHistoryModal = ref(false)

const offsetLimits = computed(() => {
    switch (bulkTestForm.value.prefix) {
        case 'M': return { min: 1, max: 2500, label: '1 - 2500' }
        case 'R': return { min: 2501, max: 5000, label: '2501 - 5000' }
        case 'D': return { min: 5001, max: 10000, label: '5001 - 10000' }
        default: return { min: 0, max: 65535, label: '0 - 65535' }
    }
})

const bulkTestForm = ref({
    plc_id: '',
    prefix: 'D',
    start: 0,
    count: 1000,
    value: '',
    duration: '0s'
})

function onBulkPrefixChange() {
    bulkTestForm.value.value = ''
}

const bulkTestStartInput = computed({
    get: () => bulkTestForm.value.start,
    set: (val) => {
        if (typeof val === 'string') {
            let s = val.trim().toUpperCase();
            const match = s.match(/^([A-Z]+)(\d+)$/);
            if (match) {
                let prefix = match[1];
                if (['D', 'M', 'R', 'W', 'B', 'ZR'].includes(prefix)) {
                    bulkTestForm.value.prefix = prefix;
                }
                bulkTestForm.value.start = parseInt(match[2], 10);
            } else {
                let num = parseInt(s, 10);
                if (!isNaN(num)) bulkTestForm.value.start = num;
            }
        } else {
            bulkTestForm.value.start = val;
        }
    }
})
const bulkTestChartOption = ref({
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: ['Bulk Read Latency', 'Bulk Write Latency'] },
    yAxis: { type: 'value', name: 'Time (ms)' },
    series: [{
        data: [0, 0],
        type: 'bar',
        itemStyle: {
            color: function(params) {
                var colorList = ['#3498db','#e74c3c'];
                return colorList[params.dataIndex];
            }
        }
    }]
})

async function runBulkTest(op) {
    bulkTestRunning.value = true
    bulkTestStatus.value = `Running bulk ${op} test for ${bulkTestForm.value.count} tags on PLC...`
    bulkTestStats.value = null

    // Clear chart before test
    bulkTestChartOption.value = {
        ...bulkTestChartOption.value,
        series: [{
            ...bulkTestChartOption.value.series[0],
            data: [0, 0]
        }]
    }

    try {
        if (bulkTestForm.value.start < offsetLimits.value.min || bulkTestForm.value.start > offsetLimits.value.max) {
            bulkTestStatus.value = `Error: Invalid range! ${bulkTestForm.value.prefix} tags are mapped from ${offsetLimits.value.min} to ${offsetLimits.value.max}.`
            bulkTestRunning.value = false
            return
        }
        
        let url = `${API}/rtt-test/v2/bulk-test?plc_id=${bulkTestForm.value.plc_id}&prefix=${bulkTestForm.value.prefix}&start=${bulkTestForm.value.start}&count=${bulkTestForm.value.count}&value=${bulkTestForm.value.value}&duration=${bulkTestForm.value.duration}&op=${op}`
        
        const res = await fetch(url)
        const data = await res.json()
        if (data.error) {
            bulkTestStatus.value = "Error: " + data.error
        } else {
            if (op === 'write') {
                bulkTestStatus.value = `Success: You have written value '${bulkTestForm.value.value}' to ${bulkTestForm.value.count} tags starting from ${bulkTestForm.value.prefix}${bulkTestForm.value.start}.`
            } else {
                bulkTestStatus.value = `Success: You have fetched ${bulkTestForm.value.count} tags starting from ${bulkTestForm.value.prefix}${bulkTestForm.value.start}.`
            }
            bulkTestStats.value = data
            
            let newData = [...bulkTestChartOption.value.series[0].data]
            if (op === 'read' || op === 'both') newData[0] = data.read_latency_ms
            if (op === 'write' || op === 'both') newData[1] = data.write_latency_ms
            
            bulkTestChartOption.value = {
                ...bulkTestChartOption.value,
                series: [{
                    ...bulkTestChartOption.value.series[0],
                    data: newData
                }]
            }
        }
    } catch (e) {
        bulkTestStatus.value = "Network Error: Could not reach API"
    }
    bulkTestRunning.value = false
}

</script>

<template>
    <div id="app" class="theme-dark">
        <!-- Toast -->
        <div class="toast-container">
            <div v-for="(t,i) in toasts" :key="i" :class="'toast '+t.type">
                <ion-icon :name="t.icon"></ion-icon>
                <span>{{ t.msg }}</span>
            </div>
        </div>

        <!-- ===================== LOGIN PAGE ===================== -->
        <div v-if="!isLoggedIn" class="login-page">
            <canvas ref="starCanvas" class="login-canvas"></canvas>
            <div class="login-overlay"></div>
            <div class="login-card">
                <div class="login-brand">
                    <!-- <div style="display: flex; justify-content: center; align-items: center; gap: 1.5rem; margin-bottom: 1.5rem; background: rgba(255,255,255,0.8); padding: 10px; border-radius: 12px;">
                        <img src="/taikisha-full-logo.png" alt="Taikisha" height="40" style="object-fit: contain;">
                    </div> -->
                    <!--<h1>i-Tips</h1>-->
                    <h1>PlcNexus</h1>
                   <!-- <p class="login-subtitle">Device Gateway Admin</p>-->
                </div>
                <div v-if="authError" class="login-error">{{ authError }}</div>
                <div class="login-form">
                    <div class="login-field">
                        <label>Username / Email</label>
                        <input type="email" v-model="authForm.email" placeholder="Enter your email" @keyup.enter="handleAuth">
                    </div>
                    <div class="login-field">
                        <label>Password</label>
                        <input type="password" v-model="authForm.password" placeholder="Enter your password" @keyup.enter="handleAuth">
                    </div>
                    <button class="login-btn" @click="handleAuth">
                        <ion-icon name="log-in-outline"></ion-icon>
                        Login
                    </button>
                </div>
            </div>
        </div>

        <!-- ===================== LOGGED IN UI ===================== -->
        <template v-if="isLoggedIn">
        <aside class="sidebar" :class="{ collapsed: isSidebarCollapsed }">
            <div class="sidebar-header">
                <div class="logo">
                    <ion-icon name="cube" class="logo-icon"></ion-icon>
                    <h2 v-show="!isSidebarCollapsed">Plc<span>Nexus</span></h2>
                </div>
                <button class="toggle-btn" @click="isSidebarCollapsed = !isSidebarCollapsed">
                    <ion-icon :name="isSidebarCollapsed ? 'chevron-forward' : 'chevron-back'"></ion-icon>
                </button>
            </div>
            <nav class="sidebar-nav">
                <a v-if="hasModule('dashboard')" class="nav-item" :class="{ active: tab === 'dashboard' }" @click.prevent="tab='dashboard'">
                    <ion-icon name="grid-outline"></ion-icon>
                    <span v-show="!isSidebarCollapsed">Dashboard</span>
                </a>



                <div class="nav-label" v-show="!isSidebarCollapsed" v-if="hasModule('plcs') || hasModule('robots') || hasModule('tags')">MANAGEMENT</div>
                <a v-if="hasModule('plcs')" class="nav-item" :class="{ active: tab === 'plcs' }" @click.prevent="tab='plcs'">
                    <ion-icon name="server-outline"></ion-icon>
                    <span v-show="!isSidebarCollapsed">PLCs</span>
                </a>
                <a v-if="hasModule('robots')" class="nav-item" :class="{ active: tab === 'robots' }" @click.prevent="tab='robots'">
                    <ion-icon name="hardware-chip-outline"></ion-icon>
                    <span v-show="!isSidebarCollapsed">Robots</span>
                </a>
                <a v-if="hasModule('tags')" class="nav-item" :class="{ active: tab === 'tags' }" @click.prevent="tab='tags'">
                    <ion-icon name="pricetags-outline"></ion-icon>
                    <span v-show="!isSidebarCollapsed">Tags</span>
                </a>

            </nav>
            <!-- User Profile / Auth -->
            <div class="sidebar-user">
                <template v-if="isLoggedIn && user">
                    <div class="user-avatar" :style="{ background: isSuperadmin ? 'var(--accent-blue)' : isCoAdmin ? 'var(--accent-purple)' : 'var(--accent-cyan)' }">
                        {{ user.name.charAt(0).toUpperCase() }}
                    </div>
                    <div class="user-info" v-show="!isSidebarCollapsed">
                        <div class="user-name">{{ user.name }}</div>
                        <div class="user-role-badge" :class="user.role">{{ user.role_data?.label || user.role }}</div>
                    </div>
                    <button class="user-logout-btn" @click="logout" :title="isSidebarCollapsed ? 'Logout' : ''">
                        <ion-icon name="log-out-outline"></ion-icon>
                    </button>
                </template>
                <template v-else>
                    <button class="btn btn-primary btn-sm" style="width:100%;justify-content:center;margin:0 12px" @click="openAuth('login')">
                        <ion-icon name="log-in-outline"></ion-icon>
                        <span v-show="!isSidebarCollapsed">Sign In</span>
                    </button>
                </template>
            </div>
        </aside>

        <!-- Main -->
        <main class="main-content">
            <header class="top-header">
                <div class="search-box">
                    <ion-icon name="search-outline"></ion-icon>
                    <input type="text" v-model="globalSearch" placeholder="Search...">
                </div>
                <div class="header-actions">
                    <div class="status-badge" :class="apiHealthy ? 'connected' : 'disconnected'">
                        <span class="pulse-dot"></span>
                        {{ apiHealthy ? 'System Online' : 'Disconnected' }}
                    </div>
                    <button class="icon-btn" @click="refreshAll"><ion-icon name="refresh-outline"></ion-icon></button>
                </div>
            </header>

            <!-- ===================== DEMO RTT VIEW ===================== -->
            <div class="view-content" v-if="tab === 'demo_rtt'" style="padding: 0; height: calc(100vh - 64px); display: flex; flex-direction: column;">
                <iframe src="demo_rtt.html" style="width: 100%; height: 100%; border: none; flex-grow: 1;"></iframe>
            </div>

            <!-- ===================== DASHBOARD ===================== -->
            <div class="view-content" v-if="tab === 'dashboard'">
                <div class="page-header">
                    <div>
                        <h1>PLC Health Dashboard</h1>
                        <p class="subtitle">Real-time connectivity status of all devices</p>
                    </div>
                    <button class="btn btn-primary" @click="fetchPLCHealth">
                        <ion-icon name="refresh-outline"></ion-icon> Refresh
                    </button>
                </div>
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-icon info-bg"><ion-icon name="server-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Total PLCs</h3>
                            <h2>{{ health.length }}</h2>
                        </div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-icon" :class="offlineCount > 0 ? 'danger-bg' : 'success-bg'"><ion-icon name="wifi-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Connectivity</h3>
                            <h2 style="font-size: 1.25rem;">
                                <span style="color: var(--accent-green);">{{ onlineCount }} On</span> / 
                                <span style="color: var(--accent-red);">{{ offlineCount }} Off</span>
                            </h2>
                            <div class="stat-sub" :style="{ color: offlineCount > 0 ? 'var(--accent-red)' : 'var(--accent-green)' }">
                                {{ offlineCount > 0 ? 'Some PLCs Offline' : 'All Systems Online' }}
                            </div>
                        </div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-icon primary-bg"><ion-icon name="hardware-chip-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Total Robots</h3>
                            <h2>{{ robots.length }}</h2>
                            <div class="stat-sub">configured</div>
                        </div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-icon mitsubishi-bg"><ion-icon name="pricetags-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Total Tags</h3>
                            <h2>{{ tagCount }}</h2>
                            <div class="stat-sub">in tag table</div>
                        </div>
                    </div>
                </div>
                <div class="stats-grid" style="margin-top:1rem">
                    <div class="stat-card">
                        <div class="stat-icon primary-bg"><ion-icon name="pulse-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Read RTT</h3>
                            <h2>{{ (pollerStats.read_latency_ms || 0).toFixed(2) }}<small>ms</small></h2>
                            <div class="stat-sub">{{ pollerStats.read_ops || 0 }} total ops</div>
                        </div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-icon warning-bg"><ion-icon name="swap-horizontal-outline"></ion-icon></div>
                        <div class="stat-details">
                            <h3>Write RTT</h3>
                            <h2>{{ (pollerStats.write_latency_ms || 0).toFixed(2) }}<small>ms</small></h2>
                            <div class="stat-sub">{{ pollerStats.write_ops || 0 }} total ops</div>
                        </div>
                    </div>
                    <div class="stat-card" @click="toggleErrorSidebar" style="cursor:pointer">
                        <div class="stat-icon" :class="pollerStats.error_count > 0 ? 'danger-bg' : 'success-bg'">
                            <ion-icon name="bug-outline"></ion-icon>
                        </div>
                        <div class="stat-details">
                            <h3>Errors</h3>
                            <h2 :style="{ color: pollerStats.error_count > 0 ? 'var(--accent-red)' : 'var(--accent-green)' }">
                                {{ pollerStats.error_count || 0 }}
                            </h2>
                            <div class="stat-sub">poller errors &middot; click to view</div>
                        </div>
                    </div>
                </div>
                <div class="section-label">
                    <ion-icon name="pulse-outline"></ion-icon>
                    <span>PLCs ({{ onlineCount }} online, {{ offlineCount }} offline)</span>
                </div>
                <div class="health-grid">
                    <!-- Dedicated MC Simulator Card -->
                    <div v-if="mcSimulator" class="health-card card-simulator" :class="mcSimulator.status">
                        <div class="health-card-header">
                            <div>
                                <h3>
                                    <svg style="width:22px;height:22px;vertical-align:middle;margin-right:6px;color:var(--accent-cyan)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                                        <line x1="8" y1="21" x2="16" y2="21"/>
                                        <line x1="12" y1="17" x2="12" y2="21"/>
                                    </svg>
                                    MC Simulator
                                </h3>
                                <span class="plc-type" style="color:var(--accent-cyan)">Mitsubishi / MC Protocol</span>
                            </div>
                            <div class="health-status">
                                <span class="dot" :class="mcSimulator.status"></span>
                                {{ mcSimulator.status === 'online' ? 'Connected' : 'Slow' }}
                            </div>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="globe-outline"></ion-icon>
                            <span class="code-badge">{{ mcSimulator.ip_address }}</span>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="git-branch-outline"></ion-icon>
                            Port <span class="code-badge">{{ mcSimulator.port }}</span>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="pulse-outline"></ion-icon>
                            <span style="color:var(--text-secondary)">RTT</span>
                            <span class="rtt-badge rtt-good">{{ mcSimulator.latency_ms }}ms</span>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="pulse-outline"></ion-icon>
                            <span style="color:var(--text-secondary)">Polling</span>
                            <span class="code-badge" style="color:var(--accent-cyan)">{{ pollerStats.read_ops || 0 }} ops</span>
                        </div>
                        <div class="health-latency">
                            <span>Health</span>
                            <span style="color:var(--accent-green)"><strong>Healthy Online</strong></span>
                        </div>
                    </div>

                    <!-- All Other PLCs -->
                    <div v-for="plc in otherPLCs" :key="plc.id" class="health-card" :class="plc.status">
                        <div class="health-card-header">
                            <div>
                                <h3>
                                    <svg style="width:22px;height:22px;vertical-align:middle;margin-right:6px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :style="{ color: plc.status === 'online' ? 'var(--accent-green)' : 'var(--accent-red)' }">
                                        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                                        <line x1="8" y1="21" x2="16" y2="21"/>
                                        <line x1="12" y1="17" x2="12" y2="21"/>
                                    </svg>
                                    {{ plc.name || 'Unnamed' }}
                                </h3>
                                <span class="plc-type">{{ plc.type }}</span>
                            </div>
                            <div class="health-status">
                                <span class="dot" :class="plc.status"></span>
                                {{ plc.status === 'online' ? 'Connected' : 'Offline' }}
                            </div>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="globe-outline"></ion-icon>
                            <span class="code-badge">{{ plc.ip_address }}</span>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="git-branch-outline"></ion-icon>
                            Port <span class="code-badge">{{ plc.port }}</span>
                        </div>
                        <div class="health-detail" v-if="plc.status === 'online'">
                            <ion-icon name="pulse-outline"></ion-icon>
                            <span style="color:var(--text-secondary)">RTT</span>
                            <span class="rtt-badge" :class="plc.latency_ms > 100 ? 'rtt-slow' : 'rtt-good'">
                                {{ plc.latency_ms }}ms
                            </span>
                        </div>
                        <div class="health-latency">
                            <span>Health</span>
                            <span :style="{ color: plc.status === 'online' ? 'var(--accent-green)' : 'var(--accent-red)' }">
                                <strong>{{ plc.status === 'online' ? 'Healthy Online' : 'Unhealthy' }}</strong>
                            </span>
                        </div>
                    </div>
                    <div v-if="otherPLCs.length === 0 && !mcSimulator" class="empty-state" style="grid-column:1/-1">
                        <div class="empty-icon"><ion-icon name="server-outline"></ion-icon></div>
                        <h2>No PLCs</h2>
                        <p>No PLCs configured. Add a PLC to get started.</p>
                    </div>
                </div>
                <!-- ===================== TRENDING CHART ===================== -->
                <div class="page-header" style="margin-top: 2rem;">
                    <div>
                        <h1>Real-Time RTT Trending</h1>
                        <p class="subtitle">Historical Read/Write RTT latency analysis</p>
                    </div>
                    <div style="display: flex; gap: 1rem; align-items: center; background: rgba(56, 189, 248, 0.1); padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid rgba(56, 189, 248, 0.3); box-shadow: 0 4px 12px rgba(56, 189, 248, 0.08);">
                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                            <ion-icon name="time-outline" style="color:#0284c7; font-size: 1.1rem;"></ion-icon>
                            <select v-model="rttTimeRange" style="background: transparent; color: var(--text-primary); border: none; outline: none; cursor: pointer; font-size: 0.95rem; font-weight: 500;">
                                <option style="background: var(--bg-card); color: var(--text-primary);" v-for="t in timeRangeOptions" :key="t.value" :value="t.value">{{ t.label }}</option>
                            </select>
                        </div>
                        <div style="width: 1px; height: 20px; background: rgba(56, 189, 248, 0.4);"></div>
                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                            <ion-icon name="layers-outline" style="color:#0284c7; font-size: 1.1rem;"></ion-icon>
                            <select v-model="rttHistoryLimit" style="background: transparent; color: var(--text-primary); border: none; outline: none; cursor: pointer; font-size: 0.95rem; font-weight: 500;">
                                <option style="background: var(--bg-card); color: var(--text-primary);" v-for="limit in historyLimitOptions" :key="limit" :value="limit">{{ limit }} Records</option>
                            </select>
                        </div>
                        <div style="width: 1px; height: 30px; background: rgba(56, 189, 248, 0.4); margin: 0 0.5rem;"></div>
                        <div style="display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap;">
                            <button 
                                @click="toggleChartPlc('Global')"
                                :style="chipStyle('Global')"
                                style="padding: 0.4rem 1rem; border-radius: 20px; font-size: 0.85rem; font-weight: 600; cursor: pointer; display: flex; align-items: center; gap: 0.4rem; transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);"
                            >
                                <ion-icon name="globe-outline"></ion-icon> Global Stats
                            </button>
                            <button 
                                v-for="p in health" :key="p.id"
                                @click="toggleChartPlc(p.id)"
                                :style="chipStyle(p.id)"
                                style="padding: 0.4rem 1rem; border-radius: 20px; font-size: 0.85rem; font-weight: 600; cursor: pointer; display: flex; align-items: center; gap: 0.4rem; transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);"
                            >
                                <ion-icon name="hardware-chip-outline"></ion-icon> {{ p.name || p.ip_address }}
                            </button>
                        </div>
                    </div>
                </div>
                <div class="table-card" style="padding: 1.5rem; margin-top: 1rem; border-top: 3px solid #38bdf8;">
                    <div style="height: 350px; position: relative;">
                        <v-chart :option="rttChartOption" autoresize />
                    </div>
                    
                    <div v-for="(metric, idx) in rttMetrics" :key="idx" style="margin-top: 2rem;">
                        <h4 style="margin-bottom: 0.5rem; color: #38bdf8;">{{ metric.label }}</h4>
                        <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
                            <div class="stat-card" style="flex: 1; min-width: 180px; padding: 1rem; background: rgba(56, 189, 248, 0.05); border: 1px solid rgba(56, 189, 248, 0.2); box-shadow: none;">
                                <div class="stat-icon" style="background: rgba(84, 112, 198, 0.15); color: #5470C6; width: 36px; height: 36px; font-size: 1.1rem; display: flex; align-items: center; justify-content: center;">
                                    <ion-icon name="arrow-down-outline"></ion-icon>
                                </div>
                                <div class="stat-details" style="margin-left: 0; width: 100%;">
                                    <h3 style="font-size: 0.85rem; margin-bottom: 0.5rem; color: var(--text-primary);">Read RTT (ms)</h3>
                                    <div style="display: flex; justify-content: space-between; font-size: 0.85rem;">
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Min</small><br><b>{{ metric.read.min }}</b></div>
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Avg</small><br><b style="color: #5470C6">{{ metric.read.avg }}</b></div>
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Max</small><br><b style="color:var(--accent-red)">{{ metric.read.max }}</b></div>
                                    </div>
                                </div>
                            </div>
                            <div class="stat-card" style="flex: 1; min-width: 180px; padding: 1rem; background: rgba(56, 189, 248, 0.05); border: 1px solid rgba(56, 189, 248, 0.2); box-shadow: none;">
                                <div class="stat-icon" style="background: rgba(238, 102, 102, 0.15); color: #EE6666; width: 36px; height: 36px; font-size: 1.1rem; display: flex; align-items: center; justify-content: center;">
                                    <ion-icon name="arrow-up-outline"></ion-icon>
                                </div>
                                <div class="stat-details" style="margin-left: 0; width: 100%;">
                                    <h3 style="font-size: 0.85rem; margin-bottom: 0.5rem; color: var(--text-primary);">Write RTT (ms)</h3>
                                    <div style="display: flex; justify-content: space-between; font-size: 0.85rem;">
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Min</small><br><b>{{ metric.write.min }}</b></div>
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Avg</small><br><b style="color: #EE6666">{{ metric.write.avg }}</b></div>
                                        <div><small style="color:var(--text-secondary); font-size:0.75rem;">Max</small><br><b style="color:var(--accent-red)">{{ metric.write.max }}</b></div>
                                    </div>
                                </div>
                            </div>
                            <div class="stat-card" style="flex: 1; min-width: 180px; padding: 1rem; background: rgba(56, 189, 248, 0.05); border: 1px solid rgba(56, 189, 248, 0.2); box-shadow: none;">
                                <div class="stat-icon info-bg" style="width: 36px; height: 36px; font-size: 1.1rem; display: flex; align-items: center; justify-content: center;">
                                    <ion-icon name="pricetags-outline"></ion-icon>
                                </div>
                                <div class="stat-details" style="margin-left: 0;">
                                    <h3 style="font-size: 0.85rem; margin-bottom: 0.5rem; color: var(--text-primary);">Tags Polled</h3>
                                    <b style="font-size: 1.4rem;">{{ metric.ops }}</b>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- ===================== BULK STRESS CHART ===================== -->
                <div class="page-header" style="margin-top: 2rem;">
                    <div>
                        <h1>Custom Range Stress Test</h1>
                        <p class="subtitle">Specify a precise tag range to benchmark bulk read and write latency.</p>
                    </div>
                </div>
                <div class="table-card" style="padding: 2rem; margin-top: 1rem; border-top: 3px solid #3498db;">
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1.5rem; margin-bottom: 2rem;">
                        <div class="login-field" style="margin-bottom: 0;">
                            <label style="color:var(--text-primary)">Target PLC</label>
                            <select v-model="bulkTestForm.plc_id" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                                <option value="" disabled>Select PLC</option>
                                <option v-for="p in health" :key="p.id" :value="p.id">{{ p.name || p.ip_address }}</option>
                            </select>
                        </div>
                        <div class="login-field" style="margin-bottom: 0;">
                            <label style="color:var(--text-primary)">Tag Type</label>
                            <select v-model="bulkTestForm.prefix" @change="onBulkPrefixChange" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                                <option value="D">D (INT Reg)</option>
                                <option value="R">R (String Reg)</option>
                                <option value="ZR">ZR (String Reg ZR)</option>
                                <option value="M">M (Bool Relay)</option>
                            </select>
                        </div>
                        <div class="login-field" style="margin-bottom: 0; position: relative;">
                            <label style="color:var(--text-primary)" title="The starting device number. Valid range depends on your Tag Type mapping.">
                                Start Offset <ion-icon name="information-circle-outline" style="vertical-align: middle; cursor: help; color: var(--accent-blue);"></ion-icon>
                                <span style="font-size: 0.7rem; color: var(--accent-blue); float: right; margin-top: 4px;">({{ offsetLimits.label }})</span>
                            </label>
                            <input type="text" v-model="bulkTestStartInput" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                        </div>
                        <div class="login-field" style="margin-bottom: 0; position: relative;">
                            <label style="color:var(--text-primary)" title="How many consecutive registers to test. Max 1000.">
                                Count (Range) <ion-icon name="information-circle-outline" style="vertical-align: middle; cursor: help; color: var(--accent-blue);"></ion-icon>
                            </label>
                            <input type="number" min="1" max="1000" v-model="bulkTestForm.count" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                        </div>
                        <div class="login-field" style="margin-bottom: 0;">
                            <label style="color:var(--text-primary)">Write Value</label>
                            
                            <select v-if="bulkTestForm.prefix === 'M' || bulkTestForm.prefix === 'B'" v-model="bulkTestForm.value" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                                <option value="" disabled selected>e.g. 0 or 1</option>
                                <option value="0">0 (False)</option>
                                <option value="1">1 (True)</option>
                            </select>

                            <input v-else-if="bulkTestForm.prefix === 'D' || bulkTestForm.prefix === 'W'" type="number" placeholder="e.g. 100" v-model="bulkTestForm.value" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                            
                            <template v-else-if="bulkTestForm.prefix === 'R' || bulkTestForm.prefix === 'ZR'">
                                <input type="text" placeholder="e.g. Hello" v-model="bulkTestForm.value" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                                <span v-if="bulkTestForm.value && typeof bulkTestForm.value === 'string'" style="display: block; font-size: 0.75rem; color: var(--accent-blue); margin-top: 4px;">
                                    {{ bulkTestForm.value.length }} characters = {{ Math.ceil(bulkTestForm.value.length / 2) }} registers
                                </span>
                            </template>
                            
                            <input v-else type="text" placeholder="Value..." v-model="bulkTestForm.value" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                        </div>
                        <div class="login-field" style="margin-bottom: 0;">
                            <label style="color:var(--text-primary)" title="Continuous stress testing loop duration.">
                                Duration <ion-icon name="time-outline" style="vertical-align: middle; color: var(--accent-blue);"></ion-icon>
                            </label>
                            <input list="duration-options" v-model="bulkTestForm.duration" placeholder="e.g. 1s, 500ms, 1m" style="padding: 12px; border-radius: 8px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); width: 100%;">
                            <datalist id="duration-options">
                                <option value="0s" label="Once (0s)"></option>
                                <option value="1s" label="1 Second"></option>
                                <option value="5s" label="5 Seconds"></option>
                                <option value="10s" label="10 Seconds"></option>
                            </datalist>
                        </div>
                    </div>
                    
                    <div style="display: flex; gap: 1rem; justify-content: center; margin-bottom: 1.5rem; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 1.5rem;">
                        <button class="btn btn-success" style="height: 48px; min-width: 180px; font-weight: 600;" @click="runBulkTest('read')" :disabled="bulkTestRunning || !bulkTestForm.plc_id">
                            <ion-icon name="arrow-down-outline" style="margin-right: 0.5rem;"></ion-icon> Run Read Test
                        </button>
                        <button class="btn btn-danger" style="height: 48px; min-width: 180px; font-weight: 600;" @click="runBulkTest('write')" :disabled="bulkTestRunning || !bulkTestForm.plc_id">
                            <ion-icon name="arrow-up-outline" style="margin-right: 0.5rem;"></ion-icon> Run Write Test
                        </button>
                    </div>

                    <div v-if="bulkTestStatus" :style="{ marginBottom: '1rem', fontWeight: 'bold', color: bulkTestStatus.includes('Error') ? '#e74c3c' : '#3498db' }">
                        {{ bulkTestStatus }}
                    </div>

                    <div style="height: 350px; position: relative;">
                        <v-chart :option="bulkTestChartOption" autoresize />
                    </div>
                    
                    <div v-if="bulkTestStats" style="display: flex; gap: 1rem; margin-top: 1rem; flex-wrap: wrap;">
                        <div class="stat-card" style="flex: 1; min-width: 150px; padding: 1rem; background: rgba(52, 152, 219, 0.05); border: 1px solid rgba(52, 152, 219, 0.2); box-shadow: none;">
                            <div class="stat-details" style="margin-left: 0;">
                                <h3 style="color:var(--text-primary)">Tags Tested</h3><h2 style="color:var(--accent-blue)">{{ bulkTestStats.count }}</h2>
                            </div>
                        </div>
                        <div class="stat-card" @click="showBulkHistoryModal = true" style="flex: 1; min-width: 150px; padding: 1rem; background: rgba(155, 89, 182, 0.05); border: 1px solid rgba(155, 89, 182, 0.2); box-shadow: none; cursor: pointer;" title="Click to view full iteration JSON history">
                            <div class="stat-details" style="margin-left: 0;">
                                <h3 style="color:var(--text-primary)">Test Iterations</h3><h2 style="color:#9b59b6">{{ bulkTestStats.total_records }}</h2>
                            </div>
                        </div>
                        <div class="stat-card" style="flex: 1; min-width: 150px; padding: 1rem; background: rgba(46, 204, 113, 0.05); border: 1px solid rgba(46, 204, 113, 0.2); box-shadow: none;">
                            <div class="stat-details" style="margin-left: 0;">
                                <h3 style="color:var(--text-primary)">Test Value</h3>
                                <h2 style="color:#2ecc71; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 150px;" :title="bulkTestStats.value">
                                    {{ bulkTestStats.value }}
                                </h2>
                            </div>
                        </div>
                        <div class="stat-card" style="flex: 1; min-width: 150px; padding: 1rem; background: rgba(52, 152, 219, 0.05); border: 1px solid rgba(52, 152, 219, 0.2); box-shadow: none;">
                            <div class="stat-details" style="margin-left: 0;">
                                <h3 style="color:var(--text-primary)">Read RTT</h3><h2 style="color:var(--accent-blue)">{{ bulkTestStats.read_latency_ms.toFixed(2) }}ms</h2>
                            </div>
                        </div>
                        <div class="stat-card" style="flex: 1; min-width: 150px; padding: 1rem; background: rgba(231, 76, 60, 0.05); border: 1px solid rgba(231, 76, 60, 0.2); box-shadow: none;">
                            <div class="stat-details" style="margin-left: 0;">
                                <h3 style="color:var(--text-primary)">Write RTT</h3><h2 style="color:var(--accent-red)">{{ bulkTestStats.write_latency_ms.toFixed(2) }}ms</h2>
                            </div>
                        </div>
                    </div>
                </div>


                <!-- ===================== ROBOT HEALTH DASHBOARD ===================== -->
                <div class="page-header" style="margin-top: 2rem;">
                    <div>
                        <h1>Robot Info Dashboard</h1>
                        <p class="subtitle">Information and health status of all robots</p>
                    </div>
                    <button class="btn btn-primary" @click="openAddRobot">
                        <ion-icon name="add-outline"></ion-icon> Add Robot
                    </button>
                </div>
                <div class="health-grid">
                    <div v-for="robot in robots" :key="robot.id" class="health-card" :class="getRobotStatus(robot).class">
                        <div class="health-card-header">
                            <div>
                                <h3 style="font-size: 1.1rem; margin-bottom: 4px;">
                                    <svg style="width:20px;height:20px;vertical-align:middle;margin-right:6px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <rect x="4" y="5" width="16" height="10" rx="2"/>
                                        <line x1="12" y1="15" x2="12" y2="19"/>
                                        <line x1="8" y1="19" x2="16" y2="19"/>
                                        <circle cx="9" cy="10" r="1.5" fill="currentColor" stroke="none"/>
                                        <circle cx="15" cy="10" r="1.5" fill="currentColor" stroke="none"/>
                                        <line x1="2" y1="9" x2="4" y2="10"/>
                                        <line x1="22" y1="9" x2="20" y2="10"/>
                                    </svg>
                                    {{ robot.name || 'Unnamed' }}
                                </h3>
                                <span class="plc-type">{{ robot.model ? robot.model.manufacturer + ' - ' + robot.model.name : 'Unknown Make' }}</span>
                            </div>
                            <div class="health-status">
                                <span class="dot" :class="getRobotStatus(robot).class"></span>
                                {{ getRobotStatus(robot).text }}
                            </div>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="finger-print-outline"></ion-icon>
                            <span class="text-muted">ID:</span>
                            <span class="code-badge" style="font-size: 0.75rem;">{{ robot.id?.substring(0,8) || '' }}</span>
                        </div>
                        <div class="health-detail">
                            <ion-icon name="globe-outline"></ion-icon>
                            <span class="text-muted">IP:</span>
                            <span class="code-badge">{{ robot.ip_address || '-' }}</span>
                        </div>
                        <div class="health-latency" style="border-top: 1px solid var(--border-color); padding-top: 0.8rem; margin-top: 0.5rem;">
                            <span>Stats</span>
                            <span :class="getRobotStats(robot).class">
                                <strong>{{ getRobotStats(robot).text }}</strong>
                            </span>
                        </div>
                    </div>
                    <div v-if="robots.length === 0" class="empty-state" style="grid-column:1/-1">
                        <div class="empty-icon"><ion-icon name="hardware-chip-outline"></ion-icon></div>
                        <h2>No Robots Configured</h2>
                        <p>Add robots via the Robot Management tab to see their info here.</p>
                    </div>
                </div>
            </div>

            <!-- ===================== PLCS ===================== -->
            <div class="view-content" v-if="tab === 'plcs'">
                <div class="page-header">
                    <div>
                        <h1>PLC Management</h1>
                        <p class="subtitle">Configure and manage programmable logic controllers</p>
                    </div>
                    <div class="btn-group">
                        <button class="btn btn-outline" @click="scanAllPLCPorts" :disabled="isScanningPorts">
                            <ion-icon name="scan-outline"></ion-icon> {{ isScanningPorts ? 'Scanning...' : 'Port Scan' }}
                        </button>
                        <button class="btn btn-primary" @click="openAddPlc">
                            <ion-icon name="add"></ion-icon> Add PLC
                        </button>
                    </div>
                </div>
                <div class="table-card">
                    <div class="table-toolbar">
                        <div class="search-input">
                            <ion-icon name="search"></ion-icon>
                            <input type="text" v-model="plcSearch" placeholder="Search PLCs...">
                        </div>
                        <div class="col-picker-wrap">
                            <button class="col-picker-btn" @click="plcShowCols=!plcShowCols"><ion-icon name="eye-outline"></ion-icon> Columns</button>
                            <div class="col-picker-dropdown" v-if="plcShowCols" @click.self="plcShowCols=false">
                                <label v-for="col in plcColDefs" :key="col.key"><input type="checkbox" :checked="plcVisibleCols.includes(col.key)" @change="toggleCol(plcVisibleCols,col.key)">{{ col.label }}</label>
                            </div>
                        </div>
                        <span class="text-muted">{{ filteredPlcs.length }} PLCs</span>
                    </div>
                    <div class="table-responsive">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th v-if="plcVisibleCols.includes('facility_name')">Facility Name</th>
                                    <th v-if="plcVisibleCols.includes('driver')">Driver</th>
                                    <th v-if="plcVisibleCols.includes('ip_address')">IP Address</th>
                                    <th v-if="plcVisibleCols.includes('comtype')">Com Type</th>
                                    <th v-if="plcVisibleCols.includes('rack')">Rack</th>
                                    <th v-if="plcVisibleCols.includes('slot')">Slot</th>
                                    <th v-if="plcVisibleCols.includes('port')">Port</th>
                                    <th v-if="plcVisibleCols.includes('alarm_port')">Alarm Port</th>
                                    <th v-if="plcVisibleCols.includes('maker')">Maker</th>
                                    <th>Available Ports</th>
                                    <th>Status</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-if="loading.plcs">
                                    <td :colspan="12" class="text-center py-4">Loading...</td>
                                </tr>
                                <tr v-else-if="filteredPlcs.length === 0">
                                    <td :colspan="12" class="text-center py-4 text-muted">No PLCs found.</td>
                                </tr>
                                <tr v-for="p in filteredPlcs" :key="p.id">
                                    <td v-if="plcVisibleCols.includes('facility_name')" class="font-medium">{{ p.facility_name }}</td>
                                    <td v-if="plcVisibleCols.includes('driver')"><span class="badge badge-blue">{{ p.driver || '-' }}</span></td>
                                    <td v-if="plcVisibleCols.includes('ip_address')"><span class="code-badge">{{ p.ip_address }}</span></td>
                                    <td v-if="plcVisibleCols.includes('comtype')">{{ p.comtype || '-' }}</td>
                                    <td v-if="plcVisibleCols.includes('rack')">{{ p.rack ?? '-' }}</td>
                                    <td v-if="plcVisibleCols.includes('slot')">{{ p.slot ?? '-' }}</td>
                                    <td v-if="plcVisibleCols.includes('port')"><span class="code-badge" style="display:flex;align-items:center;"><span class="dot online" style="width:8px;height:8px;margin-right:6px;"></span>{{ p.port }}</span></td>
                                    <td v-if="plcVisibleCols.includes('alarm_port')"><span class="code-badge">{{ p.alarm_port || '-' }}</span></td>
                                    <td v-if="plcVisibleCols.includes('maker')"><span class="badge badge-purple">{{ p.maker || '-' }}</span></td>
                                    <td>
                                        <div class="port-cell">
                                            <div class="port-chips" v-if="p.openPorts && p.openPorts.length > 0">
                                                <span class="port-btn active-port" :title="'Active port: ' + p.port">
                                                    <span class="port-active-dot"></span>{{ p.port }}
                                                </span>
                                                <span 
                                                    v-for="port in (p.openPorts || []).filter(pt => pt !== p.port)" 
                                                    :key="port"
                                                    class="port-btn" 
                                                    :class="isPortInUse(port) ? 'used-port' : 'free-port'"
                                                    @click="!isPortInUse(port) && quickUpdatePort(p, port)"
                                                    :title="isPortInUse(port) ? 'Port in use by another PLC' : 'Click to use this port'"
                                                >{{ port }}</span>
                                            </div>
                                            <span class="text-muted text-sm" v-else-if="p.openPorts && p.openPorts.length === 0">None found</span>
                                            <span class="text-muted text-sm scanning-text" v-else-if="p.isScanning">Scanning...</span>
                                            <span class="text-muted text-sm" v-else>-</span>
                                            <button class="port-refresh-btn" :class="{'spin': p.isScanning}" @click="scanSinglePLC(p)" title="Scan ports">
                                                <ion-icon name="refresh-outline"></ion-icon>
                                            </button>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="badge" :class="health.find(h => h.id === p.id)?.status === 'online' ? 'badge-green' : 'badge-red'">
                                            {{ health.find(h => h.id === p.id)?.status || 'unknown' }}
                                        </span>
                                    </td>
                                    <td>
                                        <div class="action-buttons">
                                            <button class="icon-btn edit" @click="openEditPlc(p)"><ion-icon name="create-outline"></ion-icon></button>
                                            <button class="icon-btn delete" @click="deletePlc(p)"><ion-icon name="trash-outline"></ion-icon></button>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- ===================== ROBOTS ===================== -->
            <div class="view-content" v-if="tab === 'robots'">
                <div class="page-header">
                    <div>
                        <h1>Robot Management</h1>
                        <p class="subtitle">Configure robots and their associated models</p>
                    </div>
                    <div class="btn-group">
                        <button class="btn btn-primary" @click="openAddRobot">
                            <ion-icon name="add"></ion-icon> Add Robot
                        </button>
                        <button class="btn btn-outline" @click="openAddModel">
                            <ion-icon name="layers-outline"></ion-icon> Add Model
                        </button>
                    </div>
                </div>
                <div class="table-card">
                    <div class="table-toolbar">
                        <div class="search-input">
                            <ion-icon name="search"></ion-icon>
                            <input type="text" v-model="robotSearch" placeholder="Search robots...">
                        </div>
                        <div class="col-picker-wrap">
                            <button class="col-picker-btn" @click="robotShowCols=!robotShowCols"><ion-icon name="eye-outline"></ion-icon> Columns</button>
                            <div class="col-picker-dropdown" v-if="robotShowCols" @click.self="robotShowCols=false">
                                <label v-for="col in robotColDefs" :key="col.key"><input type="checkbox" :checked="robotVisibleCols.includes(col.key)" @change="toggleCol(robotVisibleCols,col.key)">{{ col.label }}</label>
                            </div>
                        </div>
                        <span class="text-muted">{{ filteredRobots.length }} robots</span>
                    </div>
                    <div class="table-responsive">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <template v-for="col in robotColDefs" :key="col.key">
                                        <th v-if="robotVisibleCols.includes(col.key)">{{ col.label }}</th>
                                    </template>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-if="loading.robots">
                                    <td :colspan="robotVisibleCols.length + 1" class="text-center py-4">Loading...</td>
                                </tr>
                                <tr v-else-if="filteredRobots.length === 0">
                                    <td :colspan="robotVisibleCols.length + 1" class="text-center py-4 text-muted">No robots found.</td>
                                </tr>
                                <tr v-for="r in filteredRobots" :key="r.id">
                                    <template v-for="col in robotColDefs" :key="col.key">
                                        <td v-if="robotVisibleCols.includes(col.key)">
                                            <template v-if="col.key === 'id'"><span class="code-badge">{{ r.id?.substring(0,8) || '' }}</span></template>
                                            <template v-else-if="col.key === 'plc_id'"><span class="code-badge">{{ r.plc_id || '-' }}</span></template>
                                            <template v-else-if="col.key === 'ip_address'"><span class="code-badge">{{ r.ip_address || '-' }}</span></template>
                                            <template v-else-if="col.key === 'model_id'">{{ r.model_id || '-' }}</template>
                                            <template v-else>{{ r[col.key] ?? '-' }}</template>
                                        </td>
                                    </template>
                                    <td>
                                        <div class="action-buttons">
                                            <button class="icon-btn edit" @click="openEditRobot(r)"><ion-icon name="create-outline"></ion-icon></button>
                                            <button class="icon-btn delete" @click="deleteRobot(r)"><ion-icon name="trash-outline"></ion-icon></button>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- ===================== TAGS ===================== -->
            <div class="view-content" v-if="tab === 'tags'">
                <div class="page-header">
                    <div>
                        <h1>Tag Management</h1>
                        <p class="subtitle">Manage PLC tags with bulk import/export</p>
                    </div>
                    <div class="btn-group">
                        <button class="btn btn-primary" @click="openAddTag">
                            <ion-icon name="add"></ion-icon> Add Tag
                        </button>
                        <button class="btn btn-success" @click="openImportCSV">
                            <ion-icon name="cloud-upload-outline"></ion-icon> Import CSV
                        </button>
                        <a :href="exportURL" class="btn btn-outline">
                            <ion-icon name="cloud-download-outline"></ion-icon> Export
                        </a>
                        <button class="btn btn-danger" @click="deleteAllTags" :disabled="tagTotal === 0">
                            <ion-icon name="trash-outline"></ion-icon> Delete All
                        </button>
                    </div>
                </div>
                <div class="table-card">
                    <div class="table-toolbar">
                        <div class="search-input">
                            <ion-icon name="search"></ion-icon>
                            <input type="text" v-model="tagSearch" @input="debounceSearchTags" placeholder="Search tag address, name, comment...">
                        </div>
                        <div class="col-picker-wrap">
                            <button class="col-picker-btn" @click="tagShowCols=!tagShowCols"><ion-icon name="eye-outline"></ion-icon> Columns</button>
                            <div class="col-picker-dropdown" v-if="tagShowCols" @click.self="tagShowCols=false">
                                <label v-for="col in tagColDefs" :key="col.key"><input type="checkbox" :checked="tagVisibleCols.includes(col.key)" @change="toggleCol(tagVisibleCols,col.key)">{{ col.label }}</label>
                            </div>
                        </div>
                        <span class="text-muted">{{ tagTotal }} total tags</span>
                    </div>
                    <div class="table-responsive">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th v-if="tagVisibleCols.includes('id')">ID</th>
                                    <th v-if="tagVisibleCols.includes('tag_address')" @click="toggleTagSort('tag_address')" style="cursor:pointer" class="hover:bg-gray-700/50 transition-colors">
                                        Tag Address
                                        <ion-icon v-if="tagSortBy === 'tag_address'" :name="tagSortOrder === 'asc' ? 'chevron-up-outline' : 'chevron-down-outline'" class="ml-1"></ion-icon>
                                        <ion-icon v-else name="swap-vertical-outline" class="ml-1 opacity-25"></ion-icon>
                                    </th>
                                    <th v-if="tagVisibleCols.includes('plc_ip')">PLC IP</th>
                                    <th v-if="tagVisibleCols.includes('tag_name')" @click="toggleTagSort('tag_name')" style="cursor:pointer" class="hover:bg-gray-700/50 transition-colors">
                                        Tag Name
                                        <ion-icon v-if="tagSortBy === 'tag_name'" :name="tagSortOrder === 'asc' ? 'chevron-up-outline' : 'chevron-down-outline'" class="ml-1"></ion-icon>
                                        <ion-icon v-else name="swap-vertical-outline" class="ml-1 opacity-25"></ion-icon>
                                    </th>
                                    <th v-if="tagVisibleCols.includes('fac_name')">Facility</th>
                                    <th v-if="tagVisibleCols.includes('robot_id')">Robot ID</th>
                                    <th v-if="tagVisibleCols.includes('plc_id')">PLC ID</th>
                                    <th v-if="tagVisibleCols.includes('data_type')">Data Type</th>
                                    <th v-if="tagVisibleCols.includes('comment')">Comment</th>
                                    <th v-if="tagVisibleCols.includes('action')">Action</th>
                                    <th v-if="tagVisibleCols.includes('screen')">Screen</th>
                                    <th v-if="tagVisibleCols.includes('svg_element')">SVG</th>
                                    <th v-if="tagVisibleCols.includes('true_condition_color')">True Color</th>
                                    <th v-if="tagVisibleCols.includes('false_condition_color')">False Color</th>
                                    <th v-if="tagVisibleCols.includes('blinking')">Blinking</th>
                                    <th v-if="tagVisibleCols.includes('refresh_rate')">Refresh Rate</th>
                                    <th v-if="tagVisibleCols.includes('value')">Value</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-if="loading.tags">
                                    <td :colspan="18" class="text-center py-4">Loading...</td>
                                </tr>
                                <tr v-else-if="tags.length === 0">
                                    <td :colspan="18" class="text-center py-4 text-muted">No tags found.</td>
                                </tr>
                                <tr v-for="t in tags" :key="t.id">
                                    <td v-if="tagVisibleCols.includes('id')">{{ t.id }}</td>
                                    <td v-if="tagVisibleCols.includes('tag_address')"><span class="code-badge">{{ t.tag_address }}</span></td>
                                    <td v-if="tagVisibleCols.includes('plc_ip')">{{ t.plc_ip || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('tag_name')">{{ t.tag_name || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('fac_name')">{{ t.fac_name || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('robot_id')">{{ t.robot_id || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('plc_id')">{{ t.plc_id || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('data_type')"><span class="badge badge-purple">{{ t.data_type || 'bit' }}</span></td>
                                    <td v-if="tagVisibleCols.includes('comment')">{{ t.comment || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('action')">{{ t.action || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('screen')">{{ t.screen || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('svg_element')">{{ t.svg_element ? 'Yes' : 'No' }}</td>
                                    <td v-if="tagVisibleCols.includes('true_condition_color')">{{ t.true_condition_color || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('false_condition_color')">{{ t.false_condition_color || '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('blinking')">{{ t.blinking ? 'Yes' : 'No' }}</td>
                                    <td v-if="tagVisibleCols.includes('refresh_rate')">{{ t.refresh_rate ?? '-' }}</td>
                                    <td v-if="tagVisibleCols.includes('value')">
                                        <span v-if="t.value != null" class="badge clickable-badge" :class="t.value ? 'badge-green' : 'badge-orange'" @click="toggleRegisterById(t.id, t.value, t.tag_address)">
                                            {{ t.value }}
                                        </span>
                                        <span v-else class="text-muted">-</span>
                                    </td>
                                    <td>
                                        <div class="action-buttons">
                                            <button class="icon-btn edit" @click="openEditTag(t)"><ion-icon name="create-outline"></ion-icon></button>
                                            <button class="icon-btn delete" @click="deleteTag(t)"><ion-icon name="trash-outline"></ion-icon></button>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="pagination" v-if="tagTotal > tagPerPage">
                        <button class="page-btn" :disabled="tagPage <= 1" @click="tagPage--; fetchTags()">
                            <ion-icon name="chevron-back"></ion-icon>
                        </button>
                        <span class="page-info">Page {{ tagPage }} of {{ tagMaxPage }}</span>
                        <button class="page-btn" :disabled="tagPage >= tagMaxPage" @click="tagPage++; fetchTags()">
                            <ion-icon name="chevron-forward"></ion-icon>
                        </button>
                    </div>
                </div>
            </div>

            <!-- ===================== ADMIN ===================== -->
            <div class="view-content" v-if="tab === 'admin'">
                <div class="page-header">
                    <div>
                        <h1>Admin Control</h1>
                        <p class="subtitle">User management and role-based access control</p>
                    </div>
                    <div class="btn-group">
                        <button class="btn btn-primary btn-sm" @click="fetchUsers(); fetchRoles()"><ion-icon name="refresh-outline"></ion-icon> Refresh</button>
                        <button class="btn btn-success btn-sm" @click="openAuth('login')"><ion-icon name="person-add-outline"></ion-icon> Add User</button>
                    </div>
                </div>
                <div class="admin-tabs" style="display:flex;gap:0.5rem;margin-bottom:1.5rem;">
                    <button class="btn" :class="adminSubTab === 'users' ? 'btn-primary' : 'btn-outline'" @click="adminSubTab='users'">
                        <ion-icon name="people-outline"></ion-icon> Users
                    </button>
                    <button class="btn" :class="adminSubTab === 'permissions' ? 'btn-primary' : 'btn-outline'" @click="adminSubTab='permissions'; fetchModules(); fetchAllRolePerms()" v-if="isSuperadmin">
                        <ion-icon name="shield-outline"></ion-icon> Permissions
                    </button>
                </div>

                <!-- Users Tab -->
                <template v-if="adminSubTab === 'users'">
                <div class="section-label">
                    <ion-icon name="people-outline"></ion-icon>
                    <span>Users & Roles</span>
                </div>
                <div class="table-card">
                    <div class="table-responsive">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Email</th>
                                    <th>Role</th>
                                    <th>Status</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="u in users" :key="u.id">
                                    <td><span class="font-medium">{{ u.name }}</span></td>
                                    <td><span class="text-muted">{{ u.email }}</span></td>
                                    <td>
                                        <select class="form-select" :value="u.role_id" @change="updateUserRole(u.id, $event.target.value)" v-if="isSuperadmin">
                                            <option v-for="r in roles" :key="r.id" :value="r.id">{{ r.label }}</option>
                                        </select>
                                        <span v-else class="role-badge" :class="u.role">{{ u.role_data?.label || u.role }}</span>
                                    </td>
                                    <td>
                                        <span class="status-dot-sm" :class="u.active ? 'active' : 'inactive'"></span>
                                        {{ u.active ? 'Active' : 'Inactive' }}
                                    </td>
                                    <td>
                                        <button class="icon-btn" :title="u.active ? 'Deactivate' : 'Activate'" @click="toggleUserActive(u.id, u.active)" v-if="isSuperadmin">
                                            <ion-icon :name="u.active ? 'pause-circle-outline' : 'checkmark-circle-outline'"></ion-icon>
                                        </button>
                                        <button class="icon-btn delete" title="Delete" @click="deleteUserById(u.id)" v-if="isSuperadmin">
                                            <ion-icon name="trash-outline"></ion-icon>
                                        </button>
                                    </td>
                                </tr>
                                <tr v-if="users.length === 0">
                                    <td colspan="5" class="text-center py-4">
                                        <div class="text-muted">{{ adminLoading ? 'Loading...' : 'No users found' }}</div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <div class="section-label" style="margin-top:1.5rem">
                    <ion-icon name="shield-outline"></ion-icon>
                    <span>Role Hierarchy</span>
                </div>
                <div class="stats-grid" style="grid-template-columns:repeat(auto-fit,minmax(280px,1fr))">
                    <div v-for="r in roles" :key="r.id" class="stat-card" style="flex-direction:column;align-items:flex-start">
                        <div class="stat-card-header" style="display:flex;align-items:center;gap:12px;width:100%">
                            <div class="stat-icon" :class="r.level === 1 ? 'primary-bg' : r.level <= 5 ? 'warning-bg' : 'info-bg'">
                                <ion-icon :name="r.level === 1 ? 'flash-outline' : r.level <= 5 ? 'settings-outline' : 'person-outline'"></ion-icon>
                            </div>
                            <div>
                                <h3 style="font-size:1.1rem">{{ r.label }}</h3>
                                <div class="text-muted" style="font-size:0.85rem">{{ r.description }}</div>
                            </div>
                        </div>
                        <div class="role-level-bar" style="margin-top:12px;width:100%">
                            <div class="level-track">
                                <div class="level-fill" :style="{ width: (100 - (r.level * 10)) + '%', background: r.level === 1 ? 'var(--accent-blue)' : r.level <= 5 ? 'var(--accent-orange)' : 'var(--accent-cyan)' }"></div>
                            </div>
                            <div class="text-muted" style="font-size:0.75rem;margin-top:4px">Level {{ r.level }} — {{ r.level === 1 ? 'Full access' : r.level <= 5 ? 'Limited admin' : 'Basic access' }}</div>
                        </div>
                    </div>
                </div>
                </template>

                <!-- Permissions Tab -->
                <template v-if="adminSubTab === 'permissions' && isSuperadmin">
                <div class="section-label">
                    <ion-icon name="shield-outline"></ion-icon>
                    <span>Module Permissions by Role</span>
                </div>
                <div class="table-card">
                    <div class="table-responsive">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th>Module</th>
                                    <th v-for="r in roles" :key="r.id" style="text-align:center">
                                        {{ r.label }}
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="m in modules" :key="m.id">
                                    <td>
                                        <ion-icon :name="m.icon" style="margin-right:0.5rem;vertical-align:middle"></ion-icon>
                                        {{ m.label }}
                                    </td>
                                    <td v-for="r in roles" :key="r.id" style="text-align:center">
                                        <label class="toggle-switch">
                                            <input type="checkbox" 
                                                :checked="rolePermsByRole(r.id, m.id)"
                                                @change="toggleModulePerm(r.id, m.id, $event.target.checked)">
                                            <span class="toggle-slider"></span>
                                        </label>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                </template>
            </div>
        </main>
        </template>

        <!-- ===================== MODALS ===================== -->

        <!-- Bulk History Modal -->
        <div class="modal-overlay" v-if="showBulkHistoryModal" @click.self="showBulkHistoryModal = false">
            <div class="modal wide" style="max-width: 900px;">
                <div class="modal-header">
                    <h2>Stress Test History JSON</h2>
                    <button class="modal-close" @click="showBulkHistoryModal = false">✕</button>
                </div>
                <div class="modal-body" style="max-height: 60vh; overflow-y: auto; background: var(--bg-card); padding: 1rem; border-radius: 8px;">
                    <pre style="margin: 0; color: var(--text-primary); font-family: monospace; font-size: 0.85rem; white-space: pre-wrap;">{{ JSON.stringify(bulkTestStats?.history || [], null, 4) }}</pre>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showBulkHistoryModal = false">Close</button>
                </div>
            </div>
        </div>

        <!-- PLC Modal -->
        <div class="modal-overlay" v-if="showPlcModal" @click.self="showPlcModal = false">
            <div class="modal wide" style="background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 16px; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.2);">
                <div class="modal-header" style="border-bottom: 1px solid var(--border-color); padding-bottom: 1rem;">
                    <h2 style="display: flex; align-items: center; gap: 0.5rem; font-size: 1.5rem; font-weight: 700; background: linear-gradient(90deg, var(--accent-blue) 0%, var(--accent-orange) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">
                        <ion-icon name="hardware-chip-outline" style="color: var(--accent-blue);"></ion-icon>
                        {{ editingPlc ? 'Configure PLC' : 'Add New PLC' }}
                    </h2>
                    <button class="modal-close" @click="showPlcModal = false" style="background: var(--bg-hover); border-radius: 50%; width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; transition: all 0.3s ease;">
                        <ion-icon name="close"></ion-icon>
                    </button>
                </div>
                
                <div class="modal-body" style="padding: 1.5rem 0;">
                    <!-- Section: Identity -->
                    <div style="background: var(--bg-main); padding: 1.25rem; border-radius: 12px; margin-bottom: 1.5rem; border: 1px solid var(--border-color);">
                        <h4 style="margin: 0 0 1rem 0; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 1px; color: var(--text-secondary); display: flex; align-items: center; gap: 0.5rem;">
                            <ion-icon name="id-card-outline"></ion-icon> Identity & Specs
                        </h4>
                        
                        <div class="form-row">
                            <div class="form-group" style="flex: 2;">
                                <label style="font-weight: 500; color: var(--text-primary);">Facility Name *</label>
                                <input class="form-control" v-model="plcForm.name" placeholder="e.g. Production Line 1" style="background: var(--bg-card); transition: all 0.3s;">
                            </div>
                            <div class="form-group" style="flex: 1;">
                                <label style="font-weight: 500; color: var(--text-primary);">Maker / Vendor</label>
                                <select class="form-control" v-model="plcForm.make" @change="onMakeChange" style="background: var(--bg-card);">
                                    <option value="">Select vendor...</option>
                                    <option v-for="m in famousMakes" :key="m" :value="m">{{ m }}</option>
                                </select>
                            </div>
                        </div>

                        <div class="form-row">
                            <div class="form-group">
                                <label style="font-weight: 500; color: var(--text-primary);">Driver / Series</label>
                                <select class="form-control" v-model="plcForm.series" style="background: var(--bg-card);">
                                    <option value="">Select series...</option>
                                    <option v-for="s in seriesOptions" :key="s" :value="s">{{ s }}</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label style="font-weight: 500; color: var(--text-primary);">Com Type / Protocol</label>
                                <div class="input-with-detect" style="display: flex; gap: 0.5rem;">
                                    <input class="form-control" v-model="plcForm.protocol" placeholder="Binary, ASCII, etc" style="flex: 1; background: var(--bg-card);">
                                    <button class="btn btn-sm btn-outline detect-btn" @click="autoDetectPlc" :title="'Auto-detect protocol on ' + plcForm.ip_address" style="background: rgba(56, 189, 248, 0.1); color: var(--accent-blue); border-color: rgba(56, 189, 248, 0.3); border-radius: 8px;">
                                        <ion-icon name="flash" style="animation: pulse 2s infinite;"></ion-icon> Detect
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Section: Network -->
                    <div style="background: var(--bg-main); padding: 1.25rem; border-radius: 12px; margin-bottom: 1.5rem; border: 1px solid var(--border-color);">
                        <h4 style="margin: 0 0 1rem 0; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 1px; color: var(--text-secondary); display: flex; align-items: center; gap: 0.5rem;">
                            <ion-icon name="globe-outline"></ion-icon> Network Configuration
                        </h4>

                        <div class="form-group">
                            <label style="font-weight: 500; color: var(--text-primary);">IP Address *</label>
                            <div class="input-with-detect" style="display: flex; gap: 0.5rem;">
                                <input class="form-control" v-model="plcForm.ip_address" placeholder="192.168.1.100" style="flex: 1; background: var(--bg-card); font-family: monospace; font-size: 1.1rem;">
                                <button class="btn btn-sm btn-outline detect-btn" @click="autoDetectPlc" style="background: var(--bg-hover); border-color: var(--border-color); border-radius: 8px;">
                                    <ion-icon name="search-outline"></ion-icon> Auto-Detect
                                </button>
                            </div>
                        </div>

                        <div class="form-group" style="margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px dashed var(--border-color);">
                            <div class="port-scan-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                                <label style="margin: 0; font-weight: 500; color: var(--text-primary); display: flex; align-items: center; gap: 0.5rem;">
                                    <ion-icon name="radio-outline" style="color: var(--accent-blue);"></ion-icon> Port Scanning
                                </label>
                                <button class="btn btn-sm" style="background: linear-gradient(135deg, var(--accent-blue) 0%, var(--accent-orange) 100%); color: white; border: none; border-radius: 20px; padding: 0.4rem 1rem; font-weight: 600; box-shadow: 0 4px 15px rgba(251, 146, 60, 0.3); transition: all 0.3s;" @click="scanOpenPorts" :disabled="scanningPorts || !plcForm.ip_address">
                                    <ion-icon :name="scanningPorts ? 'sync-outline' : 'radar-outline'" :style="scanningPorts ? 'animation: spin 1s linear infinite;' : ''"></ion-icon>
                                    {{ scanningPorts ? 'Scanning...' : 'Scan Ports' }}
                                </button>
                            </div>
                            
                            <div v-if="scanningPorts" class="scan-progress" style="display: flex; flex-direction: column; align-items: center; padding: 2rem; background: var(--bg-hover); border-radius: 8px;">
                                <ion-icon name="aperture-outline" style="font-size: 2.5rem; color: var(--accent-blue); animation: spin 2s linear infinite; margin-bottom: 1rem;"></ion-icon>
                                <span style="color: var(--text-secondary);">Pinging ports for {{ plcForm.make || 'all makes' }}...</span>
                            </div>
                            <div v-else-if="scannedPorts.length > 0" class="ports-found" style="background: rgba(16, 185, 129, 0.05); padding: 1rem; border-radius: 8px; border: 1px solid rgba(16, 185, 129, 0.2);">
                                <div class="port-section" style="margin-bottom: 1rem;">
                                    <label class="port-label" style="color: var(--accent-blue); font-size: 0.85rem; font-weight: 600; margin-bottom: 0.5rem; display: block;">Available Open Ports</label>
                                    <div class="port-chips" style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
                                        <span v-for="port in scannedPorts" :key="port"
                                            class="port-chip"
                                            :style="`padding: 0.4rem 0.8rem; border-radius: 20px; font-size: 0.9rem; font-weight: 600; transition: all 0.2s; border: 1px solid ${plcForm.read_port === port ? 'var(--accent-blue)' : 'var(--border-color)'}; background: ${plcForm.read_port === port ? 'rgba(56,189,248,0.15)' : 'var(--bg-card)'}; color: ${plcForm.read_port === port ? 'var(--accent-blue)' : 'var(--text-primary)'};`">
                                            <ion-icon name="radio-button-on" style="font-size: 0.7rem; margin-right: 0.2rem;"></ion-icon> {{ port }}
                                            <small v-if="plcForm.read_port === port" style="margin-left: 0.3rem; opacity: 0.8;">[Active]</small>
                                        </span>
                                    </div>
                                </div>
                                <div class="port-assign-row" style="display: flex; gap: 1rem;">
                                    <div class="form-group" style="flex: 1; margin: 0;">
                                        <label style="font-size: 0.8rem; color: var(--text-secondary);">Port</label>
                                        <select class="form-control" v-model.number="plcForm.read_port" style="background: var(--bg-card);">
                                            <option value="0">Select port...</option>
                                            <option v-for="port in readPortOptions" :key="'r'+port" :value="port">{{ port }}</option>
                                        </select>
                                    </div>
                                </div>
                            </div>
                            <div v-else class="text-muted" style="font-size: 0.85rem; padding: 1rem; background: var(--bg-hover); border-radius: 8px; display: flex; align-items: flex-start; gap: 0.5rem; color: var(--text-secondary);">
                                <ion-icon name="information-circle-outline" style="font-size: 1.2rem; color: var(--accent-blue); flex-shrink: 0;"></ion-icon>
                                <div>
                                    Enter IP Address and click <strong>Scan Ports</strong> to discover available ports.<br>
                                    <span style="opacity: 0.7; font-size: 0.75rem;">Default Mitsubishi ranges: 1024-1040, 5000-5020</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="modal-footer" style="border-top: 1px solid var(--border-color); padding-top: 1.5rem; display: flex; justify-content: space-between; align-items: center;">
                    <div class="uuid-display" style="display: flex; align-items: center; gap: 0.5rem; opacity: 0.8; cursor: help;" title="PLC UUID (auto-generated)">
                        <ion-icon name="finger-print-outline" style="color: var(--text-secondary);"></ion-icon>
                        <span style="font-family: monospace; font-size: 0.8rem; color: var(--text-secondary);">{{ plcForm.id ? plcForm.id.substring(0,8) + '...' : 'generating...' }}</span>
                    </div>
                    <div style="display: flex; gap: 1rem;">
                        <button class="btn btn-outline" @click="showPlcModal = false" style="border-radius: 8px; padding: 0.6rem 1.5rem;">Cancel</button>
                        <button class="btn btn-primary" @click="savePlc" style="border-radius: 8px; padding: 0.6rem 2rem; background: linear-gradient(90deg, var(--accent-blue) 0%, var(--accent-orange) 100%); border: none; font-weight: 600; box-shadow: 0 10px 20px -10px rgba(251, 146, 60, 0.8); transition: transform 0.2s; display: flex; align-items: center; gap: 0.5rem; color: white;">
                            <ion-icon :name="editingPlc ? 'save-outline' : 'add-circle-outline'"></ion-icon>
                            {{ editingPlc ? 'Update PLC' : 'Create PLC' }}
                        </button>
                    </div>
                </div>
        </div>
        </div>

        <!-- Robot Modal -->
        <div class="modal-overlay" v-if="showRobotModal" @click.self="showRobotModal = false">
            <div class="modal">
                <div class="modal-header">
                    <h2>{{ editingRobot ? 'Edit Robot' : 'Add Robot' }}</h2>
                    <button class="modal-close" @click="showRobotModal = false"><ion-icon name="close"></ion-icon></button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label>Robot Name</label>
                        <input class="form-control" v-model="robotForm.name" placeholder="e.g. Welding Arm 1">
                    </div>
                    <div class="form-row">
                        <div class="form-group" style="flex:2">
                            <label>PLC</label>
                            <select class="form-control" v-model="robotForm.plc_id">
                                <option value="">Select PLC...</option>
                                <option v-for="p in plcOptions" :key="p.id" :value="p.id">
                                    {{ p.name || p.ip_address }} ({{ p.ip_address }})
                                </option>
                            </select>
                        </div>
                        <div class="form-group" style="flex:1">
                            <label>Robot IP Address</label>
                            <input class="form-control" v-model="robotForm.ip_address" placeholder="192.168.1.100">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>Robot Model</label>
                            <select class="form-control" v-model="robotForm.model_id">
                                <option value="">Select model...</option>
                                <option v-for="m in robotModels" :key="m.id" :value="m.id">
                                    {{ m.manufacturer }} - {{ m.name }}
                                </option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Robot ID (auto)</label>
                            <input class="form-control" :value="editingRobot?.id || 'generating...'" readonly>
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showRobotModal = false">Cancel</button>
                    <button class="btn btn-primary" @click="saveRobot">{{ editingRobot ? 'Update' : 'Create' }}</button>
                </div>
            </div>
        </div>

        <!-- Model Modal -->
        <div class="modal-overlay" v-if="showModelModal" @click.self="showModelModal = false">
            <div class="modal">
                <div class="modal-header">
                    <h2>Add Robot Model</h2>
                    <button class="modal-close" @click="showModelModal = false"><ion-icon name="close"></ion-icon></button>
                </div>
                <div class="modal-body">
                    <div class="form-row">
                        <div class="form-group">
                            <label>Manufacturer</label>
                            <input class="form-control" v-model="modelForm.manufacturer" placeholder="e.g. Fanuc, ABB, KUKA">
                        </div>
                        <div class="form-group">
                            <label>Model Name</label>
                            <input class="form-control" v-model="modelForm.name" placeholder="e.g. R-2000iC">
                        </div>
                    </div>
                    <div class="table-card" v-if="robotModels.length">
                        <div class="table-responsive">
                            <table class="data-table">
                                <thead><tr><th>Manufacturer</th><th>Model</th><th>Actions</th></tr></thead>
                                <tbody>
                                    <tr v-for="m in robotModels" :key="m.id">
                                        <td>{{ m.manufacturer }}</td>
                                        <td>{{ m.name }}</td>
                                        <td><button class="icon-btn delete" @click="deleteModel(m)"><ion-icon name="trash-outline"></ion-icon></button></td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showModelModal = false">Close</button>
                    <button class="btn btn-success" @click="saveModel">Add Model</button>
                </div>
            </div>
        </div>

        <!-- Tag Modal -->
        <div class="modal-overlay" v-if="showTagModal" @click.self="showTagModal = false">
            <div class="modal wide">
                <div class="modal-header">
                    <h2>{{ editingTag ? 'Edit Tag' : 'Add Tag' }}</h2>
                    <button class="modal-close" @click="showTagModal = false"><ion-icon name="close"></ion-icon></button>
                </div>
                <div class="modal-body">
                    <div class="form-row">
                        <div class="form-group">
                            <label>Tag Address *</label>
                            <input class="form-control" v-model="tagForm.tag_address" placeholder="e.g. D100">
                        </div>
                        <div class="form-group">
                            <label>Data Type</label>
                            <select class="form-control" v-model="tagForm.data_type">
                                <option value="">Select type...</option>
                                <option v-for="dt in mitsubishiDataTypes" :key="dt" :value="dt">{{ dt.toUpperCase() }}</option>
                            </select>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>Tag Name</label>
                            <input class="form-control" v-model="tagForm.tag_name" placeholder="e.g. MotorSpeed">
                        </div>
                        <div class="form-group">
                            <label>Facility Name</label>
                            <input class="form-control" v-model="tagForm.fac_name" placeholder="e.g. Production Line 1">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group" style="flex:2">
                            <label>Comment</label>
                            <input class="form-control" v-model="tagForm.comment" placeholder="Optional description">
                        </div>
                        <div class="form-group" style="flex:1">
                            <label>Action</label>
                            <select class="form-control" v-model="tagForm.action">
                                <option value="">None</option>
                                <option value="read">Read</option>
                                <option value="write">Write</option>
                                <option value="read_write">Read / Write</option>
                            </select>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>SVG Element</label>
                            <select class="form-control" v-model="tagForm.svg_element">
                                <option :value="false">False</option>
                                <option :value="true">True</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Blinking</label>
                            <select class="form-control" v-model="tagForm.blinking">
                                <option :value="false">False</option>
                                <option :value="true">True</option>
                            </select>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>True Condition Color</label>
                            <div class="color-picker-row">
                                <input class="form-control color-hex-input" v-model="tagForm.true_condition_color" placeholder="#00FF00" maxlength="7">
                                <input class="color-picker-input" type="color" v-model="tagForm.true_condition_color">
                                <span class="color-swatch" :style="{ background: tagForm.true_condition_color || '#fff' }"></span>
                            </div>
                        </div>
                        <div class="form-group">
                            <label>False Condition Color</label>
                            <div class="color-picker-row">
                                <input class="form-control color-hex-input" v-model="tagForm.false_condition_color" placeholder="#FF0000" maxlength="7">
                                <input class="color-picker-input" type="color" v-model="tagForm.false_condition_color">
                                <span class="color-swatch" :style="{ background: tagForm.false_condition_color || '#fff' }"></span>
                            </div>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>Screen</label>
                            <input class="form-control" v-model="tagForm.screen" placeholder="e.g. main">
                        </div>
                        <div class="form-group">
                            <label>Refresh Rate (ms)</label>
                            <input class="form-control" type="number" v-model.number="tagForm.refresh_rate" placeholder="0">
                        </div>
                        <div class="form-group">
                            <label>PLC IP</label>
                            <input class="form-control" v-model="tagForm.plc_ip" placeholder="Auto-filled from robot">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>Robot</label>
                            <select class="form-control" v-model="selectedRobotId" @change="onRobotChange">
                                <option value="">Select robot...</option>
                                <option v-for="r in robots" :key="r.id" :value="r.id">
                                    {{ r.name || 'Unnamed' }} {{ r.plc?.ip_address ? '(' + r.plc.ip_address + ')' : '' }}
                                </option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>PLC ID</label>
                            <input class="form-control" v-model="tagForm.plc_id" placeholder="Auto-filled from robot" readonly>
                        </div>
                    </div>
                    <div class="form-group" style="display:none">
                        <label>Robot ID</label>
                        <input class="form-control" v-model="tagForm.robot_id" placeholder="Auto-filled">
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showTagModal = false">Cancel</button>
                    <button class="btn btn-primary" @click="saveTag">{{ editingTag ? 'Update' : 'Create' }}</button>
                </div>
            </div>
        </div>

        <!-- Import CSV Modal -->
        <div class="modal-overlay" v-if="showImportModal" @click.self="showImportModal = false">
            <div class="modal wide">
                <div class="modal-header">
                    <h2>Import Tags from CSV</h2>
                    <button class="modal-close" @click="showImportModal = false"><ion-icon name="close"></ion-icon></button>
                </div>
                <div class="modal-body">
                    <div class="file-upload" :class="{ 'has-file': importFile }" @click="csvInput?.click()">
                        <ion-icon :name="importFile ? 'checkmark-circle-outline' : 'cloud-upload-outline'"></ion-icon>
                        <p v-if="!importFile">Click to select CSV file or drag here</p>
                        <p v-else class="file-name">{{ importFile.name }}</p>
                        <p v-if="!importFile" style="margin-top:4px;font-size:0.8rem;opacity:0.6">
                            Required columns: tag_address. Optional: data_type, plc_ip, tag_name, comment, action, screen, refresh_rate
                        </p>
                        <input type="file" ref="csvInput" accept=".csv" style="display: none" @change="onFileSelect">
                    </div>

                    <div v-if="importPreview.length" style="margin-top:1.5rem">
                        <h3 style="margin-bottom:1rem;font-size:1rem;font-weight:600">
                            Preview ({{ importPreview.length }} rows)
                        </h3>
                        <div class="table-card">
                            <div class="table-responsive" style="max-height:400px">
                                <table class="data-table">
                                    <thead>
                                        <tr>
                                            <th>#</th>
                                            <th>Tag Address</th>
                                            <th>Data Type</th>
                                            <th>PLC IP</th>
                                            <th>Tag Name</th>
                                            <th>Comment</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr v-for="(row, i) in importPreview" :key="i">
                                            <td>{{ i+1 }}</td>
                                            <td><span class="code-badge">{{ row.tag_address }}</span></td>
                                            <td>{{ row.data_type || '-' }}</td>
                                            <td>{{ row.plc_ip || '-' }}</td>
                                            <td>{{ row.tag_name || '-' }}</td>
                                            <td>{{ row.comment || '-' }}</td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showImportModal = false">Cancel</button>
                    <button class="btn btn-success" :disabled="!importFile" @click="confirmImport">
                        <ion-icon name="cloud-upload-outline"></ion-icon>
                        Import {{ importPreview.length }} Tags
                    </button>
                </div>
            </div>
            </div>

        <!-- Auth Modal -->
        <div class="modal-overlay" v-if="showAuthModal" @click.self="showAuthModal = false">
            <div class="modal" style="max-width:420px">
                <div class="modal-header">
                    <h2>{{ isAdmin ? 'Add User' : 'Sign In' }}</h2>
                    <button class="modal-close" @click="showAuthModal = false"><ion-icon name="close"></ion-icon></button>
                </div>
                <div class="modal-body">
                    <div v-if="authError" class="form-error">{{ authError }}</div>
                    <div class="form-group">
                        <label>Email</label>
                        <input type="email" class="form-input" v-model="authForm.email" placeholder="email@example.com">
                    </div>
                    <div class="form-group" v-if="isAdmin">
                        <label>Name</label>
                        <input type="text" class="form-input" v-model="authForm.name" placeholder="Full name">
                    </div>
                    <div class="form-group">
                        <label>Password</label>
                        <input type="password" class="form-input" v-model="authForm.password" placeholder="Min 6 characters">
                    </div>
                    <div class="form-group" v-if="isAdmin">
                        <label>Role</label>
                        <select class="form-select" v-model="authForm.role_id">
                            <option value="">Select role...</option>
                            <option v-for="r in roles" :key="r.id" :value="r.id">{{ r.label }}</option>
                        </select>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-outline" @click="showAuthModal = false">Cancel</button>
                    <button class="btn btn-primary" @click="isAdmin ? createUserByAdmin() : handleAuth()">
                        <ion-icon :name="isAdmin ? 'person-add-outline' : 'log-in-outline'"></ion-icon>
                        {{ isAdmin ? 'Create User' : 'Sign In' }}
                    </button>
                </div>
            </div>
        </div>
    </div>

    <!-- Error Log Sidebar -->
    <transition name="slide-right">
        <div v-if="showErrorSidebar" class="error-sidebar-overlay" @click.self="toggleErrorSidebar">
            <div class="error-sidebar">
                <div class="error-sidebar-header">
                    <div class="error-sidebar-title">
                        <ion-icon name="bug-outline" style="color:var(--accent-red);font-size:20px"></ion-icon>
                        <span>Poller Errors</span>
                        <span class="error-badge" v-if="pollerStats.error_count > 0">{{ pollerStats.error_count }}</span>
                    </div>
                    <div class="error-sidebar-actions">
                        <button v-if="errorLog.length > 0" class="error-clear-btn" @click="clearAllErrors">
                            <ion-icon name="trash-outline"></ion-icon>
                            Clear
                        </button>
                        <button class="error-sidebar-close" @click="toggleErrorSidebar">
                            <ion-icon name="close-outline"></ion-icon>
                        </button>
                    </div>
                </div>
                <div class="error-sidebar-body">
                    <div v-if="errorLog.length === 0" class="error-empty">
                        <ion-icon name="checkmark-circle-outline" style="font-size:40px;color:var(--accent-green)"></ion-icon>
                        <p>No errors recorded</p>
                    </div>
                    <div v-else class="error-list">
                        <div v-for="(err, i) in errorLog" :key="i" class="error-item">
                            <div class="error-item-dot"></div>
                            <div class="error-item-content">
                                <div class="error-item-header">
                                    <span class="error-item-time">{{ err.timestamp }}</span>
                                    <span class="error-item-plc">{{ err.plc_ip }}</span>
                                </div>
                                <div class="error-item-msg">{{ err.message }}</div>
                                <div class="error-item-meta">
                                    <span class="error-item-tag">{{ err.device }}{{ err.offset }}</span>
                                </div>
                            </div>
                            <button class="error-item-dismiss" @click.stop="dismissError(i)">
                                <ion-icon name="close-outline"></ion-icon>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </transition>
</template>
