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
/* Clean Sky-Blue and White Theme */
body {
  margin: 0;
  padding: 0;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  background-color: var(--bg-main, #f0f9ff);
  color: var(--text-primary, #0f172a);
}
.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
.sidebar {
  width: 250px;
  background: var(--bg-card, #ffffff);
  border-right: 1px solid var(--border-color, #bae6fd);
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 10px rgba(0,0,0,0.02);
}
.logo {
  padding: 24px;
  border-bottom: 1px solid var(--border-color, #bae6fd);
}
.logo h2 {
  margin: 0;
  color: var(--accent-blue, #0ea5e9);
  font-weight: 800;
  letter-spacing: 1px;
}
.nav-links {
  display: flex;
  flex-direction: column;
  padding: 20px 0;
}
.nav-item {
  padding: 16px 24px;
  color: var(--text-secondary, #475569);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.3s ease;
  border-left: 4px solid transparent;
  margin: 4px 12px;
  border-radius: 8px;
}
.nav-item:hover {
  background: rgba(14, 165, 233, 0.08);
  color: var(--accent-blue, #0ea5e9);
}
.nav-item.router-link-active {
  background: rgba(14, 165, 233, 0.15);
  color: var(--accent-blue, #0ea5e9);
  border-left-color: var(--accent-blue, #0ea5e9);
  box-shadow: 0 4px 12px rgba(14, 165, 233, 0.1);
}
.main-content {
  flex: 1;
  overflow-y: auto;
  background: radial-gradient(circle at top right, #e0f2fe 0%, #f0f9ff 100%);
}
</style>
