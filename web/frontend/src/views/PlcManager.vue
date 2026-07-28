<template>
  <div class="manager-view">
    <div class="header-section">
      <h1>PLC Management</h1>
      <button class="btn-3d" @click="openAddForm">Add PLC</button>
    </div>

    <!-- Search & Filter Bar -->
    <div class="search-bar">
      <input v-model="searchQuery" @input="fetchPlcs(1)" placeholder="Search by IP Address..." class="input-3d search-input" />
      <select v-model="filterFacility" @change="fetchPlcs(1)" class="input-3d filter-select">
        <option value="">All Facilities</option>
        <option value="booth">booth</option>
        <option value="pretreatment">pretreatment</option>
        <option value="oven">oven</option>
      </select>
    </div>

    <div v-if="showForm" class="form-card glass-card">
      <h3>{{ isEditing ? 'Edit PLC' : 'Add New PLC' }}</h3>
      <form @submit.prevent="savePlc" class="plc-form">
        <input class="input-3d" v-model="formPlc.ip_address" placeholder="IP Address (e.g. 192.168.1.10)" required />
        <input class="input-3d" v-model="formPlc.port" type="number" placeholder="Port" required />
        <input class="input-3d" v-model="formPlc.facility_name" placeholder="Facility (e.g. booth)" required />
        <input class="input-3d" v-model="formPlc.driver" placeholder="Driver (e.g. mitsubishi_mc)" required />
        <div class="form-actions">
          <button type="submit" class="btn-3d write">Save</button>
          <button type="button" class="btn-3d" @click="showForm = false">Cancel</button>
        </div>
      </form>
    </div>

    <div class="table-container glass-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID (UUID)</th>
            <th>IP Address</th>
            <th>Port</th>
            <th>Facility</th>
            <th>Driver</th>
            <th>Status</th>
            <th>Created At</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="plc in plcs" :key="plc.id">
            <td><code class="uuid-text">{{ plc.id }}</code></td>
            <td class="highlight-text">{{ plc.ip_address }}</td>
            <td>{{ plc.port }}</td>
            <td><span class="badge">{{ plc.facility_name }}</span></td>
            <td>{{ plc.driver }}</td>
            <td>
              <span class="badge" :class="getHealthStatus(plc.id) === 'online' ? 'badge-green' : 'badge-red'">
                {{ getHealthStatus(plc.id) }}
              </span>
            </td>
            <td class="date-text">{{ new Date(plc.created_at).toLocaleString() }}</td>
            <td class="actions-cell">
              <button class="btn-3d btn-sm" @click="scanPlc(plc.id)" title="Scan Ports & Reconnect">
                Scan
              </button>
              <button class="btn-3d btn-sm" @click="openEditForm(plc)">Edit</button>
              <button class="btn-3d btn-sm danger" @click="deletePlc(plc.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="plcs.length === 0">
            <td colspan="8" class="empty-state">No PLCs found.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination Controls -->
    <div class="pagination-controls" v-if="totalPages > 1">
      <button class="btn-3d btn-sm" :disabled="page <= 1" @click="fetchPlcs(page - 1)">Previous</button>
      <span class="page-info">Page {{ page }} of {{ totalPages }} (Total: {{ totalItems }})</span>
      <button class="btn-3d btn-sm" :disabled="page >= totalPages" @click="fetchPlcs(page + 1)">Next</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';

const plcs = ref([]);
const showForm = ref(false);
const isEditing = ref(false);
const formPlc = ref({ id: null, ip_address: '', port: null, facility_name: '', driver: 'mitsubishi_mc' });
const health = ref([]);

// Pagination & Search
const page = ref(1);
const limit = ref(10);
const totalPages = ref(1);
const totalItems = ref(0);
const searchQuery = ref('');
const filterFacility = ref('');

const fetchPlcs = async (targetPage = page.value) => {
  try {
    const res = await axios.get(`http://${window.location.hostname}:6080/api/plcs`, {
      params: {
        page: targetPage,
        limit: limit.value,
        search: searchQuery.value,
        filter: filterFacility.value
      }
    });
    plcs.value = res.data.data || [];
    page.value = res.data.page;
    totalPages.value = res.data.total_pages;
    totalItems.value = res.data.total;
  } catch (err) {
    console.error(err);
  }
};

const fetchHealth = async () => {
  try {
    const res = await axios.get(`http://${window.location.hostname}:6080/api/health/plcs`);
    health.value = res.data || [];
  } catch (err) {
    console.error(err);
  }
};

const getHealthStatus = (id) => {
  const h = health.value.find(h => h.id === id);
  return h ? h.status : 'unknown';
};

const openAddForm = () => {
  isEditing.value = false;
  formPlc.value = { id: null, ip_address: '', port: null, facility_name: '', driver: 'mitsubishi_mc' };
  showForm.value = true;
};

const openEditForm = (plc) => {
  isEditing.value = true;
  formPlc.value = { ...plc };
  showForm.value = true;
};

const savePlc = async () => {
  try {
    if (isEditing.value) {
      await axios.put(`http://${window.location.hostname}:6080/api/plcs/${formPlc.value.id}`, formPlc.value);
    } else {
      await axios.post(`http://${window.location.hostname}:6080/api/plcs`, formPlc.value);
    }
    showForm.value = false;
    fetchPlcs(isEditing.value ? page.value : 1);
  } catch (err) {
    console.error(err);
  }
};

const deletePlc = async (id) => {
  if (!confirm('Are you sure you want to delete this PLC?')) return;
  try {
    await axios.delete(`http://${window.location.hostname}:6080/api/plcs/${id}`);
    fetchPlcs();
  } catch (err) {
    console.error(err);
  }
};

const scanPlc = async (id) => {
  try {
    const res = await axios.post(`http://${window.location.hostname}:6080/api/plcs/${id}/scan`);
    alert(res.data.message || 'Scan initiated');
    fetchPlcs();
  } catch (err) {
    console.error(err);
    alert('Failed to trigger scan');
  }
};

let healthInterval;

onMounted(() => {
  fetchPlcs(1);
  fetchHealth();
  healthInterval = setInterval(fetchHealth, 5000);
});
</script>

<style scoped>
.manager-view {
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

.search-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}
.search-input {
  flex: 1;
  max-width: 400px;
}

.glass-card {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 24px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(12px);
}

.form-card h3 {
  color: var(--text-primary);
  margin-top: 0;
  margin-bottom: 16px;
}
.plc-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}
.form-actions {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.table-container {
  overflow-x: auto;
  padding: 0; /* Override */
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  color: var(--text-primary);
}
.data-table th, .data-table td {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}
.data-table th {
  background: rgba(15, 23, 42, 0.6);
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  font-size: 0.85rem;
  letter-spacing: 1px;
}
.data-table tr:hover {
  background: rgba(255, 255, 255, 0.02);
}

.uuid-text {
  font-size: 0.8rem;
  color: var(--text-secondary);
  background: rgba(0,0,0,0.3);
  padding: 4px 8px;
  border-radius: 4px;
}
.highlight-text {
  font-weight: 700;
  color: var(--accent-blue);
}
.date-text {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.badge {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid var(--border-color);
}
.badge-green {
  background: rgba(16, 185, 129, 0.15);
  color: var(--accent-neon);
  border-color: var(--accent-neon);
}
.badge-red {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border-color: #f87171;
}

.actions-cell {
  display: flex;
  gap: 8px;
}

.btn-sm {
  padding: 6px 12px;
  font-size: 0.8rem;
  box-shadow: 0 2px 0 #0f172a, 0 3px 5px rgba(0,0,0,0.3);
}
.btn-sm.danger {
  border-color: #f87171;
  color: #f87171;
}
.btn-sm.danger:hover {
  box-shadow: 0 2px 0 #991b1b, 0 8px 20px rgba(248, 113, 113, 0.3);
}

.empty-state {
  text-align: center;
  color: var(--text-secondary);
  padding: 32px !important;
  font-style: italic;
}

.pagination-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
}
.page-info {
  color: var(--text-secondary);
  font-weight: 600;
}
</style>
