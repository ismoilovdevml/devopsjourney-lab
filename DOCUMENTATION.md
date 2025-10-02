# DevOps Lab v2.0 - Full Technical Documentation

## 📋 Table of Contents
1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Technology Stack](#technology-stack)
4. [System Requirements](#system-requirements)
5. [Installation & Setup](#installation--setup)
6. [Project Structure](#project-structure)
7. [Core Components](#core-components)
8. [Business Logic](#business-logic)
9. [Database Schema](#database-schema)
10. [API Documentation](#api-documentation)
11. [Frontend Architecture](#frontend-architecture)
12. [Docker Integration](#docker-integration)
13. [Session Management](#session-management)
14. [Task Validation System](#task-validation-system)
15. [Security Considerations](#security-considerations)
16. [Troubleshooting](#troubleshooting)
17. [Future Improvements](#future-improvements)

---

## 📖 Project Overview

**DevOps Lab v2.0** - Bu interaktiv DevOps/Linux o'rganish platformasi.

### Asosiy Maqsad
Studentlarga real Linux containerda amaliy mashqlar bajarish imkoniyatini berish:
- Nginx web server sozlash
- Docker boshqaruv
- Systemd service yaratish
- User va permission management
- UFW Firewall konfiguratsiya
- Cron jobs va backup avtomatlashtirish

### Asosiy Xususiyatlar
- ✅ Real Docker container'larda ishlatish
- ✅ WebSocket orqali real-time terminal
- ✅ Avtomatik validatsiya va scoring
- ✅ PostgreSQL'da progress tracking
- ✅ Bitta session'da barcha tasklar
- ✅ Auto-cleanup (1 soat TTL)
- ✅ ANSI terminal ranglari

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT BROWSER                        │
│                   (Vue.js + Vite + TypeScript)               │
│                    http://localhost:3000                     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ HTTP/REST + WebSocket
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                    BACKEND SERVER                            │
│              (Node.js + Native HTTP Server)                  │
│                   http://localhost:8080                      │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  REST API Endpoints                                  │   │
│  │  - /api/auth/login, /api/auth/register              │   │
│  │  - /api/tasks, /api/sessions                        │   │
│  │  - /api/check/:sessionId                            │   │
│  │  - /api/progress                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  WebSocket Handler                                   │   │
│  │  - /ws?sessionId=xxx                                │   │
│  │  - Bidirectional terminal I/O                       │   │
│  └─────────────────────────────────────────────────────┘   │
└──────┬────────────────────────────┬─────────────────────────┘
       │                            │
       │                            │
       ▼                            ▼
┌─────────────────┐      ┌──────────────────────┐
│   PostgreSQL    │      │   Docker Engine      │
│   Database      │      │   (dockerode SDK)    │
│                 │      │                      │
│ - users         │      │ ┌──────────────────┐ │
│ - sessions      │      │ │ Container 1      │ │
│ - progress      │      │ │ devopslab:latest │ │
│                 │      │ └──────────────────┘ │
└─────────────────┘      │ ┌──────────────────┐ │
                         │ │ Container 2      │ │
                         │ │ devopslab:latest │ │
                         │ └──────────────────┘ │
                         └──────────────────────┘
```

### Data Flow

#### 1. User Login
```
Browser → POST /api/auth/login → PostgreSQL (users table) → JWT Token → Browser
```

#### 2. Start Session
```
Browser → POST /api/sessions {taskId}
    → Docker: create container (devopslab:latest)
    → Docker: start container
    → PostgreSQL: save session record
    → Response: {sessionId, expiresAt, ttlSeconds}
    → Browser opens WebSocket connection
```

#### 3. Terminal Interaction
```
Browser (xterm.js) → WebSocket /ws?sessionId=xxx
    → Backend: container.exec('/bin/bash')
    → Docker: hijack stream
    → Bidirectional data flow:
        User input → WS → Docker stdin
        Docker stdout → WS → xterm.js
```

#### 4. Check Solution
```
Browser → POST /api/check/:sessionId {taskId}
    → Backend: container.exec(['which', 'nginx'])
    → Backend: container.exec(['test', '-f', '/var/www/html/index.html'])
    → Backend: container.exec(['cat', '/var/www/html/index.html'])
    → Backend: container.exec(['service', 'nginx', 'status'])
    → Calculate score
    → PostgreSQL: save progress
    → Response: {passed, score, total, results[]}
```

#### 5. Stop Session / Logout
```
Browser → DELETE /api/sessions/:sessionId
    → Docker: container.stop()
    → Docker: container.remove()
    → PostgreSQL: delete session record
    → Memory: clear sessions Map, containers Map
```

---

## 🛠️ Technology Stack

### Backend
- **Runtime:** Node.js v18+ (JavaScript ES6)
- **HTTP Server:** Native `http` module (no Express)
- **WebSocket:** Custom implementation (RFC 6455)
- **Docker SDK:** `dockerode` v4.x
- **Database:** `pg` (PostgreSQL client) v8.x
- **Port:** 8080

### Frontend
- **Framework:** Vue.js 3.4+ (Composition API)
- **Build Tool:** Vite 5.4+
- **Language:** TypeScript 5.x
- **State Management:** Pinia 2.x
- **HTTP Client:** Axios
- **Terminal:** xterm.js + xterm-addon-fit
- **UI:** TailwindCSS 3.x
- **Icons:** lucide-vue-next
- **Router:** Vue Router 4.x
- **Port:** 3000

### Database
- **PostgreSQL:** 17.x
- **Database Name:** devopslab
- **User:** devopslab
- **Password:** devopslab_secret

### Infrastructure
- **Docker:** 24.x+
- **Custom Image:** devopslab:latest (Ubuntu 22.04 + tools)

---

## 💻 System Requirements

### Development Environment
- **OS:** macOS, Linux, or Windows (with WSL2)
- **Node.js:** v18.0.0 or higher
- **npm:** v9.0.0 or higher
- **Docker Desktop:** 4.25.0 or higher
- **PostgreSQL:** 17.x or higher
- **RAM:** Minimum 8GB (16GB recommended)
- **Disk:** 10GB free space

### Network Ports
- `3000` - Frontend (Vue.js dev server)
- `8080` - Backend (Node.js HTTP server)
- `5432` - PostgreSQL database

---

## 📦 Installation & Setup

### 1. Clone Repository
```bash
cd /Users/macbook/Documents/devops
git clone <repository-url> devops-lab
cd devops-lab
```

### 2. Setup PostgreSQL Database
```bash
# Install PostgreSQL 17
brew install postgresql@17  # macOS
# or
sudo apt install postgresql-17  # Ubuntu

# Start PostgreSQL
brew services start postgresql@17  # macOS
# or
sudo systemctl start postgresql  # Linux

# Create database and user
psql postgres
```

```sql
CREATE DATABASE devopslab;
CREATE USER devopslab WITH PASSWORD 'devopslab_secret';
GRANT ALL PRIVILEGES ON DATABASE devopslab TO devopslab;
\q
```

### 3. Build Custom Docker Image
```bash
cd /Users/macbook/Documents/devops/devops-lab
docker build -f Dockerfile.simple -t devopslab:latest .
```

**Dockerfile.simple** includes:
- Ubuntu 22.04
- vim, nano, curl, wget
- nginx, systemctl, cron
- net-tools, iputils-ping
- tar, gzip
- TERM=xterm-256color

### 4. Setup Backend
```bash
# Backend is in /tmp/backend-production.js
cd /tmp
npm init -y
npm install dockerode pg

# Start backend
node backend-production.js
```

Backend will:
- Connect to PostgreSQL
- Create tables (users, sessions, progress)
- Start HTTP server on port 8080
- Enable CORS for frontend

### 5. Setup Frontend
```bash
cd /Users/macbook/Documents/devops/devops-lab/frontend-vue
npm install
npm run dev
```

Frontend will:
- Start Vite dev server on port 3000
- Hot reload enabled
- Proxy API requests to backend

### 6. Verify Installation
```bash
# Check backend
curl http://localhost:8080/api/tasks

# Check frontend
open http://localhost:3000

# Check Docker
docker images | grep devopslab

# Check PostgreSQL
psql -U devopslab -d devopslab -c "\dt"
```

---

## 📁 Project Structure

```
devops-lab/
├── frontend-vue/                      # Vue.js Frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── CheckResults.vue       # Validation natijalarini ko'rsatish
│   │   │   ├── ProgressSummary.vue    # Umumiy score modal
│   │   │   ├── TaskDetail.vue         # Task tavsifi
│   │   │   └── Terminal.vue           # xterm.js terminal component
│   │   ├── pages/
│   │   │   └── index.tsx              # Legacy landing page
│   │   ├── services/
│   │   │   └── api.ts                 # Axios HTTP client
│   │   ├── stores/
│   │   │   ├── auth.ts                # Pinia: authentication
│   │   │   ├── session.ts             # Pinia: session management
│   │   │   └── task.ts                # Pinia: task data
│   │   ├── types/
│   │   │   └── index.ts               # TypeScript type definitions
│   │   ├── views/
│   │   │   ├── LabView.vue            # Main lab interface
│   │   │   ├── Login.vue              # Login page
│   │   │   └── Register.vue           # Registration page
│   │   ├── App.vue                    # Root component
│   │   └── main.ts                    # Entry point
│   ├── package.json
│   ├── vite.config.ts                 # Vite configuration
│   ├── tailwind.config.js             # TailwindCSS config
│   └── tsconfig.json                  # TypeScript config
│
├── backend/                           # Legacy Go backend (ARCHIVED)
│   └── server.js                      # Old Node.js backend (NOT USED)
│
├── /tmp/backend-production.js         # 🔥 ACTIVE BACKEND (Production)
│
├── backend/tasks/                     # Task YAML definitions
│   ├── nginx-health/
│   │   └── task.yaml                  # Nginx web server task
│   ├── docker-basics/
│   │   └── task.yaml                  # Docker management task
│   ├── systemd-service/
│   │   └── task.yaml                  # Systemd service task
│   ├── user-management/
│   │   └── task.yaml                  # User permissions task
│   ├── firewall-config/
│   │   └── task.yaml                  # UFW firewall task
│   └── cron-backup/
│       └── task.yaml                  # Cron jobs task
│
├── Dockerfile.simple                  # Custom Docker image
├── docker-compose.yml                 # Docker Compose setup
├── README.md                          # Project overview (Uzbek)
├── QUICKSTART.md                      # Quick start guide (Uzbek)
└── DOCUMENTATION.md                   # This file
```

---

## 🧩 Core Components

### Backend Components (`/tmp/backend-production.js`)

#### 1. HTTP Server
```javascript
const server = http.createServer((req, res) => {
  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,DELETE,OPTIONS');

  // Route handling
  if (path === '/api/tasks') { /* ... */ }
  if (path === '/api/sessions') { /* ... */ }
  if (path === '/api/check/:id') { /* ... */ }
});

server.listen(8080);
```

#### 2. WebSocket Upgrade Handler
```javascript
server.on('upgrade', (request, socket, head) => {
  if (pathname === '/ws') {
    // WebSocket handshake
    const acceptKey = crypto.createHash('sha1')
      .update(request.headers['sec-websocket-key'] + MAGIC_STRING)
      .digest('base64');

    socket.write('HTTP/1.1 101 Switching Protocols\r\n...');

    // Attach to Docker container
    container.exec({
      Cmd: ['/bin/bash'],
      AttachStdin: true,
      AttachStdout: true,
      Tty: true
    }, (err, exec) => {
      exec.start({ hijack: true }, (err, stream) => {
        // Bidirectional streaming
        stream.on('data', chunk => sendWS(socket, chunk));
        socket.on('data', buffer => {
          const frame = parseWSFrame(buffer);
          stream.write(frame.payload);
        });
      });
    });
  }
});
```

#### 3. Docker Manager
```javascript
const docker = new Docker({ socketPath: '/var/run/docker.sock' });

async function createSessionContainer(taskId) {
  const container = await docker.createContainer({
    Image: 'devopslab:latest',
    name: `devops-lab-${sessionId}`,
    Tty: true,
    OpenStdin: true,
    Cmd: ['/bin/bash'],
    Env: ['TERM=xterm-256color'],
    HostConfig: {
      Memory: 512 * 1024 * 1024,    // 512MB RAM limit
      CpuShares: 512,                 // 50% CPU
      AutoRemove: false
    }
  });

  await container.start();
  return container;
}
```

#### 4. PostgreSQL Client
```javascript
const pgClient = new Client({
  host: 'localhost',
  port: 5432,
  database: 'devopslab',
  user: 'devopslab',
  password: 'devopslab_secret'
});

await pgClient.connect();
```

#### 5. Task Validator (Nginx Example)
```javascript
async function checkNginxTask(container, task) {
  const results = [];

  // Check 1: nginx installed
  const exec1 = await container.exec({
    Cmd: ['which', 'nginx'],
    AttachStdout: true
  });
  const stream1 = await exec1.start({ hijack: true });
  let output1 = '';
  stream1.on('data', chunk => output1 += chunk.toString());
  await new Promise(resolve => stream1.on('end', resolve));

  results.push({
    name: '1. nginx o\'rnatilgan',
    ok: output1.includes('/nginx'),
    output: output1.trim() || 'nginx not found'
  });

  // Check 2: index.html exists
  const exec2 = await container.exec({
    Cmd: ['test', '-f', '/var/www/html/index.html']
  });
  const stream2 = await exec2.start({ hijack: true });
  await new Promise(resolve => stream2.on('end', resolve));
  const info2 = await exec2.inspect();

  results.push({
    name: '2. index.html fayl yaratilgan',
    ok: info2.ExitCode === 0,
    output: info2.ExitCode === 0 ? '✓ File exists' : '✗ File not found'
  });

  // ... more checks

  return results;
}
```

#### 6. Session Cleanup
```javascript
async function cleanupSession(sessionId) {
  const container = containers.get(sessionId);
  if (container) {
    try {
      await container.stop({ t: 5 });      // 5 second timeout
      await container.remove();             // Remove container
      await pgClient.query(
        'DELETE FROM sessions WHERE session_id = $1',
        [sessionId]
      );
    } catch (err) {
      console.error('Cleanup error:', err.message);
    }
    containers.delete(sessionId);
    sessions.delete(sessionId);
  }
}

// Auto-cleanup after 1 hour
setTimeout(() => cleanupSession(sessionId), 3600000);
```

### Frontend Components

#### 1. LabView.vue (Main Interface)
**Mas'uliyati:**
- Task navigation (Previous/Next/Skip)
- Session lifecycle management
- Terminal container
- Check button handler

**Key Features:**
```vue
<template>
  <!-- Header: Timer, Progress button, Logout -->
  <!-- Sidebar: Task list, navigation, Start/Stop/Check buttons -->
  <!-- Main: Terminal component -->
  <!-- Modal: Progress summary -->
</template>

<script setup>
const currentTaskIndex = ref(0);
const currentTask = computed(() => taskStore.tasks[currentTaskIndex.value]);

// Navigation - faqat UI o'zgaradi, session saqlanadi
function handleNext() {
  if (currentTaskIndex.value < taskStore.tasks.length - 1) {
    currentTaskIndex.value++;
    taskStore.loadTask(taskStore.tasks[currentTaskIndex.value].id);
  }
}

// Check - hozirgi taskni tekshiradi
async function handleCheck() {
  await sessionStore.checkSolution(currentTask.value?.id);
}
</script>
```

#### 2. Terminal.vue (xterm.js Integration)
**Mas'uliyati:**
- WebSocket connection management
- Terminal rendering
- Keyboard input handling
- ANSI color support

```vue
<script setup>
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';

const terminal = new Terminal({
  cursorBlink: true,
  fontSize: 14,
  theme: {
    background: '#1e1e1e',
    foreground: '#d4d4d4',
    // 16 ANSI colors
  }
});

const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);

// WebSocket connection
const ws = new WebSocket(`ws://localhost:8080/ws?sessionId=${props.sessionId}`);
ws.binaryType = 'arraybuffer';

ws.onmessage = (event) => {
  terminal.write(new Uint8Array(event.data));
};

terminal.onData((data) => {
  ws.send(data);
});
</script>
```

#### 3. Pinia Stores

**auth.ts - Authentication**
```typescript
export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const token = ref<string | null>(localStorage.getItem('auth_token'));

  async function login(credentials: LoginRequest) {
    const response = await api.login(credentials);
    token.value = response.token;
    user.value = response.user;
    localStorage.setItem('auth_token', response.token);
    localStorage.setItem('username', response.user.username);
  }

  function logout() {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('username');
    user.value = null;
    token.value = null;
  }

  return { user, token, login, logout };
});
```

**session.ts - Session Management**
```typescript
export const useSessionStore = defineStore('session', () => {
  const sessionId = ref<string | null>(null);
  const timeRemaining = ref(0);

  async function createSession(taskId: string) {
    const response = await api.createSession(taskId);
    sessionId.value = response.sessionId;
    timeRemaining.value = response.ttlSeconds;
    startCountdown();
  }

  async function checkSolution(taskId?: string) {
    const result = await api.checkSolution(sessionId.value!, taskId);
    checkResult.value = result;
    return result;
  }

  async function deleteSession() {
    await api.deleteSession(sessionId.value!);
    sessionId.value = null;
    timeRemaining.value = 0;
  }

  return { sessionId, createSession, checkSolution, deleteSession };
});
```

**task.ts - Task Data**
```typescript
export const useTaskStore = defineStore('task', () => {
  const tasks = ref<Task[]>([]);
  const selectedTask = ref<Task | null>(null);

  async function loadTasks() {
    tasks.value = await api.getTasks();
  }

  async function loadTask(id: string) {
    selectedTask.value = await api.getTask(id);
  }

  return { tasks, selectedTask, loadTasks, loadTask };
});
```

---

## 🧠 Business Logic

### 1. Session Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    SESSION LIFECYCLE                         │
└─────────────────────────────────────────────────────────────┘

1. USER CLICKS "START SESSION"
   ↓
   POST /api/sessions { taskId: "nginx-health" }
   ↓
   Backend:
   - sessionId = "session-1-1759310269976"
   - Create Docker container (devopslab:latest)
   - Start container
   - Save to PostgreSQL sessions table
   - Save to memory: sessions.set(sessionId, {...})
   - Save to memory: containers.set(sessionId, containerObj)
   - Set auto-cleanup timer (1 hour)
   ↓
   Response: { sessionId, expiresAt, ttlSeconds: 3600 }
   ↓
   Frontend:
   - Save sessionId to Pinia store
   - Open WebSocket connection
   - Start countdown timer
   - Show terminal

2. USER NAVIGATES: NEXT → TASK 2
   ↓
   Frontend only:
   - currentTaskIndex++
   - Load task description
   - Container REMAINS RUNNING ✅
   - Session ID stays same ✅

3. USER CLICKS "CHECK"
   ↓
   POST /api/check/:sessionId { taskId: "docker-basics" }
   ↓
   Backend:
   - Get container from containers.get(sessionId)
   - Use taskId from request body (NOT session.taskId)
   - Run validation checks in container
   - Calculate score
   - Save progress to PostgreSQL
   ↓
   Response: { passed, score, total, results: [...] }

4. USER CLICKS "STOP SESSION"
   ↓
   DELETE /api/sessions/:sessionId
   ↓
   Backend cleanupSession():
   - container.stop({ t: 5 })
   - container.remove()
   - DELETE FROM sessions WHERE session_id = $1
   - sessions.delete(sessionId)
   - containers.delete(sessionId)
   - clearTimeout(autoCleanupTimer)
   ↓
   Frontend:
   - sessionId = null
   - Close WebSocket
   - Clear terminal

5. AUTO-CLEANUP (after 1 hour OR logout)
   ↓
   Same as step 4
```

### 2. Task Navigation Logic

**IMPORTANT:** Session o'zgarmaydi, faqat UI task o'zgaradi!

```javascript
// ❌ NOTO'G'RI (eski yondashuv)
function handleNext() {
  await deleteSession();           // Container o'chiriladi
  currentTaskIndex++;
  await createSession(newTaskId);  // Yangi container
}

// ✅ TO'G'RI (joriy yondashuv)
function handleNext() {
  currentTaskIndex++;                              // Faqat UI
  taskStore.loadTask(tasks[currentTaskIndex].id);  // Faqat description
  // Container va session o'zgarmaydi!
}
```

### 3. Check Validation Logic

```javascript
// Frontend: current task ID yuboriladi
await api.checkSolution(sessionId, currentTask.id);

// Backend: request body'dan taskId olinadi
const { taskId } = JSON.parse(body);
const task = taskDetails[taskId];  // Bu taskni tekshiramiz
const container = containers.get(sessionId);  // Session containeri

// Container'da task validatsiyasini bajaramiz
if (task.id === 'nginx-health') {
  const results = await checkNginxTask(container, task);
  // ...
} else if (task.id === 'docker-basics') {
  const results = await checkDockerTask(container, task);
  // ...
}
```

**Natija:** Bitta container'da turli tasklar tekshiriladi!

### 4. Progress Tracking

```sql
-- Har check muvaffaqiyatli bo'lganda:
INSERT INTO progress (user_id, task_id, score, total, passed)
VALUES (1, 'nginx-health', 25, 25, true);

-- Progress Summary query:
SELECT task_id,
       MAX(score) as score,
       MAX(total) as total,
       BOOL_OR(passed) as passed
FROM progress
WHERE user_id = $1
GROUP BY task_id;

-- Total score:
SELECT SUM(max_score) as total_score
FROM (
  SELECT task_id, MAX(score) as max_score
  FROM progress
  WHERE user_id = $1
  GROUP BY task_id
) subquery;
```

---

## 🗄️ Database Schema

### Users Table
```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(100) UNIQUE NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(50) DEFAULT 'student',
  created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

**Sample Data:**
```sql
INSERT INTO users (username, email, password, role)
VALUES ('student1', 'student1@example.com', 'hashed_password', 'student');
```

### Sessions Table
```sql
CREATE TABLE sessions (
  session_id VARCHAR(100) PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  task_id VARCHAR(100) NOT NULL,
  container_id VARCHAR(100) NOT NULL,
  status VARCHAR(50) DEFAULT 'active',
  created_at TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL
);

-- Indexes
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

**Sample Data:**
```sql
INSERT INTO sessions (session_id, user_id, task_id, container_id, expires_at)
VALUES (
  'session-1-1759310269976',
  1,
  'nginx-health',
  'abc123def456',
  NOW() + INTERVAL '1 hour'
);
```

### Progress Table
```sql
CREATE TABLE progress (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  task_id VARCHAR(100) NOT NULL,
  score INTEGER DEFAULT 0,
  total INTEGER DEFAULT 0,
  passed BOOLEAN DEFAULT FALSE,
  completed_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_progress_user_id ON progress(user_id);
CREATE INDEX idx_progress_task_id ON progress(task_id);
CREATE INDEX idx_progress_user_task ON progress(user_id, task_id);
```

**Sample Data:**
```sql
INSERT INTO progress (user_id, task_id, score, total, passed)
VALUES (1, 'nginx-health', 25, 25, true);

INSERT INTO progress (user_id, task_id, score, total, passed)
VALUES (1, 'docker-basics', 30, 35, false);
```

**Query Examples:**
```sql
-- User progress summary
SELECT
  task_id,
  MAX(score) as best_score,
  MAX(total) as total_points,
  BOOL_OR(passed) as ever_passed,
  COUNT(*) as attempts
FROM progress
WHERE user_id = 1
GROUP BY task_id
ORDER BY MAX(completed_at) DESC;

-- Total score
SELECT SUM(best_score) as total
FROM (
  SELECT MAX(score) as best_score
  FROM progress
  WHERE user_id = 1
  GROUP BY task_id
) t;
```

---

## 🔌 API Documentation

### Base URL
```
http://localhost:8080/api
```

### Authentication

#### POST `/auth/register`
**Request:**
```json
{
  "username": "student1",
  "email": "student1@example.com",
  "password": "password123"
}
```

**Response:** `201 Created`
```json
{
  "token": "jwt-1759310269976",
  "user": {
    "id": 1,
    "username": "student1",
    "email": "student1@example.com",
    "role": "student"
  }
}
```

#### POST `/auth/login`
**Request:**
```json
{
  "username": "student1",
  "password": "password123"
}
```

**Response:** `200 OK`
```json
{
  "token": "jwt-1759310269976",
  "user": {
    "id": 1,
    "username": "student1",
    "email": "student1@example.com",
    "role": "student"
  }
}
```

**Error:** `404 Not Found`
```json
{
  "error": "User not found",
  "message": "Iltimos avval ro'yxatdan o'ting."
}
```

### Tasks

#### GET `/tasks`
**Response:** `200 OK`
```json
[
  {
    "id": "nginx-health",
    "title": "Test 1: Nginx Web Server",
    "order": 1,
    "difficulty": 1,
    "points": 25,
    "unlocked": true,
    "completed": false
  },
  {
    "id": "docker-basics",
    "title": "Test 2: Docker Management",
    "order": 2,
    "difficulty": 2,
    "points": 35,
    "unlocked": true,
    "completed": false
  }
]
```

#### GET `/tasks/:id`
**Response:** `200 OK`
```json
{
  "id": "nginx-health",
  "title": "Test 1: Nginx Web Server",
  "description": "Nginx web server o'rnatish va sozlash.",
  "hints": [
    "apt-get update",
    "nginx o'rnatish",
    "index.html yaratish",
    "/var/www/html/ papka",
    "nginx start"
  ],
  "difficulty": 1,
  "points": 25,
  "order": 1
}
```

### Sessions

#### POST `/sessions`
**Headers:**
```
Authorization: Bearer jwt-1759310269976
x-username: student1
```

**Request:**
```json
{
  "taskId": "nginx-health"
}
```

**Response:** `201 Created`
```json
{
  "sessionId": "session-1-1759310269976",
  "taskId": "nginx-health",
  "containerId": "abc123def456",
  "status": "active",
  "expiresAt": "2024-01-01T13:00:00.000Z",
  "ttlSeconds": 3600
}
```

**Process:**
1. Extract userId from x-username header
2. Create Docker container (devopslab:latest)
3. Start container
4. Save session to PostgreSQL
5. Set 1-hour auto-cleanup timer
6. Return session details

#### DELETE `/sessions/:sessionId`
**Response:** `200 OK`
```json
{
  "message": "Session deleted"
}
```

**Process:**
1. Get container from memory
2. Stop container (5 sec timeout)
3. Remove container
4. Delete from PostgreSQL
5. Clear from memory maps

### Validation

#### POST `/check/:sessionId`
**Request:**
```json
{
  "taskId": "nginx-health"
}
```

**Response:** `200 OK`
```json
{
  "passed": true,
  "score": 25,
  "total": 25,
  "results": [
    {
      "name": "1. nginx o'rnatilgan",
      "ok": true,
      "output": "✓ /usr/sbin/nginx"
    },
    {
      "name": "2. index.html fayl yaratilgan",
      "ok": true,
      "output": "✓ /var/www/html/index.html"
    },
    {
      "name": "3. /var/www/html/ papkada joylashtirilgan",
      "ok": true,
      "output": "✓ Content: <!DOCTYPE html>..."
    },
    {
      "name": "4. nginx service ishlab turibdi",
      "ok": true,
      "output": "✓ nginx is running"
    }
  ]
}
```

**Validation Process:**
1. Get sessionId from URL
2. Get taskId from request body
3. Get container from containers Map
4. Execute validation commands in container
5. Calculate score (passedCount / totalChecks * points)
6. Save progress to PostgreSQL
7. Return results

### Progress

#### GET `/progress`
**Headers:**
```
Authorization: Bearer jwt-1759310269976
```

**Response:** `200 OK`
```json
{
  "progress": [
    {
      "task_id": "nginx-health",
      "score": 25,
      "total": 25,
      "passed": true,
      "completed_at": "2024-01-01T12:30:00.000Z"
    },
    {
      "task_id": "docker-basics",
      "score": 30,
      "total": 35,
      "passed": false,
      "completed_at": "2024-01-01T12:45:00.000Z"
    }
  ],
  "totalScore": 55,
  "totalPossible": 210,
  "percentage": 26
}
```

### WebSocket

#### WS `/ws?sessionId=xxx`
**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?sessionId=session-1-1759310269976');
ws.binaryType = 'arraybuffer';
```

**Client → Server (User Input):**
```
WebSocket Frame:
  FIN: 1
  Opcode: 0x01 (Text)
  Mask: true
  Payload: "ls -la\n"
```

**Server → Client (Terminal Output):**
```
WebSocket Frame:
  FIN: 1
  Opcode: 0x01 (Text)
  Mask: false
  Payload: "total 64\ndrwxr-xr-x 1 root root 4096..."
```

**Connection Flow:**
1. Client sends WebSocket handshake
2. Server calculates Sec-WebSocket-Accept key
3. Server upgrades connection to WebSocket
4. Server executes `/bin/bash` in container
5. Server hijacks Docker exec stream
6. Bidirectional data flow established:
   - User types → WS → Docker stdin
   - Docker stdout → WS → xterm.js

---

## 🎨 Frontend Architecture

### Vue Router Setup
```typescript
// router/index.ts
const routes = [
  { path: '/', redirect: '/login' },
  { path: '/login', component: Login },
  { path: '/register', component: Register },
  {
    path: '/lab',
    component: LabView,
    meta: { requiresAuth: true }
  }
];

router.beforeEach((to, from, next) => {
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login');
  } else {
    next();
  }
});
```

### Axios Interceptors
```typescript
// services/api.ts
this.client.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  const username = localStorage.getItem('username');

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  if (username) {
    config.headers['x-username'] = username;
  }

  return config;
});

this.client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Auto logout on 401
      authStore.logout();
      router.push('/login');
    }
    return Promise.reject(error);
  }
);
```

### Component Communication
```
App.vue
  ├── Router
  │   ├── Login.vue
  │   │   └── uses authStore.login()
  │   ├── Register.vue
  │   │   └── uses authStore.register()
  │   └── LabView.vue (Main)
  │       ├── Header
  │       │   ├── Timer (sessionStore.timeRemaining)
  │       │   ├── Progress button → ProgressSummary.vue
  │       │   └── Logout (authStore.logout)
  │       ├── Sidebar
  │       │   ├── Task Navigation (Previous/Next/Skip)
  │       │   ├── TaskDetail.vue (taskStore.selectedTask)
  │       │   ├── Start/Stop/Check buttons
  │       │   └── CheckResults.vue (sessionStore.checkResult)
  │       └── Terminal Area
  │           └── Terminal.vue (props: sessionId)
  │               ├── xterm.js
  │               └── WebSocket connection
```

### State Management Flow
```
User Action → Component Method → Pinia Store Action → API Call → Backend
                                                                     ↓
User sees result ← Component updates ← Store updates ← Response ←──┘
```

Example:
```
User clicks "Check"
  ↓
LabView.handleCheck()
  ↓
sessionStore.checkSolution(taskId)
  ↓
api.checkSolution(sessionId, taskId)
  ↓
POST /api/check/:sessionId { taskId }
  ↓
Backend validates in container
  ↓
Response: { passed, score, results }
  ↓
sessionStore.checkResult = response
  ↓
CheckResults.vue updates (reactive)
  ↓
User sees validation results
```

---

## 🐳 Docker Integration

### Custom Image: devopslab:latest

**Dockerfile.simple:**
```dockerfile
FROM ubuntu:22.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color

# Install required packages
RUN apt-get update && apt-get install -y \
    vim \
    nano \
    curl \
    wget \
    sudo \
    nginx \
    systemctl \
    cron \
    net-tools \
    iputils-ping \
    tar \
    gzip \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root
CMD ["/bin/bash"]
```

**Build:**
```bash
docker build -f Dockerfile.simple -t devopslab:latest .
```

**Image size:** ~300MB

### Container Lifecycle

#### 1. Create Container
```javascript
const container = await docker.createContainer({
  Image: 'devopslab:latest',
  name: `devops-lab-${sessionId}`,
  Tty: true,
  OpenStdin: true,
  Cmd: ['/bin/bash'],
  Env: [
    'DEBIAN_FRONTEND=noninteractive',
    'TERM=xterm-256color'
  ],
  HostConfig: {
    Memory: 512 * 1024 * 1024,  // 512MB RAM
    CpuShares: 512,              // 50% CPU (out of 1024)
    AutoRemove: false            // Manual cleanup
  }
});
```

#### 2. Start Container
```javascript
await container.start();
console.log('Container ID:', container.id.substring(0, 12));
```

#### 3. Execute Commands
```javascript
// Interactive bash
const exec = await container.exec({
  Cmd: ['/bin/bash'],
  AttachStdin: true,
  AttachStdout: true,
  AttachStderr: true,
  Tty: true
});

const stream = await exec.start({
  Tty: true,
  stdin: true,
  hijack: true  // Bidirectional stream
});

// Validation command
const exec2 = await container.exec({
  Cmd: ['which', 'nginx'],
  AttachStdout: true,
  AttachStderr: true
});

const stream2 = await exec2.start({ hijack: true });
let output = '';
stream2.on('data', chunk => output += chunk.toString());
await new Promise(resolve => stream2.on('end', resolve));

console.log('Output:', output);
```

#### 4. Stop Container
```javascript
await container.stop({ t: 5 }); // 5 second timeout
```

#### 5. Remove Container
```javascript
await container.remove();
```

### Resource Limits

| Resource | Limit | Purpose |
|----------|-------|---------|
| Memory | 512MB | Prevent memory exhaustion |
| CPU | 50% (512 shares) | Fair CPU distribution |
| Timeout | 1 hour | Auto-cleanup inactive sessions |
| Stop timeout | 5 seconds | Graceful shutdown |

### Container Naming Convention
```
devops-lab-session-{counter}-{timestamp}

Examples:
- devops-lab-session-1-1759310269976
- devops-lab-session-2-1759310345123
```

### Cleanup Strategies

**1. Manual Cleanup (User triggered):**
```javascript
// User clicks "Stop Session" or "Logout"
await sessionStore.deleteSession();
  → DELETE /api/sessions/:sessionId
    → cleanupSession(sessionId)
```

**2. Auto-cleanup (1 hour TTL):**
```javascript
setTimeout(async () => {
  await cleanupSession(sessionId);
}, 3600000); // 1 hour
```

**3. Forced cleanup (if backend restarts):**
```bash
# Find orphaned containers
docker ps -a | grep devops-lab

# Remove all
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f
```

---

## 🔐 Session Management

### In-Memory Storage
```javascript
const sessions = new Map();      // sessionId → session object
const containers = new Map();    // sessionId → Docker container object

// Session object structure:
{
  sessionId: 'session-1-1759310269976',
  taskId: 'nginx-health',
  containerId: 'abc123def456',
  userId: 1,
  status: 'active',
  createdAt: '2024-01-01T12:00:00.000Z',
  expiresAt: '2024-01-01T13:00:00.000Z',
  ttlSeconds: 3600
}
```

### Session ID Generation
```javascript
let sessionCounter = 0;

function generateSessionId() {
  sessionCounter++;
  return `session-${sessionCounter}-${Date.now()}`;
}

// Examples:
// session-1-1759310269976
// session-2-1759310345123
```

### Timer Management
```javascript
// Frontend countdown (visual only)
function startCountdown() {
  const interval = setInterval(() => {
    if (timeRemaining.value > 0) {
      timeRemaining.value--;
    } else {
      clearInterval(interval);
      clearSession();
    }
  }, 1000);
}

// Backend auto-cleanup (actual cleanup)
const cleanupTimers = new Map(); // sessionId → setTimeout ID

function setCleanupTimer(sessionId) {
  const timerId = setTimeout(async () => {
    console.log('⏰ Auto-cleaning session:', sessionId);
    await cleanupSession(sessionId);
  }, 3600000); // 1 hour

  cleanupTimers.set(sessionId, timerId);
}

function cancelCleanupTimer(sessionId) {
  const timerId = cleanupTimers.get(sessionId);
  if (timerId) {
    clearTimeout(timerId);
    cleanupTimers.delete(sessionId);
  }
}
```

### Session Recovery

**Problem:** Backend restarts lose in-memory sessions.

**Solution 1 (Current):** Sessions are lost, users must restart.

**Solution 2 (Future):** Load active sessions from PostgreSQL on startup:
```javascript
async function recoverSessions() {
  const result = await pgClient.query(`
    SELECT * FROM sessions
    WHERE status = 'active'
    AND expires_at > NOW()
  `);

  for (const row of result.rows) {
    try {
      const container = docker.getContainer(row.container_id);
      await container.inspect(); // Check if still exists

      sessions.set(row.session_id, {
        sessionId: row.session_id,
        taskId: row.task_id,
        containerId: row.container_id,
        userId: row.user_id,
        status: 'active',
        expiresAt: row.expires_at
      });

      containers.set(row.session_id, container);
      setCleanupTimer(row.session_id);
    } catch (err) {
      // Container doesn't exist, clean up DB record
      await pgClient.query(
        'DELETE FROM sessions WHERE session_id = $1',
        [row.session_id]
      );
    }
  }
}

// Call on startup
pgClient.connect().then(() => {
  initDatabase();
  recoverSessions();
});
```

---

## ✅ Task Validation System

### Validation Architecture

```
Frontend                Backend                    Docker Container
   │                       │                              │
   │ POST /api/check       │                              │
   │ {taskId: "nginx"}     │                              │
   ├──────────────────────→│                              │
   │                       │ container.exec(['which','nginx'])
   │                       ├─────────────────────────────→│
   │                       │                              │ /usr/sbin/nginx
   │                       │←─────────────────────────────┤
   │                       │                              │
   │                       │ container.exec(['test','-f','/var/www/html/index.html'])
   │                       ├─────────────────────────────→│
   │                       │                              │ exit code: 0
   │                       │←─────────────────────────────┤
   │                       │                              │
   │                       │ Calculate score              │
   │                       │ Save to PostgreSQL           │
   │                       │                              │
   │ {passed, score, results[]}                          │
   │←──────────────────────┤                              │
```

### Nginx Task Validator

```javascript
async function checkNginxTask(container, task) {
  const results = [];

  // Check 1: nginx o'rnatilgan
  try {
    const exec1 = await container.exec({
      Cmd: ['which', 'nginx'],
      AttachStdout: true,
      AttachStderr: true
    });
    const stream1 = await exec1.start({ hijack: true, stdin: false });
    let output1 = '';
    stream1.on('data', chunk => output1 += chunk.toString());
    await new Promise(resolve => stream1.on('end', resolve));

    const hasNginx = output1.includes('/nginx');
    results.push({
      name: '1. nginx o\'rnatilgan',
      ok: hasNginx,
      output: hasNginx ? '✓ /usr/sbin/nginx' : '✗ nginx not found'
    });
  } catch (err) {
    results.push({
      name: '1. nginx o\'rnatilgan',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  // Check 2: index.html fayl yaratilgan
  try {
    const exec2 = await container.exec({
      Cmd: ['test', '-f', '/var/www/html/index.html'],
      AttachStdout: true
    });
    const stream2 = await exec2.start({ hijack: true });
    await new Promise(resolve => stream2.on('end', resolve));
    const info2 = await exec2.inspect();

    const hasFile = info2.ExitCode === 0;
    results.push({
      name: '2. index.html fayl yaratilgan',
      ok: hasFile,
      output: hasFile ? '✓ /var/www/html/index.html' : '✗ File not found'
    });
  } catch (err) {
    results.push({
      name: '2. index.html fayl yaratilgan',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  // Check 3: /var/www/html/ papkada joylashtirilgan
  try {
    const exec3 = await container.exec({
      Cmd: ['cat', '/var/www/html/index.html'],
      AttachStdout: true
    });
    const stream3 = await exec3.start({ hijack: true });
    let content = '';
    stream3.on('data', chunk => content += chunk.toString());
    await new Promise(resolve => stream3.on('end', resolve));

    const hasContent = content.length > 0;
    results.push({
      name: '3. /var/www/html/ papkada joylashtirilgan',
      ok: hasContent,
      output: hasContent
        ? `✓ Content: ${content.substring(0, 100)}...`
        : '✗ File is empty'
    });
  } catch (err) {
    results.push({
      name: '3. /var/www/html/ papkada joylashtirilgan',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  // Check 4: nginx service ishlab turibdi
  try {
    const exec4 = await container.exec({
      Cmd: ['service', 'nginx', 'status'],
      AttachStdout: true,
      AttachStderr: true
    });
    const stream4 = await exec4.start({ hijack: true });
    let status = '';
    stream4.on('data', chunk => status += chunk.toString());
    await new Promise(resolve => stream4.on('end', resolve));

    const isRunning = status.includes('active') || status.includes('running');
    results.push({
      name: '4. nginx service ishlab turibdi',
      ok: isRunning,
      output: isRunning ? '✓ nginx is running' : '✗ nginx is not running'
    });
  } catch (err) {
    results.push({
      name: '4. nginx service ishlab turibdi',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  return results;
}
```

### Scoring Algorithm

```javascript
const passedCount = results.filter(r => r.ok).length;
const totalChecks = results.length;
const score = Math.floor((passedCount / totalChecks) * task.points);

// Example:
// passedCount = 3
// totalChecks = 4
// task.points = 25
// score = Math.floor((3/4) * 25) = Math.floor(18.75) = 18

const passed = passedCount === totalChecks; // true if all checks pass
```

### Adding New Task Validators

**Template:**
```javascript
async function check{TaskName}Task(container, task) {
  const results = [];

  // Check 1: Description
  try {
    const exec = await container.exec({
      Cmd: ['your', 'command', 'here'],
      AttachStdout: true,
      AttachStderr: true
    });
    const stream = await exec.start({ hijack: true });
    let output = '';
    stream.on('data', chunk => output += chunk.toString());
    await new Promise(resolve => stream.on('end', resolve));

    const success = /* your validation logic */;
    results.push({
      name: '1. Check description',
      ok: success,
      output: success ? '✓ Success message' : '✗ Failure message'
    });
  } catch (err) {
    results.push({
      name: '1. Check description',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  // More checks...

  return results;
}
```

**Register in backend:**
```javascript
if (task.id === 'nginx-health') {
  checkNginxTask(container, task).then(/* ... */);
} else if (task.id === 'your-new-task') {
  checkYourNewTask(container, task).then(/* ... */);
}
```

---

## 🔒 Security Considerations

### Current Implementation

⚠️ **WARNING:** This is a LEARNING PLATFORM, NOT production-ready!

**Known Security Issues:**

1. **No Real Authentication**
   - JWT tokens are fake (`jwt-${timestamp}`)
   - No password hashing (plain text in DB)
   - No session validation

2. **Container Security**
   - Containers run as root
   - No network isolation
   - No resource quotas beyond RAM/CPU
   - Shared Docker socket access

3. **API Security**
   - No rate limiting
   - No input validation
   - CORS wide open (`*`)
   - No HTTPS

4. **Database Security**
   - Hardcoded credentials
   - No connection pooling
   - No prepared statement parameterization in some queries

### Recommended Production Fixes

#### 1. Authentication & Authorization
```javascript
// Install
npm install bcrypt jsonwebtoken

// Hash passwords
const bcrypt = require('bcrypt');
const hashedPassword = await bcrypt.hash(password, 10);

// Verify password
const isValid = await bcrypt.compare(password, user.password);

// Generate real JWT
const jwt = require('jsonwebtoken');
const token = jwt.sign(
  { userId: user.id, username: user.username },
  process.env.JWT_SECRET,
  { expiresIn: '24h' }
);

// Verify JWT
const decoded = jwt.verify(token, process.env.JWT_SECRET);
```

#### 2. Environment Variables
```bash
# .env file
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=devopslab
DATABASE_USER=devopslab
DATABASE_PASSWORD=super_secret_password
JWT_SECRET=random_secret_key_here
DOCKER_SOCKET=/var/run/docker.sock
```

```javascript
// Load with dotenv
require('dotenv').config();

const pgClient = new Client({
  host: process.env.DATABASE_HOST,
  port: process.env.DATABASE_PORT,
  database: process.env.DATABASE_NAME,
  user: process.env.DATABASE_USER,
  password: process.env.DATABASE_PASSWORD
});
```

#### 3. Input Validation
```javascript
// Install
npm install joi

const Joi = require('joi');

const loginSchema = Joi.object({
  username: Joi.string().alphanum().min(3).max(30).required(),
  password: Joi.string().min(6).required()
});

const { error, value } = loginSchema.validate(req.body);
if (error) {
  return res.status(400).json({ error: error.details[0].message });
}
```

#### 4. Rate Limiting
```javascript
// Install
npm install express-rate-limit

const rateLimit = require('express-rate-limit');

const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
});

app.use('/api/', limiter);
```

#### 5. Container Security
```javascript
const container = await docker.createContainer({
  Image: 'devopslab:latest',
  User: '1000:1000',  // Non-root user
  HostConfig: {
    Memory: 512 * 1024 * 1024,
    CpuShares: 512,
    NetworkMode: 'none',  // No network access
    ReadonlyRootfs: false,
    CapDrop: ['ALL'],
    CapAdd: ['CHOWN', 'SETUID', 'SETGID'],
    SecurityOpt: ['no-new-privileges']
  }
});
```

#### 6. HTTPS/TLS
```javascript
const https = require('https');
const fs = require('fs');

const options = {
  key: fs.readFileSync('privkey.pem'),
  cert: fs.readFileSync('cert.pem')
};

https.createServer(options, app).listen(443);
```

---

## 🐛 Troubleshooting

### Common Issues

#### 1. Backend won't start - Port 8080 already in use
```bash
# Find process
lsof -ti:8080

# Kill process
lsof -ti:8080 | xargs -r kill -9

# Restart backend
cd /tmp && node backend-production.js
```

#### 2. PostgreSQL connection error
```bash
# Check PostgreSQL is running
brew services list | grep postgresql  # macOS
sudo systemctl status postgresql      # Linux

# Start PostgreSQL
brew services start postgresql@17     # macOS
sudo systemctl start postgresql       # Linux

# Test connection
psql -U devopslab -d devopslab -c "SELECT 1"
```

#### 3. Docker containers not cleaning up
```bash
# List all devops-lab containers
docker ps -a | grep devops-lab

# Remove all manually
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f

# Check backend logs
tail -f /tmp/backend-production.log
```

#### 4. Terminal not connecting
**Symptoms:** WebSocket fails, terminal shows "Connecting..."

**Solutions:**
```bash
# 1. Check backend is running
curl http://localhost:8080/api/tasks

# 2. Check WebSocket endpoint
# Open browser console, should see:
# "Connecting to WebSocket: ws://localhost:8080/ws?sessionId=xxx"

# 3. Check container exists
docker ps | grep devops-lab

# 4. Restart session
# Click "Stop Session" → "Start Session"
```

#### 5. Frontend build errors
```bash
# Clear node_modules and reinstall
cd frontend-vue
rm -rf node_modules package-lock.json
npm install

# Clear Vite cache
rm -rf .vite

# Restart dev server
npm run dev
```

#### 6. Custom Docker image not found
```bash
# Check if image exists
docker images | grep devopslab

# Rebuild image
docker build -f Dockerfile.simple -t devopslab:latest .

# Verify build
docker run -it devopslab:latest bash
```

#### 7. Session expires too quickly
**Change TTL:**
```javascript
// Backend: /tmp/backend-production.js
const expiresAt = new Date(Date.now() + 7200000); // 2 hours instead of 1
```

#### 8. Check results always "Not yet implemented"
**Reason:** Validator not implemented for that task.

**Solution:**
```javascript
// Add validator in backend
if (task.id === 'your-task-id') {
  checkYourTask(container, task).then(/* ... */);
}
```

### Debug Mode

**Enable verbose logging:**
```javascript
// Backend
console.log('🔍 DEBUG:', variable);

// Frontend
console.log('🎨 DEBUG:', variable);
```

**Watch logs:**
```bash
# Backend
tail -f /tmp/backend-production.log

# Frontend
# Open browser DevTools → Console tab

# PostgreSQL queries
# Add to pgClient:
pgClient.on('query', (query) => {
  console.log('📊 SQL:', query.text);
});
```

---

## 🚀 Future Improvements

### Planned Features

#### 1. Multi-Task Validators
Currently only nginx task has real validation. Implement:
- `checkDockerTask()` - Verify Docker commands
- `checkSystemdTask()` - Check systemd service
- `checkUserManagementTask()` - Validate users/permissions
- `checkFirewallTask()` - Test UFW rules
- `checkCronTask()` - Verify cron jobs

#### 2. Real-time Collaboration
Allow instructors to join student sessions:
```javascript
// Instructor watches student terminal
GET /api/sessions/:sessionId/watch
→ WebSocket with read-only access
```

#### 3. Session Recording
Record terminal sessions for review:
```javascript
// Save all terminal I/O to database
const recording = [];
stream.on('data', chunk => {
  recording.push({
    timestamp: Date.now(),
    type: 'output',
    data: chunk.toString()
  });
});

// Save on session end
await pgClient.query(
  'INSERT INTO session_recordings (session_id, recording) VALUES ($1, $2)',
  [sessionId, JSON.stringify(recording)]
);
```

#### 4. Hints System
Progressive hints if user is stuck:
```javascript
// After 5 minutes without progress
if (timeElapsed > 300 && checkAttempts === 0) {
  showHint(task.hints[0]);
}
```

#### 5. Leaderboard
```sql
CREATE TABLE leaderboard (
  user_id INTEGER,
  total_score INTEGER,
  tasks_completed INTEGER,
  average_time INTEGER,
  rank INTEGER
);

-- Update after each task completion
```

#### 6. Certificate Generation
```javascript
// When user completes all tasks with >80% score
async function generateCertificate(userId) {
  const totalScore = await getTotalScore(userId);
  if (totalScore / 210 >= 0.8) {
    // Generate PDF certificate
    const pdf = await generatePDF({
      name: user.username,
      score: totalScore,
      date: new Date(),
      certificateId: uuid()
    });
    return pdf;
  }
}
```

#### 7. Advanced Monitoring
```javascript
// Track container resource usage
setInterval(async () => {
  const stats = await container.stats({ stream: false });
  await pgClient.query(
    'INSERT INTO container_stats (session_id, cpu, memory, timestamp) VALUES ($1, $2, $3, NOW())',
    [sessionId, stats.cpu_percent, stats.memory_usage]
  );
}, 30000); // Every 30 seconds
```

#### 8. Task Templates
Allow instructors to create custom tasks via UI:
```javascript
// Task schema
{
  id: 'custom-task-1',
  title: 'Custom Task',
  description: '...',
  hints: ['...'],
  checks: [
    {
      type: 'file_exists',
      path: '/etc/myconfig.conf'
    },
    {
      type: 'command_output',
      command: ['systemctl', 'status', 'myservice'],
      contains: 'active (running)'
    },
    {
      type: 'port_listening',
      port: 8080
    }
  ]
}
```

#### 9. Mobile App
React Native or Flutter mobile app for learning on-the-go.

#### 10. Integrations
- **GitHub:** Auto-commit solutions
- **Slack:** Notifications on task completion
- **Zoom:** Video help sessions
- **Google Classroom:** Grade sync

---

## 📚 Additional Resources

### Documentation Links
- **Docker API:** https://docs.docker.com/engine/api/
- **dockerode:** https://github.com/apocas/dockerode
- **PostgreSQL:** https://www.postgresql.org/docs/
- **Vue 3:** https://vuejs.org/guide/introduction.html
- **xterm.js:** https://xtermjs.org/docs/
- **WebSocket RFC:** https://datatracker.ietf.org/doc/html/rfc6455

### Learning Resources
- Linux commands: https://linuxcommand.org/
- DevOps roadmap: https://roadmap.sh/devops
- Docker tutorial: https://docker-curriculum.com/
- Nginx guide: https://nginx.org/en/docs/

---

## 📝 Changelog

### Version 2.0 (Current)
- ✅ Migrated from Go to Node.js backend
- ✅ Migrated from React to Vue.js frontend
- ✅ Real Docker container integration
- ✅ WebSocket terminal (custom implementation)
- ✅ PostgreSQL database
- ✅ Progress tracking
- ✅ Single session for all tasks
- ✅ Auto-cleanup system
- ✅ ANSI terminal colors
- ✅ Nginx task validator

### Version 1.0 (Archived)
- Go backend with Echo framework
- React frontend with Next.js
- Mock terminal (no real containers)
- In-memory task data
- No validation system

---

## 👥 Credits

**Developer:** Claude Code + Human Collaboration
**Project Type:** Educational Platform
**License:** MIT
**Year:** 2025

---

## 📧 Support

For questions or issues:
1. Check this documentation
2. Review troubleshooting section
3. Check backend logs: `tail -f /tmp/backend-production.log`
4. Check browser console for frontend errors
5. Inspect Docker containers: `docker ps -a`

---

**End of Documentation**

Generated: 2025-01-01
Backend: `/tmp/backend-production.js`
Frontend: `/Users/macbook/Documents/devops/devops-lab/frontend-vue`
Database: PostgreSQL 17 (devopslab)
