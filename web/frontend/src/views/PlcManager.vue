<template>
  <div class="manager-view">
    <div class="header-section">
      <h1>PLC Management</h1>
      <button class="btn-primary" @click="openAddForm">Add PLC</button>
    </div>

    <!-- Search & Filter Bar -->
    <div class="search-bar">
      <input v-model="searchQuery" @input="fetchPlcs(1)" placeholder="Search by IP Address..." class="search-input" />
      <select v-model="filterFacility" @change="fetchPlcs(1)" class="filter-select">
        <option value="">All Facilities</option>
        <option value="booth">booth</option>
        <option value="pretreatment">pretreatment</option>
        <option value="oven">oven</option>
      </select>
    </div>

    <div v-if="showForm" class="form-card">
      <h3>{{ isEditing ? 'Edit PLC' : 'Add New PLC' }}</h3>
      <form @submit.prevent="savePlc" class="plc-form">
        <input v-model="formPlc.ip_address" placeholder="IP Address (e.g. 192.168.1.10)" required />
        <input v-model="formPlc.port" type="number" placeholder="Port" required />
        <input v-model="formPlc.facility_name" placeholder="Facility (e.g. booth)" required />
        <input v-model="formPlc.driver" placeholder="Driver (e.g. mitsubishi_mc)" required />
        <div class="form-actions">
          <button type="submit" class="btn-success">Save</button>
          <button type="button" class="btn-secondary" @click="showForm = false">Cancel</button>
        </div>
      </form>
    </div>

    <div class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>IP Address</th>
            <th>Port</th>
            <th>Facility</th>
            <th>Driver</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="plc in plcs" :key="plc.id">
            <td>{{ plc.ip_address }}</td>
            <td>{{ plc.port }}</td>
            <td><span class="badge badge-blue">{{ plc.facility_name }}</span></td>
            <td>{{ plc.driver }}</td>
            <td>
              <span class="badge" :class="getHealthStatus(plc.id) === 'online' ? 'badge-green' : 'badge-red'">
                {{ getHealthStatus(plc.id) }}
              </span>
            </td>
            <td>
              <button class="btn-primary btn-sm" @click="scanPlc(plc.id)" style="margin-right:8px;" title="Scan Ports & Reconnect">
                <ion-icon name="refresh-outline" style="vertical-align: middle;"></ion-icon> Scan
              </button>
              <button class="btn-warning btn-sm" @click="openEditForm(plc)">Edit</button>
              <button class="btn-danger btn-sm" @click="deletePlc(plc.id)" style="margin-left:8px;">Delete</button>
            </td>
          </tr>
          <tr v-if="plcs.length === 0">
            <td colspan="6" class="empty-state">No PLCs found.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination Controls -->
    <div class="pagination-controls" v-if="totalPages > 1">
      <button class="btn-secondary btn-sm" :disabled="page <= 1" @click="fetchPlcs(page - 1)">Previous</button>
      <span class="page-info">Page {{ page }} of {{ totalPages }} (Total: {{ totalItems }})</span>
      <button class="btn-secondary btn-sm" :disabled="page >= totalPages" @click="fetchPlcs(page + 1)">Next</button>
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
