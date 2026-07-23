<template>
  <div class="dashboard-view">
    <div class="header-section">
      <h1>Real-Time Facility & PLC Health Dashboard</h1>
      <div class="status-indicator" :class="{ connected: wsConnected }">
        {{ wsConnected ? 'Live Data Connected' : 'Connecting...' }}
      </div>
    </div>
    
    <!-- Top Stats Row -->
    <div class="stats-row">
      <div class="stat-card">
        <h3>Total PLCs</h3>
        <div class="stat-val">{{ health.length }}</div>
      </div>
      <div class="stat-card success">
        <h3>Online</h3>
        <div class="stat-val">{{ health.filter(h => h.status === 'online').length }}</div>
      </div>
      <div class="stat-card error">
        <h3>Offline</h3>
        <div class="stat-val">{{ health.filter(h => h.status === 'offline').length }}</div>
      </div>
      <div class="stat-card">
        <h3>Read Latency (Global)</h3>
        <div class="stat-val">{{ pollerStats.read_latency_ms || 0 }} <span class="unit">ms</span></div>
      </div>
      <div class="stat-card">
        <h3>Write Latency (Global)</h3>
        <div class="stat-val">{{ pollerStats.write_latency_ms || 0 }} <span class="unit">ms</span></div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="charts-row">
      <div class="chart-container">
        <div class="chart-header">
          <h2>PLC Performance Trend (RTT)</h2>
          <div class="plc-selector">
            <span class="plc-chip" :class="{ active: selectedChartPlc.includes('Global') }" @click="toggleChartPlc('Global')">Global Stats</span>
            <span v-for="plc in health" :key="plc.id" class="plc-chip" :class="{ active: selectedChartPlc.includes(plc.id) }" @click="toggleChartPlc(plc.id)">
              {{ plc.name || plc.ip_address }}
            </span>
          </div>
        </div>
        <v-chart class="chart" :option="rttChartOption" autoresize />
      </div>
    </div>

    <!-- PLCs Status Grid -->
    <div class="section-header">
      <h2>PLC Connection Status</h2>
    </div>
    <div class="plcs-grid">
      <div v-for="plc in health" :key="plc.id" class="plc-card" :class="plc.status">
        <div class="plc-header">
          <h3>{{ plc.name || 'Unknown PLC' }}</h3>
          <div style="display:flex; align-items:center; gap:8px;">
            <button class="icon-btn" @click="scanPlc(plc.id)" title="Scan Ports & Reconnect" style="background:transparent; border:none; color:var(--text-secondary); cursor:pointer;">
              <ion-icon name="refresh-outline" style="font-size: 1.2rem;"></ion-icon>
            </button>
            <span class="status-badge">{{ plc.status }}</span>
          </div>
        </div>
        <div class="plc-body">
          <p><strong>IP:</strong> {{ plc.ip_address }}</p>
          <p><strong>Latency:</strong> <span class="latency">{{ plc.latency_ms.toFixed(1) }} ms</span></p>
        </div>
      </div>
    </div>

    <!-- Tags by Facility Grid -->
    <div class="section-header" style="margin-top: 30px;">
      <h2>Live Facility Tags</h2>
    </div>
    <div class="facilities-grid">
      <div v-for="(facilityTags, facilityName) in tagsByFacility" :key="facilityName" class="facility-card">
        <h2>{{ facilityName || 'Unassigned Facility' }}</h2>
        <div class="tag-list">
          <div v-for="tag in facilityTags" :key="tag.id" class="tag-item">
            <span class="tag-name">{{ tag.tag_name }} ({{ tag.tag_address }})</span>
            <span class="tag-value" :class="{ 'updated': highlightTags[tag.id] }">
              {{ tagValues[tag.id] !== undefined ? tagValues[tag.id] : '--' }}
            </span>
          </div>
          <div v-if="facilityTags.length === 0" class="no-tags">
            No tags configured for this facility.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue';
import axios from 'axios';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart, BarChart } from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  GridComponent,
  MarkAreaComponent,
  VisualMapComponent
} from 'echarts/components';
import VChart from 'vue-echarts';

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
]);

// ----------------- STATE -----------------
const tags = ref([]);
const tagValues = ref({});
const highlightTags = ref({});
const wsConnected = ref(false);
let ws = null;

const health = ref([]);
const pollerStats = ref({ read_latency_ms: 0, write_latency_ms: 0, read_ops: 0, write_ops: 0, error_count: 0 });

const rttHistory = ref([]);
const selectedChartPlc = ref(['Global']);
const rttHistoryLimit = ref(50);
const rttTimeRange = ref(11000); // 11 Seconds default

let pollingInterval = null;

// ----------------- COMPUTED -----------------
const tagsByFacility = computed(() => {
  const grouped = {
    'booth': [],
    'pretreatment': [],
    'oven': []
  };
  
  tags.value.forEach(tag => {
    const fac = (tag.fac_name || '').toLowerCase();
    if (grouped[fac]) {
      grouped[fac].push(tag);
    } else {
      if (!grouped['Other']) grouped['Other'] = [];
      grouped['Other'].push(tag);
    }
  });
  return grouped;
});

const rttChartOption = computed(() => {
    const now = Date.now();
    const colors = ['#5470C6', '#EE6666', '#91CC75', '#FAC858', '#73C0DE', '#3BA272', '#FC8452', '#9A60B4'];
    
    let series = [];
    let legendData = [];
    let times = [];
    let colorIdx = 0;

    selectedChartPlc.value.forEach(selId => {
        let hist = rttHistory.value.filter(h => h.plc_id === selId && (now - h.time.getTime() <= rttTimeRange.value));
        hist = hist.slice(-rttHistoryLimit.value);

        if (times.length === 0 && hist.length > 0) {
            times = hist.map(h => {
                const d = h.time;
                return `${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}:${d.getSeconds().toString().padStart(2,'0')}`;
            });
        }

        let label = selId === 'Global' ? 'Global' : (health.value.find(p => p.id === selId)?.ip_address || selId);

        legendData.push(`${label} Read`);
        legendData.push(`${label} Write`);

        series.push({
            name: `${label} Read`,
            type: 'line',
            smooth: true,
            emphasis: { focus: 'series' },
            itemStyle: { color: colors[colorIdx % colors.length] },
            data: hist.map(h => h.read)
        });

        series.push({
            name: `${label} Write`,
            type: 'line',
            smooth: true,
            emphasis: { focus: 'series' },
            itemStyle: { color: colors[(colorIdx+1) % colors.length] },
            data: hist.map(h => h.write)
        });

        colorIdx += 2;
    });

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
            top: 40, bottom: 40, left: '3%', right: '4%', containLabel: true
        },
        legend: {
            data: legendData,
            textStyle: { color: '#ccc' },
            type: 'scroll',
            top: 0
        },
        xAxis: [
            {
                type: 'category',
                boundaryGap: false,
                axisTick: { alignWithLabel: true },
                axisLine: { lineStyle: { color: '#666' } },
                data: times
            }
        ],
        yAxis: [
            {
                type: 'value',
                axisLabel: { formatter: '{value} ms', color: '#666' },
                splitLine: { lineStyle: { color: 'rgba(0,0,0,0.05)' } },
                min: 0,
                max: function(value) {
                    return Math.max(10, Math.ceil(value.max * 1.2));
                }
            }
        ],
        series: series
    };
});

// ----------------- METHODS -----------------
function toggleChartPlc(id) {
    const idx = selectedChartPlc.value.indexOf(id);
    if (idx > -1) {
        if (selectedChartPlc.value.length > 1) {
            selectedChartPlc.value.splice(idx, 1);
        }
    } else {
        selectedChartPlc.value.push(id);
    }
}

function trimHistory() {
    const maxEntries = 5000;
    if (rttHistory.value.length > maxEntries * 5) {
        rttHistory.value = rttHistory.value.slice(-maxEntries * 2);
    }
}

const scanPlc = async (id) => {
  try {
    const res = await axios.post(`http://${window.location.hostname}:6080/api/plcs/${id}/scan`);
    alert(res.data.message || 'Scan initiated');
    fetchPLCHealth();
  } catch (err) {
    console.error(err);
    alert('Failed to trigger scan');
  }
};

const fetchTags = async () => {
  try {
    const res = await axios.get(`http://${window.location.hostname}:6080/api/tags?limit=1000`);
    tags.value = res.data.data || [];
  } catch (err) {
    console.error('Failed to fetch tags', err);
  }
};

const fetchPLCHealth = async () => {
    try {
        const res = await axios.get(`http://${window.location.hostname}:6080/api/health/plcs`);
        health.value = res.data || [];
        const now = new Date();
        (res.data || []).forEach(p => {
            rttHistory.value.push({ time: now, read: p.latency_ms || 0, write: 0, plc_id: p.id, ops: 0 });
        });
        trimHistory();
    } catch (err) {
        console.error('Failed to fetch PLC health', err);
    }
};

const fetchPollerStats = async () => {
    try {
        const res = await axios.get(`http://${window.location.hostname}:6080/api/poller-stats`);
        pollerStats.value = res.data || {};
        rttHistory.value.push({ 
          time: new Date(), 
          read: res.data.read_latency_ms || 0, 
          write: res.data.write_latency_ms || 0, 
          plc_id: 'Global', 
          ops: res.data.read_ops || 0 
        });
        trimHistory();
    } catch (err) {
        console.error('Failed to fetch Poller Stats', err);
    }
};

const connectWs = () => {
  ws = new WebSocket(`ws://${window.location.hostname}:6080/ws`);
  
  ws.onopen = () => {
    wsConnected.value = true;
  };
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.registers && data.tag_ids) {
        const updates = {};
        for (const [key, value] of Object.entries(data.registers)) {
          const tagId = data.tag_ids[key];
          if (tagId) {
            updates[tagId] = value;
            
            // Highlight animation
            highlightTags.value[tagId] = true;
            setTimeout(() => {
              highlightTags.value[tagId] = false;
            }, 300);
          }
        }
        tagValues.value = { ...tagValues.value, ...updates };
      }
    } catch (e) {
      console.error('Error parsing WS message', e);
    }
  };
  
  ws.onclose = () => {
    wsConnected.value = false;
    setTimeout(connectWs, 3000); // Reconnect after 3s
  };
};

// ----------------- LIFECYCLE -----------------
onMounted(() => {
  fetchTags();
  connectWs();
  
  // Initial Fetch
  fetchPLCHealth();
  fetchPollerStats();
  
  // Start Polling every 2 seconds
  pollingInterval = setInterval(() => {
    fetchPLCHealth();
    fetchPollerStats();
  }, 2000);
});

onUnmounted(() => {
  if (ws) ws.close();
  if (pollingInterval) clearInterval(pollingInterval);
});
</script>

<style scoped>
.dashboard-view {
  padding: 20px;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.status-indicator {
  padding: 8px 16px;
  border-radius: 20px;
  background: #333;
  color: #aaa;
  font-weight: 500;
  transition: all 0.3s ease;
}
.status-indicator.connected {
  background: rgba(46, 213, 115, 0.2);
  color: #2ed573;
  box-shadow: 0 0 10px rgba(46, 213, 115, 0.2);
}

.section-header {
  margin-bottom: 16px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
  padding-bottom: 8px;
}

/* Stats Row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: rgba(30, 30, 35, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 16px;
  text-align: center;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
}
.stat-card h3 {
  margin: 0;
  font-size: 0.9rem;
  color: #aaa;
  text-transform: uppercase;
  letter-spacing: 1px;
}
.stat-val {
  font-size: 2rem;
  font-weight: bold;
  color: #fff;
  margin-top: 8px;
}
.stat-val .unit {
  font-size: 1rem;
  color: #888;
}
.stat-card.success .stat-val { color: #2ed573; }
.stat-card.error .stat-val { color: #ff4757; }

/* Charts */
.charts-row {
  margin-bottom: 30px;
}
.chart-container {
  background: rgba(30, 30, 35, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.chart-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: #fff;
}
.plc-selector {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.plc-chip {
  padding: 6px 12px;
  border-radius: 16px;
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.1);
  color: #aaa;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}
.plc-chip:hover {
  background: rgba(255,255,255,0.1);
}
.plc-chip.active {
  background: linear-gradient(135deg, #0ea5e9, #38bdf8);
  color: #fff;
  border-color: transparent;
  box-shadow: 0 4px 12px rgba(56,189,248,0.35);
}
.chart {
  height: 400px;
  width: 100%;
}

/* PLCs Grid */
.plcs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}
.plc-card {
  background: rgba(30, 30, 35, 0.8);
  border-left: 4px solid #777;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
}
.plc-card.online {
  border-left-color: #2ed573;
}
.plc-card.offline {
  border-left-color: #ff4757;
}
.plc-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.plc-header h3 {
  margin: 0;
  font-size: 1.1rem;
  color: #fff;
}
.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: bold;
  text-transform: uppercase;
}
.plc-card.online .status-badge {
  background: rgba(46, 213, 115, 0.2);
  color: #2ed573;
}
.plc-card.offline .status-badge {
  background: rgba(255, 71, 87, 0.2);
  color: #ff4757;
}
.plc-body p {
  margin: 4px 0;
  font-size: 0.9rem;
  color: #ccc;
}
.plc-body .latency {
  font-family: monospace;
  color: #38bdf8;
  font-size: 1rem;
}

/* Tags Grid */
.facilities-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 24px;
}
.facility-card {
  background: rgba(30, 30, 35, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}
.facility-card h2 {
  margin-top: 0;
  margin-bottom: 16px;
  color: #fff;
  text-transform: capitalize;
  font-size: 1.2rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
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
  background: rgba(0, 0, 0, 0.2);
  padding: 12px 16px;
  border-radius: 8px;
}
.tag-name {
  color: #ddd;
  font-size: 0.95rem;
}
.tag-value {
  font-family: monospace;
  font-size: 1.1rem;
  font-weight: bold;
  color: #3498db;
  transition: color 0.3s ease;
}
.tag-value.updated {
  color: #f1c40f;
}
.no-tags {
  color: #777;
  font-style: italic;
  text-align: center;
  padding: 20px 0;
}
</style>
