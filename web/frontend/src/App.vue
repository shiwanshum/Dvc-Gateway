<template>
  <div class="app-layout">
    <aside class="sidebar">
      <div class="logo">
        <h2>Dvc Gateway</h2>
      </div>
      <nav class="nav-links">
        <router-link to="/" class="nav-item">Dashboard</router-link>
        <router-link to="/plcs" class="nav-item">PLC Management</router-link>
        <router-link to="/tags" class="nav-item">Tag Management</router-link>
      </nav>
    </aside>
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
// Layout wrapper for the application
</script>

<style>
/* Reset and Base Styles applied here or in style.css */
body {
  margin: 0;
  padding: 0;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  background-color: #121212;
  color: #e0e0e0;
}
.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
.sidebar {
  width: 250px;
  background: #1e1e24;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
}
.logo {
  padding: 24px;
  border-bottom: 1px solid #333;
}
.logo h2 {
  margin: 0;
  color: #3498db;
  font-weight: 700;
  letter-spacing: 1px;
}
.nav-links {
  display: flex;
  flex-direction: column;
  padding: 20px 0;
}
.nav-item {
  padding: 16px 24px;
  color: #bbb;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s;
  border-left: 4px solid transparent;
}
.nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
}
.nav-item.router-link-active {
  background: rgba(52, 152, 219, 0.1);
  color: #3498db;
  border-left-color: #3498db;
}
.main-content {
  flex: 1;
  overflow-y: auto;
  background: radial-gradient(circle at top right, #1a1a24 0%, #121212 100%);
}
</style>
