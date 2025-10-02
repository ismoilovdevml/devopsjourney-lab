# DevOps Journey Lab - Deployment Guide

Complete guide for deploying DevOps Journey Lab using Docker Compose.

## 📋 Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 2GB RAM available
- 10GB free disk space

## 🚀 Quick Start (Production Deployment)

### 1. Clone the Repository

```bash
git clone <repository-url>
cd devops-lab
```

### 2. Build DevOps Lab Student Container Image

This image is used for student lab sessions:

```bash
docker build -f Dockerfile.simple -t devopslab:latest .
```

### 3. Configure Environment Variables

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` if you need to customize:

```bash
nano .env
```

**Important:** Change `POSTGRES_PASSWORD` in production!

### 4. Start All Services

```bash
docker-compose up -d
```

This will start:
- **PostgreSQL** (port 5432) - Database
- **Backend** (port 8080) - API Server
- **Frontend** (port 3000) - Web Interface

### 5. Verify Services

Check all services are running:

```bash
docker-compose ps
```

Expected output:
```
NAME                    IMAGE                  STATUS         PORTS
devopslab-backend       devopslab-backend      Up 10 seconds  0.0.0.0:8080->8080/tcp
devopslab-frontend      devopslab-frontend     Up 10 seconds  0.0.0.0:3000->80/tcp
devopslab-postgres      postgres:17-alpine     Up 15 seconds  0.0.0.0:5432->5432/tcp
```

### 6. Access the Application

- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080/api
- **Health Check:** http://localhost:8080/healthz

### 7. Create Test User

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

## 🔧 Management Commands

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres
```

### Restart Services

```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart backend
```

### Stop Services

```bash
docker-compose stop
```

### Stop and Remove Containers

```bash
docker-compose down
```

### Stop and Remove Everything (including volumes)

```bash
docker-compose down -v
```

⚠️ **Warning:** This will delete all database data!

## 🛠️ Development Mode

For development with hot reload:

### Backend (Development)

```bash
cd /path/to/backend
node backend.js
```

### Frontend (Development)

```bash
cd frontend-vue
npm install
npm run dev
```

Access at http://localhost:5173 (Vite dev server)

## 📦 Building Images Separately

### Build Backend Image

```bash
docker build -f Dockerfile.backend -t devopslab-backend:latest .
```

### Build Frontend Image

```bash
docker build -f frontend-vue/Dockerfile -t devopslab-frontend:latest ./frontend-vue
```

### Build Student Lab Image

```bash
docker build -f Dockerfile.simple -t devopslab:latest .
```

## 🐛 Troubleshooting

### Backend Can't Connect to PostgreSQL

```bash
# Check if PostgreSQL is healthy
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Restart PostgreSQL
docker-compose restart postgres
```

### Frontend Can't Connect to Backend

1. Check backend is running:
```bash
curl http://localhost:8080/healthz
```

2. Check nginx configuration in `frontend-vue/nginx.conf`

3. Verify CORS settings in backend

### Student Containers Not Starting

1. Ensure `devopslab:latest` image exists:
```bash
docker images | grep devopslab
```

2. Check Docker socket permissions:
```bash
ls -la /var/run/docker.sock
```

3. Check backend logs:
```bash
docker-compose logs backend | grep "Container"
```

### Port Already in Use

```bash
# Find process using port 8080
lsof -ti:8080

# Kill the process
lsof -ti:8080 | xargs kill -9

# Or change port in docker-compose.yml
```

## 🔐 Security Considerations

### Production Deployment

1. **Change default passwords:**
   - PostgreSQL password in `.env`
   - Create strong admin user password

2. **Use HTTPS:**
   - Set up reverse proxy (nginx/traefik)
   - Obtain SSL certificate (Let's Encrypt)

3. **Limit exposed ports:**
   - Only expose frontend (port 80/443)
   - Keep backend and database internal

4. **Enable firewall:**
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

5. **Regular updates:**
```bash
docker-compose pull
docker-compose up -d
```

## 📊 Monitoring

### Check Resource Usage

```bash
docker stats
```

### Check Container Health

```bash
docker inspect devopslab-backend | grep -A 10 Health
```

### Database Backup

```bash
docker exec devopslab-postgres pg_dump -U devopslab devopslab > backup.sql
```

### Database Restore

```bash
docker exec -i devopslab-postgres psql -U devopslab devopslab < backup.sql
```

## 🌐 Production Deployment with Domain

### 1. Add Reverse Proxy (nginx)

Create `nginx.conf` for reverse proxy:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 2. SSL with Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## 📝 Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_HOST` | PostgreSQL hostname | `postgres` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | Database user | `devopslab` |
| `POSTGRES_PASSWORD` | Database password | `devopslab_secret` |
| `POSTGRES_DB` | Database name | `devopslab` |
| `PORT` | Backend port | `8080` |
| `NODE_ENV` | Environment | `production` |
| `DEFAULT_IMAGE` | Student container image | `devopslab:latest` |
| `SESSION_TTL` | Session timeout (seconds) | `3600` |
| `CONTAINER_MEMORY` | Max memory per container | `512m` |
| `VITE_API_URL` | Frontend API URL | `http://localhost:8080/api` |

## 🎯 Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐      ┌──────────────┐
│  Frontend       │      │  Backend     │
│  (nginx:alpine) │─────▶│  (node:20)   │
│  Port: 3000     │      │  Port: 8080  │
└─────────────────┘      └──────┬───────┘
                                │
                    ┌───────────┴────────────┐
                    │                        │
                    ▼                        ▼
            ┌────────────────┐      ┌────────────────┐
            │  PostgreSQL    │      │  Docker Engine │
            │  Port: 5432    │      │  (Students)    │
            └────────────────┘      └────────────────┘
```

## 📞 Support

For issues and questions:
- Check logs: `docker-compose logs`
- Check health: `docker-compose ps`
- GitHub Issues: [Repository Issues]

## 📄 License

Copyright © 2025 DevOps Journey Lab
