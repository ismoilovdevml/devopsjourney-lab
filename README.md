# 🚀 DevOps Lab v2.0 - Interactive Learning Platform

**Real Docker container'larda DevOps/Linux o'rganish platformasi**

> ⚠️ **Production-ready emas!** Bu ta'lim platformasi - o'quv maqsadida ishlatiladi.

---

## ✨ Asosiy Xususiyatlar

- 🐳 **Real Docker Containers** - Har bir session uchun alohida container
- 💻 **Interactive Terminal** - Browser'da to'liq Linux terminal (xterm.js)
- ✅ **Auto-validation** - Yechimlarni avtomatik tekshirish
- 📊 **Progress Tracking** - PostgreSQL'da progress saqlash
- 🔄 **Single Session** - Barcha tasklar bitta container'da
- ⏱️ **Auto-cleanup** - 1 soatdan keyin avtomatik tozalash
- 🎨 **ANSI Colors** - Terminal ranglari qo'llab-quvvatlaydi
- 📈 **Score System** - Har bir task uchun ball to'plash

---

## 🏗️ Arxitektura

```
┌─────────────────────────────────────────────────────────┐
│               User Browser (Vue.js)                     │
│                 http://localhost:3000                   │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP/REST + WebSocket
┌────────────────────▼────────────────────────────────────┐
│           Node.js Backend (Native HTTP)                 │
│                 http://localhost:8080                   │
│    /tmp/backend-production.js                          │
└──────────┬─────────────────────┬────────────────────────┘
           │                     │
           ▼                     ▼
┌──────────────────┐   ┌─────────────────────────┐
│   PostgreSQL 17  │   │   Docker Engine         │
│   Database       │   │   (dockerode SDK)       │
│                  │   │                         │
│   - users        │   │   ┌─────────────────┐  │
│   - sessions     │   │   │ Container       │  │
│   - progress     │   │   │ devopslab:latest│  │
└──────────────────┘   │   └─────────────────┘  │
                       └─────────────────────────┘
```

---

## 🛠️ Texnologiyalar

### Backend
- **Runtime:** Node.js 18+
- **Server:** Native `http` module (Express yo'q)
- **WebSocket:** Custom RFC 6455 implementation
- **Docker:** `dockerode` v4.x
- **Database:** PostgreSQL 17 + `pg` driver
- **Port:** 8080

### Frontend
- **Framework:** Vue.js 3.4+ (Composition API)
- **Language:** TypeScript 5.x
- **Build:** Vite 5.4+
- **State:** Pinia 2.x
- **Terminal:** xterm.js + xterm-addon-fit
- **Styling:** TailwindCSS 3.x
- **Icons:** lucide-vue-next
- **Port:** 3000

### Infrastructure
- **Database:** PostgreSQL 17
- **Containers:** Docker 24+
- **Image:** Custom `devopslab:latest` (Ubuntu 22.04)

---

## 📦 O'rnatish

### 1. Requirements
- Node.js 18+
- Docker Desktop
- PostgreSQL 17
- 8GB+ RAM

### 2. Database Setup
```bash
# PostgreSQL o'rnatish
brew install postgresql@17  # macOS
# yoki
sudo apt install postgresql-17  # Ubuntu

# Ishga tushirish
brew services start postgresql@17

# Database yaratish
psql postgres
```

```sql
CREATE DATABASE devopslab;
CREATE USER devopslab WITH PASSWORD 'devopslab_secret';
GRANT ALL PRIVILEGES ON DATABASE devopslab TO devopslab;
\q
```

### 3. Docker Image Build
```bash
cd /Users/macbook/Documents/devops/devops-lab
docker build -f Dockerfile.simple -t devopslab:latest .
```

### 4. Backend Ishga Tushirish
```bash
cd /tmp
npm install dockerode pg  # Birinchi marta
node backend-production.js
```

**Natija:**
```
✅ DevOps Lab Backend v2.0 PRODUCTION
   HTTP: http://localhost:8080
   Features: Real Validation + Auto Cleanup + Colors
✅ PostgreSQL connected
✅ Database initialized
```

### 5. Frontend Ishga Tushirish
```bash
cd /Users/macbook/Documents/devops/devops-lab/frontend-vue
npm install  # Birinchi marta
npm run dev
```

**Natija:**
```
  VITE v5.4.20  ready in 141 ms

  ➜  Local:   http://localhost:3000/
```

### 6. Browser'da Ochish
```
http://localhost:3000
```

**Birinchi marta:** Register qiling
**Keyingi safar:** Login qiling

---

## 🎯 Qanday Ishlaydi?

### 1. Login/Register
- User ro'yxatdan o'tadi yoki login qiladi
- PostgreSQL `users` jadvaliga saqlanadi

### 2. Start Session
- "Start Session" tugmasi bosiladi
- Backend Docker container yaratadi:
  - Image: `devopslab:latest`
  - Resources: 512MB RAM, 50% CPU
  - TTL: 1 soat
- PostgreSQL `sessions` jadvaliga yoziladi
- WebSocket orqali terminal ulanadi

### 3. Task Navigation
- **Previous/Next/Skip** - Faqat UI o'zgaradi
- Container **o'zgarmaydi** ✅
- Bitta session'da barcha tasklar

### 4. Terminal Ishlash
- Browser'dagi xterm.js ← WebSocket → Backend → Docker container
- Real-time bidirectional communication
- ANSI ranglari qo'llab-quvvatlanadi

### 5. Check Solution
- "Check" tugmasi bosiladi
- Backend container'da validation bajaradi:
  - `which nginx` - o'rnatilganmi?
  - `test -f /var/www/html/index.html` - fayl bormi?
  - `cat /var/www/html/index.html` - content to'g'rimi?
  - `service nginx status` - ishlab turibdimi?
- Score hisoblanadi: `(passed / total) × points`
- PostgreSQL `progress` jadvaliga saqlanadi

### 6. Stop Session / Logout
- Container to'xtatiladi va o'chiriladi
- Database'dan session o'chiriladi
- Memory'dan tozalanadi

---

## 📚 Mavjud Tasklar

| # | Task ID | Nom | Difficulty | Points |
|---|---------|-----|------------|--------|
| 1 | `nginx-health` | Nginx Web Server | ⭐ | 25 |
| 2 | `docker-basics` | Docker Management | ⭐⭐ | 35 |
| 3 | `systemd-service` | Systemd Service | ⭐⭐ | 30 |
| 4 | `user-management` | User & Permissions | ⭐⭐⭐ | 35 |
| 5 | `firewall-config` | UFW Firewall | ⭐⭐⭐ | 40 |
| 6 | `cron-backup` | Cron & Backup | ⭐⭐⭐ | 45 |

**Jami:** 210 ball

**Validator Status:**
- ✅ `nginx-health` - To'liq ishlaydigan validator
- ⏳ Qolgan 5 task - Hali validator yozilmagan (generic response)

---

## 🔧 API Endpoints

### Authentication
```http
POST /api/auth/register
POST /api/auth/login
```

### Tasks
```http
GET /api/tasks
GET /api/tasks/:id
```

### Sessions
```http
POST   /api/sessions
DELETE /api/sessions/:id
```

### Validation
```http
POST /api/check/:sessionId
Body: { taskId: "nginx-health" }
```

### Progress
```http
GET /api/progress
```

### WebSocket
```http
WS /ws?sessionId=xxx
```

---

## 📁 Fayl Tuzilishi

```
devops-lab/
├── frontend-vue/              # Vue.js Frontend
│   ├── src/
│   │   ├── views/
│   │   │   └── LabView.vue    # Main interface
│   │   ├── components/
│   │   │   ├── Terminal.vue   # xterm.js terminal
│   │   │   ├── TaskDetail.vue
│   │   │   ├── CheckResults.vue
│   │   │   └── ProgressSummary.vue
│   │   ├── stores/            # Pinia state management
│   │   └── services/          # API client
│   ├── package.json
│   └── vite.config.ts
│
├── /tmp/backend-production.js # 🔥 ACTIVE BACKEND
│
├── backend/tasks/             # Task YAML definitions
│   ├── nginx-health/
│   ├── docker-basics/
│   ├── systemd-service/
│   ├── user-management/
│   ├── firewall-config/
│   └── cron-backup/
│
├── Dockerfile.simple          # Custom Docker image
├── README.md                  # Bu fayl
├── DOCUMENTATION.md           # To'liq texnik documentation
├── DEVELOPER_GUIDE.md         # Tez boshlash qo'llanmasi
└── QUICKSTART.md              # Uzbek quick start
```

---

## 🐛 Troubleshooting

### Backend ishlamayapti
```bash
# Port bandmi?
lsof -ti:8080 | xargs -r kill -9

# Qayta ishga tushir
cd /tmp && node backend-production.js
```

### PostgreSQL ulanmayapti
```bash
# Start
brew services start postgresql@17

# Test
psql -U devopslab -d devopslab -c "SELECT 1"
```

### Container tozalanmayapti
```bash
# Ko'rish
docker ps -a | grep devops-lab

# O'chirish
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f
```

### Terminal ulanmayapti
1. Backend ishlab turganini tekshir: `curl http://localhost:8080/api/tasks`
2. Container mavjudligini tekshir: `docker ps | grep devops-lab`
3. Session'ni restart qil: Stop → Start

---

## 📖 Documentation

| Fayl | Maqsad |
|------|--------|
| `README.md` | Asosiy ma'lumot (bu fayl) |
| `DOCUMENTATION.md` | To'liq texnik documentation (150+ sahifa) |
| `DEVELOPER_GUIDE.md` | Keyingi dasturchi uchun qo'llanma |
| `QUICKSTART.md` | Tez boshlash (Uzbek) |

---

## 🚀 Keyingi Qadamlar

### Dasturchilar Uchun:

1. **Documentation O'qing**
   ```bash
   cat DOCUMENTATION.md
   cat DEVELOPER_GUIDE.md
   ```

2. **Yangi Validator Qo'shing**
   - `docker-basics` task uchun
   - `systemd-service` task uchun
   - va hokazo...

3. **Security Yaxshilang**
   - Real JWT implementation
   - Password hashing (bcrypt)
   - Input validation (Joi)
   - Rate limiting

4. **Feature Qo'shing**
   - Hints system
   - Leaderboard
   - Certificates
   - Session recording

### Instructorlar Uchun:

1. **Custom Tasklar Yarating**
   - Task definition qo'shing
   - Validator yozing
   - Hints qo'shing

2. **Monitoring**
   - Container resource usage
   - User progress analytics
   - Session statistics

---

## ⚠️ Security Warning

**Bu loyiha ta'lim maqsadida!**

Production'da ishlatish uchun:
- ❌ Fake JWT tokens
- ❌ Plain text passwords
- ❌ No input validation
- ❌ CORS wide open
- ❌ Containers run as root
- ❌ No HTTPS

**Fixes:** `DOCUMENTATION.md` → Security Considerations

---

## 📊 System Requirements

- **OS:** macOS, Linux, Windows (WSL2)
- **Node.js:** 18.0.0+
- **Docker:** 24.0.0+
- **PostgreSQL:** 17.x
- **RAM:** 8GB minimum, 16GB recommended
- **Disk:** 10GB free

---

## 🎓 Learning Resources

- **Linux Commands:** https://linuxcommand.org/
- **Docker Tutorial:** https://docker-curriculum.com/
- **DevOps Roadmap:** https://roadmap.sh/devops
- **Nginx Guide:** https://nginx.org/en/docs/

---

## 📝 Changelog

### v2.0 (Current - 2025)
- ✅ Node.js backend (Go → Node.js)
- ✅ Vue.js frontend (React → Vue.js)
- ✅ Real Docker integration
- ✅ Custom WebSocket implementation
- ✅ PostgreSQL database
- ✅ Single session for all tasks
- ✅ Progress tracking
- ✅ Nginx validator

### v1.0 (Archived)
- Go backend + React frontend
- Mock terminal
- In-memory data

---

## 🤝 Contributing

1. Fork the repo
2. Create feature branch
3. Read `DOCUMENTATION.md`
4. Make changes
5. Test locally
6. Submit PR

---

## 📧 Support

**Muammo bo'lsa:**
1. `DOCUMENTATION.md` o'qing
2. `DEVELOPER_GUIDE.md` ko'ring
3. Logs tekshiring:
   - Backend: `tail -f /tmp/backend-production.log`
   - Browser: DevTools → Console
4. Container status: `docker ps -a`
5. Database: `psql -U devopslab -d devopslab`

---

## 📄 License

MIT License

---

## 👨‍💻 Credits

**Developed by:** Claude Code + Human Collaboration
**Year:** 2025
**Purpose:** Educational Platform
**Status:** Active Development

---

**🚀 Happy Learning!**

Frontend: http://localhost:3000
Backend: http://localhost:8080
Database: PostgreSQL (devopslab)
