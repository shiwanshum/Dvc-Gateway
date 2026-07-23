<template>
  <div class="manager-view">
    <div class="header-section">
      <h1>Tag Management</h1>
      <button class="btn-primary" @click="openAddForm">Add Tag</button>
    </div>

    <!-- Search & Filter Bar -->
    <div class="search-bar">
      <input v-model="searchQuery" @input="fetchTags(1)" placeholder="Search by Tag Name..." class="search-input" />
      <select v-model="filterFacility" @change="fetchTags(1)" class="filter-select">
        <option value="">All Facilities</option>
        <option value="booth">booth</option>
        <option value="pretreatment">pretreatment</option>
        <option value="oven">oven</option>
      </select>
    </div>

    <div v-if="showForm" class="form-card">
      <h3>{{ isEditing ? 'Edit Tag' : 'Add New Tag' }}</h3>
      <form @submit.prevent="saveTag" class="tag-form">
        <input v-model="formTag.tag_name" placeholder="Tag Name (e.g. M100)" required />
        <input v-model="formTag.tag_address" placeholder="Address (e.g. M100)" required />
        <input v-model="formTag.fac_name" placeholder="Facility (e.g. booth)" required />
        <input v-model="formTag.plc_id" placeholder="PLC ID (UUID)" required />
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
            <th>Tag Name</th>
            <th>Address</th>
            <th>Facility</th>
            <th>PLC</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tag in tags" :key="tag.id">
            <td>{{ tag.tag_name }}</td>
            <td><code>{{ tag.tag_address }}</code></td>
            <td><span class="badge">{{ tag.fac_name }}</span></td>
            <td>{{ tag.plc?.ip_address || tag.plc_id }}</td>
            <td>
              <button class="btn-warning btn-sm" @click="openEditForm(tag)">Edit</button>
              <button class="btn-danger btn-sm" @click="deleteTag(tag.id)" style="margin-left:8px;">Delete</button>
            </td>
          </tr>
          <tr v-if="tags.length === 0">
            <td colspan="5" class="empty-state">No Tags found.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination Controls -->
    <div class="pagination-controls" v-if="totalPages > 1">
      <button class="btn-secondary btn-sm" :disabled="page <= 1" @click="fetchTags(page - 1)">Previous</button>
      <span class="page-info">Page {{ page }} of {{ totalPages }} (Total: {{ totalItems }})</span>
      <button class="btn-secondary btn-sm" :disabled="page >= totalPages" @click="fetchTags(page + 1)">Next</button>
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
