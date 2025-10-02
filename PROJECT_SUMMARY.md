# DevOps Lab v2.0 - Project Summary

> Keyingi dasturchi uchun qisqa xulosalar va asosiy ma'lumotlar

---

## 📌 Loyiha Nima?

**DevOps Lab** - Studentlar real Linux container'larda DevOps/Linux mashqlarini bajariladigan interaktiv platforma.

**Maqsad:** Nazariy bilimlarni amaliyotda mustahkamlash.

---

## 🎯 Asosiy Xususiyatlari

1. **Real Docker Containers** - Har session uchun alohida Ubuntu 22.04 container
2. **Browser Terminal** - xterm.js orqali to'liq Linux terminal
3. **Auto-validation** - Yechimlarni avtomatik tekshirish va ball berish
4. **Progress Tracking** - PostgreSQL'da user progress saqlanadi
5. **Single Session Logic** - Bitta container'da barcha tasklar (Container saqlanadi!)

---

## 🏗️ Texnologiyalar

```
Frontend:  Vue.js 3 + TypeScript + Vite + Pinia
Backend:   Node.js (Native HTTP + Custom WebSocket)
Database:  PostgreSQL 17
Container: Docker + dockerode SDK
```

---

## 📂 Asosiy Fayllar

| Fayl | Vazifasi |
|------|----------|
| `/tmp/backend-production.js` | 🔥 **ASOSIY BACKEND** - Hamma logika shu yerda |
| `frontend-vue/src/views/LabView.vue` | Main UI interface |
| `frontend-vue/src/components/Terminal.vue` | xterm.js terminal |
| `frontend-vue/src/stores/session.ts` | Session boshqaruvi |
| `Dockerfile.simple` | Custom Docker image (Ubuntu + tools) |

---

## 🔑 Asosiy Mantiq (Session Lifecycle)

```
1. START SESSION
   → Docker container yaratiladi
   → PostgreSQL'ga session yoziladi
   → WebSocket terminal ulanadi

2. NEXT/PREVIOUS (Task navigation)
   → Faqat UI o'zgaradi ✅
   → Container QOLADI ✅
   → Bitta session'da barcha tasklar

3. CHECK
   → Frontend current task ID yuboradi
   → Backend o'sha task'ni tekshiradi (container'da validation)
   → Score hisoblanadi va PostgreSQL'ga saqlanadi

4. STOP/LOGOUT
   → Container to'xtatiladi va o'chiriladi
   → Database'dan session o'chiriladi
```

**MUHIM FARQ:** Eski yondashuvda har task uchun yangi container yaratilgan. Endi bitta container'da barcha tasklar! ✅

---

## 📊 Database Schema

### users
```sql
id | username | email | password | role | created_at
```

### sessions
```sql
session_id | user_id | task_id | container_id | status | created_at | expires_at
```

### progress
```sql
id | user_id | task_id | score | total | passed | completed_at
```

---

## 🔌 API Endpoints (Qisqacha)

```
POST   /api/auth/register
POST   /api/auth/login
GET    /api/tasks
POST   /api/sessions          { taskId }
DELETE /api/sessions/:id
POST   /api/check/:sessionId  { taskId }  ← Current task tekshirish
GET    /api/progress
WS     /ws?sessionId=xxx
```

---

## 🚀 Ishga Tushirish (3 qadam)

```bash
# 1. PostgreSQL
brew services start postgresql@17
psql postgres -c "CREATE DATABASE devopslab; CREATE USER devopslab WITH PASSWORD 'devopslab_secret';"

# 2. Backend
cd /tmp && node backend-production.js &

# 3. Frontend
cd ~/Documents/devops/devops-lab/frontend-vue && npm run dev
```

**Browser:** http://localhost:3000

---

## 🧪 Qanday Test Qilish?

1. **Register/Login** → http://localhost:3000
2. **Start Session** → 1 ta container yaratiladi
3. **Terminal'da ishlash** → `apt-get update`, `apt-get install nginx -y`, ...
4. **Check** → Natija chiqadi (pass/fail, score)
5. **Next** → 2-chi taskga o't → Container **qoladi** ✅
6. **Check** → Endi 2-chi task tekshiriladi
7. **Stop Session** → Container o'chadi

```bash
# Monitor containers
docker ps -a | grep devops-lab
```

---

## ✅ Validator Qo'shish (Template)

```javascript
// Backend: /tmp/backend-production.js

async function checkYourTask(container, task) {
  const results = [];

  try {
    const exec = await container.exec({
      Cmd: ['your', 'command'],
      AttachStdout: true
    });
    const stream = await exec.start({ hijack: true });
    let output = '';
    stream.on('data', chunk => output += chunk.toString());
    await new Promise(resolve => stream.on('end', resolve));

    results.push({
      name: '1. Check description',
      ok: output.includes('expected'),
      output: output.trim()
    });
  } catch (err) {
    results.push({
      name: '1. Check description',
      ok: false,
      output: '✗ Error: ' + err.message
    });
  }

  return results;
}

// Register validator
if (task.id === 'your-task-id') {
  checkYourTask(container, task).then(/* ... */);
}
```

---

## 🐛 Tez-tez Uchraydigan Muammolar

| Muammo | Yechim |
|--------|--------|
| Backend ishlamayapti | `lsof -ti:8080 \| xargs -r kill -9` |
| PostgreSQL ulanmayapti | `brew services start postgresql@17` |
| Container tozalanmayapti | `docker ps -a \| grep devops-lab \| awk '{print $1}' \| xargs docker rm -f` |
| Terminal ulanmayapti | Backend va container borligini tekshir, session restart qil |

---

## 📖 Documentation

| Fayl | Hajm | Maqsad |
|------|------|--------|
| `README.md` | 5 sahifa | Asosiy overview |
| `DOCUMENTATION.md` | 150 sahifa | **TO'LIQ** texnik doc |
| `DEVELOPER_GUIDE.md` | 20 sahifa | Tez boshlash qo'llanmasi |
| `PROJECT_SUMMARY.md` | Bu fayl | Qisqa xulosalar |

---

## 🔐 Security (MUHIM!)

⚠️ **Bu production-ready emas!** Faqat ta'lim uchun.

**Muammolar:**
- Fake JWT tokens
- Plain text passwords
- No input validation
- CORS `*`
- Containers run as root

**Fix:** `DOCUMENTATION.md` → Security Considerations bo'limida batafsil yechimlar.

---

## 📋 Task List

| # | Task | Validator | Status |
|---|------|-----------|--------|
| 1 | nginx-health | ✅ To'liq | Working |
| 2 | docker-basics | ⏳ Yo'q | Generic response |
| 3 | systemd-service | ⏳ Yo'q | Generic response |
| 4 | user-management | ⏳ Yo'q | Generic response |
| 5 | firewall-config | ⏳ Yo'q | Generic response |
| 6 | cron-backup | ⏳ Yo'q | Generic response |

**TODO:** Qolgan 5 ta task uchun validator yozish kerak!

---

## 💡 Keyingi Qadamlar

### Dasturchilar Uchun:
1. ✅ `DOCUMENTATION.md` o'qish
2. ✅ Codebase'ni tushunish (Backend → Frontend → DB)
3. ⏳ Qolgan validatorlarni yozish
4. ⏳ Security fixes (JWT, bcrypt, validation)
5. ⏳ New features (hints, leaderboard, certificates)

### Ta'lim Beruvchilar Uchun:
1. ✅ Platformani test qilish
2. ⏳ Custom tasklar qo'shish
3. ⏳ Progress analytics ko'rish
4. ⏳ Student monitoring

---

## 🎓 Foydali Linklar

- **Docker API:** https://docs.docker.com/engine/api/
- **Vue 3:** https://vuejs.org/guide/
- **xterm.js:** https://xtermjs.org/docs/
- **PostgreSQL:** https://www.postgresql.org/docs/

---

## 📊 Statistika

- **Backend Lines:** ~600 lines (Node.js)
- **Frontend Components:** 10+ Vue components
- **Database Tables:** 3 (users, sessions, progress)
- **API Endpoints:** 8
- **Docker Images:** 1 custom image (~300MB)
- **Tasks:** 6 (1 validator working, 5 pending)

---

## ✨ Nima Ishlaydi?

✅ Real Docker container creation
✅ WebSocket terminal (bidirectional)
✅ PostgreSQL persistence
✅ User authentication (basic)
✅ Session management
✅ Task navigation
✅ Nginx validation
✅ Progress tracking
✅ Auto-cleanup (1 hour)
✅ Terminal colors (ANSI)

## ⏳ Nima Qoldi?

⏳ Remaining task validators (5)
⏳ Real JWT tokens
⏳ Password hashing
⏳ Input validation
⏳ Rate limiting
⏳ Hints system
⏳ Leaderboard
⏳ Certificates
⏳ Session recording
⏳ Better error handling

---

## 🚀 Tez Komandalar

```bash
# Start everything
brew services start postgresql@17
cd /tmp && node backend-production.js &
cd ~/Documents/devops/devops-lab/frontend-vue && npm run dev

# Stop everything
pkill -f backend-production
lsof -ti:3000 | xargs -r kill -9

# Clean containers
docker ps -a | grep devops-lab | awk '{print $1}' | xargs docker rm -f

# Check status
curl http://localhost:8080/api/tasks
docker ps | grep devops-lab
psql -U devopslab -d devopslab -c "SELECT * FROM sessions;"
```

---

## 📞 Qayerdan Boshlash?

1. **Yangi dasturchi?**
   - `README.md` o'qi
   - `DEVELOPER_GUIDE.md` o'qi
   - Loyihani ishga tushir va test qil
   - `DOCUMENTATION.md` batafsil o'rgan

2. **Bug fix qilish?**
   - `DOCUMENTATION.md` → Troubleshooting
   - Backend logs: `tail -f /tmp/backend-production.log`
   - Browser console: DevTools → Console

3. **Validator qo'shish?**
   - `DOCUMENTATION.md` → Task Validation System
   - `DEVELOPER_GUIDE.md` → Validator Template
   - Nginx validator'ni misol qil

4. **Security fix?**
   - `DOCUMENTATION.md` → Security Considerations
   - JWT, bcrypt, validation qo'sh

---

**End of Summary**

Batafsil ma'lumot: `DOCUMENTATION.md` (150+ sahifa)

Backend: `/tmp/backend-production.js`
Frontend: `frontend-vue/`
Database: PostgreSQL (devopslab)

**Good luck! 🚀**
