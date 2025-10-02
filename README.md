# 🚀 DevOps Journey Lab

Enterprise-grade DevOps learning platform built with **Go** and **Vue.js**.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  DevOps Journey Lab                     │
├─────────────────┬─────────────────┬────────────────────┤
│   Frontend      │    Backend      │    Database        │
│   (Vue.js)      │    (Go)         │    (PostgreSQL)    │
│   + TypeScript  │    + Gin        │    + Redis         │
│   + Tailwind    │    + GORM       │                    │
│   + Nginx       │    + Docker SDK │                    │
└─────────────────┴─────────────────┴────────────────────┘
```

### 🎯 Tech Stack

- **Frontend:** Vue 3 + TypeScript + Tailwind CSS + Vite
- **Backend:** Go 1.24 + Gin + GORM + JWT
- **Database:** PostgreSQL 17 + Redis 7
- **Infrastructure:** Docker + Docker Compose
- **Terminal:** Xterm.js for in-browser shell

---

## 🚦 Quick Start

### 1️⃣ Prerequisites

- Docker & Docker Compose
- Git

### 2️⃣ Clone & Setup

```bash
git clone <repository-url>
cd devopsjourney-lab
```

### 3️⃣ Configure Environment

```bash
cp .env.example .env
# Edit .env and change admin credentials
nano .env
```

**Important ENV Variables:**
```bash
# Admin User (CHANGE IN PRODUCTION!)
SEED_ADMIN=true
ADMIN_USERNAME=admin
ADMIN_EMAIL=admin@devopslab.com
ADMIN_PASSWORD=admin123

# JWT Secret (MUST CHANGE!)
JWT_SECRET=your-super-secret-jwt-key-change-in-production
```

### 4️⃣ Build Student Lab Image

```bash
docker build -f backend-go/Dockerfile.devops -t devopslab:latest backend-go/
```

### 5️⃣ Start Platform

```bash
docker-compose up -d
```

### 6️⃣ Access Platform

- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080/api
- **Health Check:** http://localhost:8080/healthz

---

## 📂 Project Structure

```
devopsjourney-lab/
├── backend-go/              # Go Backend
│   ├── cmd/                 # Application entrypoint
│   │   └── server/main.go
│   ├── configs/             # Configuration
│   ├── internal/            # Internal packages
│   │   ├── api/             # HTTP handlers
│   │   ├── auth/            # Authentication (JWT, bcrypt)
│   │   ├── database/        # Database & migrations
│   │   ├── docker/          # Docker client
│   │   ├── middleware/      # Gin middlewares
│   │   ├── models/          # Data models
│   │   ├── session/         # Session management
│   │   ├── task/            # Task loader
│   │   └── websocket/       # WebSocket terminal
│   ├── pkg/                 # Public packages
│   ├── tasks/               # DevOps challenge definitions
│   ├── Dockerfile           # Production backend image
│   ├── go.mod
│   └── go.sum
│
├── frontend-vue/            # Vue Frontend
│   ├── src/                 # Vue source code
│   │   ├── components/      # Vue components
│   │   ├── router/          # Vue Router
│   │   ├── stores/          # Pinia state management
│   │   ├── views/           # Page views
│   │   └── main.ts
│   ├── Dockerfile           # Production frontend image
│   ├── nginx.conf           # Nginx configuration
│   ├── package.json
│   └── vite.config.ts
│
├── docker-compose.yml       # Multi-container orchestration
├── .env.example             # Environment template
├── .gitignore
└── README.md
```

---

## 🔧 Services

### PostgreSQL (Port 5432)
- Database for users, sessions, progress
- Auto-initialized with migrations

### Redis (Port 6379)
- Session caching
- Rate limiting

### Backend (Port 8080)
- REST API (Gin framework)
- WebSocket terminal
- Docker container orchestration
- JWT authentication

### Frontend (Port 3000)
- Vue.js SPA
- Nginx reverse proxy
- Real-time terminal via WebSocket

---

## 📋 Available Commands

### Development

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Restart backend only
docker-compose restart backend

# Rebuild backend
docker-compose up -d --build backend
```

### Database

```bash
# Access PostgreSQL
docker exec -it devopslab-postgres psql -U devopslab

# View tables
docker exec -it devopslab-postgres psql -U devopslab -c "\dt"
```

### Backend Development

```bash
cd backend-go

# Run locally (requires Go 1.24+)
go run cmd/server/main.go

# Run tests
go test ./...

# Build binary
go build -o server cmd/server/main.go
```

### Frontend Development

```bash
cd frontend-vue

# Install dependencies
npm install

# Dev server
npm run dev

# Build for production
npm run build
```

---

## 🔐 Authentication

### Admin User

Admin user is automatically created on first run if `SEED_ADMIN=true`:

- **Username:** `admin` (from ENV)
- **Email:** `admin@devopslab.com` (from ENV)
- **Password:** `admin123` (from ENV)
- **Role:** `admin`

**⚠️ IMPORTANT:** Change admin credentials in `.env` before production deployment!

### User Registration

Students can register at `/register`:
- Username (min 3 chars)
- Email (valid format)
- Password (min 8 chars)
- Auto-assigned role: `student`

### JWT Authentication

- Token expiration: 24h (configurable via `JWT_EXPIRATION`)
- Stored in localStorage
- Sent via `Authorization: Bearer <token>` header

---

## 🎓 Features

✅ **Multi-user support** - Students and admins
✅ **Real-time terminal** - Browser-based shell via WebSocket
✅ **Docker isolation** - Each session in separate container
✅ **Auto-cleanup** - Sessions expire after 1 hour (configurable)
✅ **Progress tracking** - Save scores and completion status
✅ **Task validation** - Automated checking with custom validators
✅ **Admin dashboard** - User management, session monitoring
✅ **ENV-based config** - No hardcoded credentials

---

## 🛡️ Security

- ✅ Passwords hashed with bcrypt
- ✅ JWT-based authentication
- ✅ CORS configured for specific origins
- ✅ Rate limiting via Redis
- ✅ Container resource limits (CPU, Memory, PIDs)
- ✅ Admin credentials from ENV only

---

## 📊 Monitoring

### Health Check

```bash
curl http://localhost:8080/healthz
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2025-10-02T10:30:00Z",
  "version": "2.0.0",
  "uptime": "2h30m15s"
}
```

### Prometheus (Optional)

Enable monitoring:
```bash
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090
```

---

## 🐛 Troubleshooting

### Backend won't start

```bash
# Check logs
docker-compose logs backend

# Common issue: PostgreSQL not ready
# Wait 10-20 seconds and restart
docker-compose restart backend
```

### Database connection failed

```bash
# Check PostgreSQL health
docker-compose ps postgres

# Restart database
docker-compose restart postgres
```

### Frontend can't reach backend

1. Check CORS origins in `.env`:
   ```bash
   CORS_ORIGINS=http://localhost:3000,http://localhost:80
   ```

2. Verify backend is running:
   ```bash
   curl http://localhost:8080/healthz
   ```

---

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 👨‍💻 Author

Built with ❤️ for DevOps learners

---

## 📞 Support

- 📧 Email: support@devopslab.com
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/devopsjourney-lab/issues)
- 📖 Docs: [Wiki](https://github.com/yourusername/devopsjourney-lab/wiki)

---

## 🎯 Roadmap

- [ ] Kubernetes deployment
- [ ] CI/CD pipeline examples
- [ ] More DevOps challenges
- [ ] Multi-language support
- [ ] Mobile responsive UI
- [ ] Team/classroom mode
- [ ] Leaderboard system

---

**Happy Learning! 🚀**
