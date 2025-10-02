# 🚀 DevOps Journey Lab - Quick Deployment Guide

## Option 1: One-Command Start (Recommended for Testing)

```bash
./start.sh
```

This will:
- Build the student lab Docker image
- Start PostgreSQL, Backend, and Frontend with Docker Compose
- Display access URLs

## Option 2: Manual Deployment

### Step 1: Build Student Lab Image

```bash
docker build -f Dockerfile.simple -t devopslab:latest .
```

### Step 2: Start All Services

```bash
docker-compose up -d
```

### Step 3: Access Application

- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080/api

## Option 3: Development Mode (Without Docker)

### Terminal 1 - Backend
```bash
# Ensure PostgreSQL is running (brew install postgresql@17)
brew services start postgresql@17

# Create database
createdb devopslab
psql devopslab -c "CREATE USER devopslab WITH PASSWORD 'devopslab_secret';"
psql devopslab -c "GRANT ALL PRIVILEGES ON DATABASE devopslab TO devopslab;"

# Start backend
cd /tmp
node backend-production.js
```

### Terminal 2 - Frontend
```bash
cd frontend-vue
npm install
npm run dev
```

Access at: http://localhost:3000

## 📊 Service Management

### View Logs
```bash
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Restart Services
```bash
docker-compose restart
```

### Stop Services
```bash
docker-compose down
```

### Full Cleanup (including data)
```bash
docker-compose down -v
docker rmi devopslab:latest devops-lab-backend devops-lab-frontend
```

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────┐
│                        Browser                           │
└────────────────────────┬─────────────────────────────────┘
                         │
                         ▼
         ┌───────────────────────────────┐
         │     Frontend (Port 3000)      │
         │   Vue 3 + TypeScript + Vite   │
         │        nginx:alpine           │
         └────────┬──────────────────────┘
                  │
                  │ /api/* → Backend
                  │
                  ▼
         ┌───────────────────────────────┐
         │      Backend (Port 8080)      │
         │    Node.js + PostgreSQL       │
         │   Dockerode (Container Mgmt)  │
         └────────┬──────────┬───────────┘
                  │          │
         ┌────────▼────┐   ┌─▼──────────────────┐
         │ PostgreSQL  │   │   Docker Engine    │
         │  (Port 5432)│   │ (Student Sessions) │
         └─────────────┘   └────────────────────┘
```

## 🎯 Ports

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Web interface |
| Backend | 8080 | REST API |
| PostgreSQL | 5432 | Database |

## 🔐 Default Credentials

**Database:**
- User: `devopslab`
- Password: `devopslab_secret`
- Database: `devopslab`

**Application:**
- Create your own user via register page

## 📝 Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
nano .env
```

## 🐛 Troubleshooting

### Port Already in Use

```bash
# Find and kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Find and kill process on port 3000
lsof -ti:3000 | xargs kill -9
```

### Backend Can't Connect to PostgreSQL

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Restart PostgreSQL
docker-compose restart postgres
```

### Docker Permission Denied

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Restart Docker Desktop (Mac/Windows)
```

### Student Containers Not Starting

```bash
# Ensure devopslab image exists
docker images | grep devopslab

# Rebuild if missing
docker build -f Dockerfile.simple -t devopslab:latest .
```

## 📦 What Gets Deployed

1. **PostgreSQL Database** - Stores users, sessions, progress
2. **Node.js Backend** - API server, manages student containers
3. **Vue Frontend** - Web interface served by nginx
4. **Student Containers** - Isolated Ubuntu environments for labs

## 🔄 Updates

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose down
docker-compose up -d --build
```

## 📈 Production Deployment

For production with domain name:

1. **Set up reverse proxy (nginx/traefik)**
2. **Enable HTTPS with Let's Encrypt**
3. **Change default passwords in `.env`**
4. **Set up automatic backups**
5. **Configure firewall rules**

See `DEPLOYMENT.md` for detailed production deployment guide.

## 📞 Support

- GitHub Issues: [Create Issue]
- Documentation: See `DEPLOYMENT.md`
- Logs: `docker-compose logs -f`
