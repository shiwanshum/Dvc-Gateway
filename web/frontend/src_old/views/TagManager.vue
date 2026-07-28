<template>
  <div class="manager-view">
    <div class="header-section">
      <h1>Tag Management</h1>
      <button class="btn-3d" @click="openAddForm">Add Tag</button>
    </div>

    <!-- Search & Filter Bar -->
    <div class="search-bar">
      <input v-model="searchQuery" @input="fetchTags(1)" placeholder="Search by Tag Name..." class="input-3d search-input" />
      <select v-model="filterFacility" @change="fetchTags(1)" class="input-3d filter-select">
        <option value="">All Facilities</option>
        <option value="booth">booth</option>
        <option value="pretreatment">pretreatment</option>
        <option value="oven">oven</option>
      </select>
    </div>

    <div v-if="showForm" class="form-card glass-card">
      <h3>{{ isEditing ? 'Edit Tag' : 'Add New Tag' }}</h3>
      <form @submit.prevent="saveTag" class="tag-form">
        <input class="input-3d" v-model="formTag.tag_name" placeholder="Tag Name (e.g. M100)" required />
        <input class="input-3d" v-model="formTag.tag_address" placeholder="Address (e.g. M100)" required />
        <input class="input-3d" v-model="formTag.fac_name" placeholder="Facility (e.g. booth)" required />
        <input class="input-3d" v-model="formTag.plc_id" placeholder="PLC ID (UUID)" required />
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
            <th>Tag Name</th>
            <th>Address</th>
            <th>Facility</th>
            <th>PLC ID</th>
            <th>Created At</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tag in tags" :key="tag.id">
            <td><code class="uuid-text">{{ tag.id }}</code></td>
            <td class="highlight-text">{{ tag.tag_name }}</td>
            <td><code>{{ tag.tag_address }}</code></td>
            <td><span class="badge">{{ tag.fac_name }}</span></td>
            <td><code class="uuid-text">{{ tag.plc?.ip_address || tag.plc_id }}</code></td>
            <td class="date-text">{{ new Date(tag.created_at).toLocaleString() }}</td>
            <td class="actions-cell">
              <button class="btn-3d btn-sm" @click="openEditForm(tag)">Edit</button>
              <button class="btn-3d btn-sm danger" @click="deleteTag(tag.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="tags.length === 0">
            <td colspan="7" class="empty-state">No Tags found.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination Controls -->
    <div class="pagination-controls" v-if="totalPages > 1">
      <button class="btn-3d btn-sm" :disabled="page <= 1" @click="fetchTags(page - 1)">Previous</button>
      <span class="page-info">Page {{ page }} of {{ totalPages }} (Total: {{ totalItems }})</span>
      <button class="btn-3d btn-sm" :disabled="page >= totalPages" @click="fetchTags(page + 1)">Next</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';

const tags = ref([]);
const showForm = ref(false);
const isEditing = ref(false);
const formTag = ref({ id: null, tag_name: '', tag_address: '', fac_name: '', plc_id: '' });

// Pagination & Search
const page = ref(1);
const limit = ref(10);
const totalPages = ref(1);
const totalItems = ref(0);
const searchQuery = ref('');
const filterFacility = ref('');

const fetchTags = async (targetPage = page.value) => {
  try {
    const res = await axios.get(`http://${window.location.hostname}:6080/api/tags`, {
      params: {
        page: targetPage,
        limit: limit.value,
        search: searchQuery.value,
        filter: filterFacility.value
      }
    });
    tags.value = res.data.data || [];
    page.value = res.data.page;
    totalPages.value = res.data.total_pages;
    totalItems.value = res.data.total;
  } catch (err) {
    console.error(err);
  }
};

const openAddForm = () => {
  isEditing.value = false;
  formTag.value = { id: null, tag_name: '', tag_address: '', fac_name: '', plc_id: '' };
  showForm.value = true;
};

const openEditForm = (tag) => {
  isEditing.value = true;
  formTag.value = { ...tag };
  showForm.value = true;
};

const saveTag = async () => {
  try {
    if (isEditing.value) {
      await axios.put(`http://${window.location.hostname}:6080/api/tags/${formTag.value.id}`, formTag.value);
    } else {
      await axios.post(`http://${window.location.hostname}:6080/api/tags`, formTag.value);
    }
    showForm.value = false;
    fetchTags(isEditing.value ? page.value : 1);
  } catch (err) {
    console.error(err);
  }
};

const deleteTag = async (id) => {
  if (!confirm('Are you sure you want to delete this tag?')) return;
  try {
    await axios.delete(`http://${window.location.hostname}:6080/api/tags/${id}`);
    fetchTags();
  } catch (err) {
    console.error(err);
  }
};

onMounted(() => fetchTags(1));
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
.tag-form {
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
  color: var(--accent-neon);
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
