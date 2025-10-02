# DevOps Lab v2.0 - Developer Quick Guide

> 🚀 Keyingi dasturchi uchun tez boshlash qo'llanmasi

---

## 📋 Loyiha Haqida

**Nima bu?** Interaktiv DevOps o'rganish platformasi - real Linux container'larda amaliy mashqlar.

**Asosiy Texnologiyalar:**
- Backend: Node.js (native HTTP server, custom WebSocket)
- Frontend: Vue.js 3 + TypeScript + Vite
- Database: PostgreSQL 17
- Container: Docker (dockerode SDK)

---

## 🏗️ Arxitektura (Bir qarashda)

```
User Browser (Vue.js)
    ↓ HTTP/REST + WebSocket
Backend (Node.js - /tmp/backend-production.js)
    ↓                    ↓
PostgreSQL          Docker Engine
(users, sessions,   (Real containers)
 progress)
```

---

## 🎯 Asosiy Mantiq

### 1. Session Hayot Sikli

```
Start Session → 1 ta container yaratiladi
    ↓
Next/Previous → Faqat UI o'zgaradi, container qoladi ✅
    ↓
Check → Hozirgi task'ni tekshiradi (taskId frontend'dan keladi)
    ↓
Stop/Logout → Container o'chadi
```

**MUHIM:** Bitta session = barcha tasklar uchun!

### 2. Task Validation

```javascript
// Frontend current task ID yuboradi
POST /api/check/:sessionId { taskId: "nginx-health" }

// Backend o'sha task'ni tekshiradi
const task = taskDetails[taskId]; // From request body
const container = containers.get(sessionId); // Same container

// Container ichida validation
await container.exec(['which', 'nginx']);
await container.exec(['test', '-f', '/var/www/html/index.html']);
// ...

// Score hisoblash
const score = (passedChecks / totalChecks) * task.points;
```

---

## 📦 Fayllar Tuzilishi

```
/tmp/backend-production.js          # 🔥 ASOSIY BACKEND (ishlaydigan)

frontend-vue/
├── src/
│   ├── views/
│   │   └── LabView.vue             # Asosiy interface
│   ├── components/
│   │   ├── Terminal.vue            # xterm.js terminal
│   │   ├── TaskDetail.vue          # Task tavsifi
│   │   ├── CheckResults.vue        # Natijalar
│   │   └── ProgressSummary.vue     # Umumiy score
│   ├── stores/
│   │   ├── auth.ts                 # Login/logout
│   │   ├── session.ts              # Session boshqaruv
│   │   └── task.ts                 # Task ma'lumotlari
│   └── services/
│       └── api.ts                  # HTTP client (Axios)

backend/tasks/                      # Task YAML fayllar (deprecated)
Dockerfile.simple                   # Custom Docker image
```

---

## 🚀 Ishga Tushirish (Development)

### 1. PostgreSQL
```bash
# Start
brew services start postgresql@17  # macOS
sudo systemctl start postgresql    # Linux

# Create DB (birinchi marta)
psql postgres
CREATE DATABASE devopslab;
CREATE USER devopslab WITH PASSWORD 'devopslab_secret';
GRANT ALL PRIVILEGES ON DATABASE devopslab TO devopslab;
\q
```

### 2. Docker Image
```bash
cd /Users/macbook/Documents/devops/devops-lab
docker build -f Dockerfile.simple -t devopslab:latest .
```

### 3. Backend
```bash
cd /tmp
node backend-production.js
# Port 8080 da ishga tushadi
```

### 4. Frontend
```bash
cd /Users/macbook/Documents/devops/devops-lab/frontend-vue
npm install  # Birinchi marta
npm run dev
# Port 3000 da ishga tushadi
```

### 5. Test
```bash
# Browser
open http://localhost:3000

# Login
username: student1
password: password123

# Yoki register qiling
```

---

## 🛠️ Asosiy Komponentlar

### Backend (/tmp/backend-production.js)

**Qayerda?** `/tmp/backend-production.js`

**Nima qiladi?**
- HTTP server (port 8080)
- WebSocket upgrade handler
- Docker container management
- PostgreSQL database operations
- Task validation logic

**Key Functions:**
```javascript
// Session yaratish
POST /api/sessions { taskId }
  → docker.createContainer()
  → container.start()
  → Save to PostgreSQL

// Task tekshirish
POST /api/check/:sessionId { taskId }
  → container.exec(['command', 'args'])
  → Calculate score
  → Save progress

// Cleanup
DELETE /api/sessions/:sessionId
  → container.stop()
  → container.remove()
  → DELETE FROM sessions
```

### Frontend (Vue.js)

**Main View:** `frontend-vue/src/views/LabView.vue`

**Struktura:**
```vue
<template>
  <Header>
    Timer | Progress Button | Logout
  </Header>

  <Sidebar>
    Task Navigation (Previous/Next/Skip)
    TaskDetail (description, hints)
    Start/Stop/Check Buttons
    CheckResults (validation natijalar)
  </Sidebar>

  <Terminal>
    xterm.js + WebSocket
  </Terminal>
</template>

<script setup>
// Navigation - faqat UI
function handleNext() {
  currentTaskIndex++;
  taskStore.loadTask(tasks[currentTaskIndex].id);
  // Container o'zgarmaydi!
}

// Check - current task
async function handleCheck() {
  await sessionStore.checkSolution(currentTask.id);
}
</script>
```

---

## 🔧 API Endpoints

### Authentication
```
POST /api/auth/register { username, email, password }
POST /api/auth/login    { username, password }
```

### Tasks
```
GET /api/tasks           # Barcha tasklar
GET /api/tasks/:id       # Bitta task detali
```

### Sessions
```
POST   /api/sessions          { taskId }
DELETE /api/sessions/:id
```

### Validation
```
POST /api/check/:sessionId   { taskId }
```

### Progress
```
GET /api/progress            # User'ning umumiy progress'i
```

### WebSocket
```
WS /ws?sessionId=xxx         # Terminal connection
```

---

## 🐳 Docker Bilan Ishlash

### Container Yaratish
```javascript
const container = await docker.createContainer({
  Image: 'devopslab:latest',
  name: `devops-lab-${sessionId}`,
  Tty: true,
  OpenStdin: true,
  Cmd: ['/bin/bash'],
  Env: ['TERM=xterm-256color'],
  HostConfig: {
    Memory: 512 * 1024 * 1024,  // 512MB
    CpuShares: 512               // 50% CPU
  }
});

await container.start();
```

### Command Bajarish
```javascript
// Interactive bash (WebSocket uchun)
const exec = await container.exec({
  Cmd: ['/bin/bash'],
  AttachStdin: true,
  AttachStdout: true,
  Tty: true
});

const stream = await exec.start({
  hijack: true  // Bidirectional stream
});

// Validation command
const exec2 = await container.exec({
  Cmd: ['which', 'nginx']
});
const stream2 = await exec2.start({ hijack: true });
```

### Cleanup
```javascript
await container.stop({ t: 5 });
await container.remove();
```

---

## 💾 Database Schema

### users
```sql
id, username, email, password, role, created_at
```

### sessions
```sql
session_id, user_id, task_id, container_id, status, created_at, expires_at
```

### progress
```sql
id, user_id, task_id, score, total, passed, completed_at
```

---

## ✅ Yangi Task Validator Qo'shish

### 1. Backend'da Validator Yoz
```javascript
// /tmp/backend-production.js

async function checkYourTask(container, task) {
  const results = [];

  // Check 1
  try {
    const exec = await container.exec({
      Cmd: ['your', 'command'],
      AttachStdout: true
    });
    const stream = await exec.start({ hijack: true });
    let output = '';
    stream.on('data', chunk => output += chunk.toString());
    await new Promise(resolve => stream.on('end', resolve));

    const success = output.includes('expected');
    results.push({
      name: '1. Your check description',
      ok: success,
      output: success ? '✓ Success' : '✗ Failed'
    });
  } catch (err) {
    results.push({
      name: '1. Your check description',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  // More checks...

  return results;
}
```

### 2. Backend'da Register Qil
```javascript
// /tmp/backend-production.js - check endpoint

if (task.id === 'nginx-health') {
  checkNginxTask(container, task).then(/* ... */);
} else if (task.id === 'your-task-id') {
  checkYourTask(container, task).then(/* ... */);
} else {
  // Generic response (not implemented)
}
```

### 3. Task Definition Qo'sh
```javascript
// /tmp/backend-production.js - taskDetails

const taskDetails = {
  'your-task-id': {
    id: 'your-task-id',
    title: 'Test X: Your Task Title',
    order: 7,
    difficulty: 2,
    unlocked: true,
    completed: false,
    points: 30,
    description: 'Your task description',
    hints: [
      'Hint 1',
      'Hint 2',
      'Hint 3'
    ]
  }
};
```

---

## 🐛 Debug Qilish

### Backend Logs
```bash
tail -f /tmp/backend-production.log
```

### Frontend Console
```javascript
// Browser DevTools → Console tab
console.log(sessionStore.sessionId);
console.log(taskStore.tasks);
```

### Container Holati
```bash
# Barcha containers
docker ps -a | grep devops-lab

# Container ichiga kirish
docker exec -it <container_id> bash

# Container logs
docker logs <container_id>
```

### PostgreSQL
```bash
psql -U devopslab -d devopslab

-- Sessions
SELECT * FROM sessions;

-- Progress
SELECT * FROM progress;

-- Users
SELECT * FROM users;
```

---

## 🔥 Tez-tez Uchraydigan Muammolar

### 1. Port band
```bash
lsof -ti:8080 | xargs -r kill -9  # Backend
lsof -ti:3000 | xargs -r kill -9  # Frontend
```

### 2. Container tozalanmayapti
```bash
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f
```

### 3. PostgreSQL ulanmayapti
```bash
# Start
brew services start postgresql@17

# Test
psql -U devopslab -d devopslab -c "SELECT 1"
```

### 4. Docker image yo'q
```bash
docker build -f Dockerfile.simple -t devopslab:latest .
```

### 5. Terminal ulanmayapti
- Backend ishlab turganini tekshir: `curl http://localhost:8080/api/tasks`
- Container mavjudligini tekshir: `docker ps | grep devops-lab`
- Session'ni restart qil: Stop → Start

---

## 📊 Monitoring

### Backend Status
```bash
# Process running?
ps aux | grep backend-production

# Port listening?
lsof -i:8080
```

### Container Stats
```bash
# Resource usage
docker stats $(docker ps | grep devops-lab | awk '{print $1}')
```

### Database Connections
```bash
psql -U devopslab -d devopslab -c "SELECT count(*) FROM pg_stat_activity;"
```

---

## 🎓 Keyingi Qadamlar

1. **To'liq Documentation O'qing:** `DOCUMENTATION.md`
2. **Codebase'ni O'rganing:** Backend → Frontend → Database
3. **Test Qiling:** Register → Start Session → Complete Task → Check
4. **Yangi Validator Qo'shing:** Qolgan 5 ta task uchun
5. **Security Yaxshilang:** Real JWT, password hashing, input validation
6. **Feature Qo'shing:** Hints, leaderboard, certificates

---

## 📚 Foydali Linklar

- **Full Documentation:** `DOCUMENTATION.md`
- **Docker API:** https://docs.docker.com/engine/api/
- **Vue 3 Guide:** https://vuejs.org/guide/
- **PostgreSQL Docs:** https://www.postgresql.org/docs/
- **xterm.js:** https://xtermjs.org/docs/

---

## ⚡ Tez Komandalar

```bash
# Hammasi ishga tushir
brew services start postgresql@17
cd /tmp && node backend-production.js &
cd ~/Documents/devops/devops-lab/frontend-vue && npm run dev

# Hammasi to'xtat
pkill -f backend-production
lsof -ti:3000 | xargs -r kill -9
brew services stop postgresql@17

# Tozalash
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f
psql -U devopslab -d devopslab -c "DELETE FROM sessions;"

# Rebuild
docker build -f Dockerfile.simple -t devopslab:latest .
cd frontend-vue && npm install
```

---

**Good luck! 🚀**

Savollar bo'lsa, `DOCUMENTATION.md` ga qarang.
