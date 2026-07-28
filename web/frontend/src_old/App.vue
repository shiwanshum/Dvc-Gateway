<template>
  <div class="app-layout">
    <aside class="sidebar">
      <div class="logo">
        <h2>DVC Gateway</h2>
      </div>
      <nav class="nav-links">
        <router-link to="/" class="nav-item">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path></svg>
          Dashboard
        </router-link>
        <router-link to="/plcs" class="nav-item">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect><rect x="9" y="9" width="6" height="6"></rect><line x1="9" y1="1" x2="9" y2="4"></line><line x1="15" y1="1" x2="15" y2="4"></line><line x1="9" y1="20" x2="9" y2="23"></line><line x1="15" y1="20" x2="15" y2="23"></line><line x1="20" y1="9" x2="23" y2="9"></line><line x1="20" y1="14" x2="23" y2="14"></line><line x1="1" y1="9" x2="4" y2="9"></line><line x1="1" y1="14" x2="4" y2="14"></line></svg>
          PLC Config
        </router-link>
        <router-link to="/tags" class="nav-item">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
          Tags Config
        </router-link>
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
/* High-Tech Dark Theme */
:root {
  --bg-main: #0f172a;
  --bg-card: #1e293b;
  --text-primary: #f8fafc;
  --text-secondary: #94a3b8;
  --accent-neon: #10b981;
  --accent-blue: #38bdf8;
  --border-color: #334155;
}
body {
  margin: 0;
  padding: 0;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  background-color: var(--bg-main);
  color: var(--text-primary);
}
.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
.sidebar {
  width: 250px;
  background: var(--bg-card);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  box-shadow: 4px 0 20px rgba(0,0,0,0.4);
  z-index: 10;
}
.logo {
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
  text-align: center;
}
.logo h2 {
  margin: 0;
  color: var(--accent-neon);
  font-weight: 900;
  letter-spacing: 2px;
  text-transform: uppercase;
  text-shadow: 0 0 10px rgba(16, 185, 129, 0.4);
}
.nav-links {
  display: flex;
  flex-direction: column;
  padding: 20px 0;
}
.nav-item {
  padding: 16px 24px;
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.3s ease;
  border-left: 4px solid transparent;
  margin: 4px 12px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.nav-item .icon {
  width: 20px;
  height: 20px;
}
.nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  transform: translateX(4px);
}
.nav-item.router-link-active {
  background: rgba(16, 185, 129, 0.1);
  color: var(--accent-neon);
  border-left-color: var(--accent-neon);
  box-shadow: 0 4px 15px rgba(16, 185, 129, 0.15);
}
.main-content {
  flex: 1;
  overflow-y: auto;
  background: radial-gradient(circle at top right, #1e293b 0%, #0f172a 100%);
}

/* 3D Utility Classes used across views */
.btn-3d {
  background: #1e293b;
  color: #f8fafc;
  border: 1px solid #334155;
  padding: 10px 24px;
  border-radius: 8px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 0 #0f172a, 0 5px 10px rgba(0,0,0,0.3);
  text-transform: uppercase;
  letter-spacing: 1px;
}
.btn-3d:hover {
  border-color: var(--accent-neon);
  color: var(--accent-neon);
  box-shadow: 0 4px 0 #047857, 0 8px 20px rgba(16, 185, 129, 0.3);
  transform: translateY(-2px);
}
.btn-3d:active {
  transform: translateY(2px);
  box-shadow: 0 0 0 transparent, 0 2px 5px rgba(16, 185, 129, 0.3);
}
.btn-3d.write {
  border-color: #38bdf8;
  color: #38bdf8;
}
.btn-3d.write:hover {
  box-shadow: 0 4px 0 #0284c7, 0 8px 20px rgba(56, 189, 248, 0.3);
}

.input-3d {
  background: #0f172a;
  border: 1px solid #334155;
  color: #f8fafc;
  padding: 10px 16px;
  border-radius: 8px;
  outline: none;
  transition: all 0.3s ease;
  box-shadow: inset 0 2px 4px rgba(0,0,0,0.4);
  font-family: 'JetBrains Mono', monospace;
  font-size: 1rem;
}
.input-3d:focus {
  border-color: var(--accent-neon);
  box-shadow: inset 0 2px 4px rgba(0,0,0,0.4), 0 0 12px rgba(16, 185, 129, 0.25);
}
</style>
