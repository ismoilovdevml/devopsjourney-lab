# Troubleshooting - Login/Register Issues

## Current Status
- ✅ Backend: Running and healthy (http://localhost:8080)
- ✅ Frontend: Running (http://localhost:3000)
- ✅ Database: PostgreSQL healthy
- ✅ Redis: Healthy

## Tested & Working
- ✅ Backend API `/api/auth/register` - Works via curl
- ✅ Backend API `/api/auth/login` - Works via curl
- ✅ JWT token generation - Working
- ✅ Database operations - Working

## Issue Description
User reports that login and register are not working from the browser interface.

## Diagnostic Steps

### 1. Check Browser Console
**Action:** Open browser DevTools (F12 or Cmd+Option+I) and go to Console tab

**Look for:**
- Network errors (CORS, timeout, connection refused)
- JavaScript errors
- Failed fetch/axios requests

Common errors and solutions:
```
❌ "Network Error" → Backend not accessible
❌ "CORS policy" → CORS configuration issue
❌ "timeout of 30000ms exceeded" → Backend slow or not responding
❌ "Failed to fetch" → Connection issue
```

### 2. Check Network Tab
**Action:** Open DevTools → Network tab → Try to register/login

**Look for:**
- Status codes: 200 (success), 400 (bad request), 401 (unauthorized), 500 (server error)
- Response time
- Request/Response payloads

### 3. Test API Directly

#### Test Registration (curl)
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser123","email":"test123@test.uz","password":"test12345"}'
```

Expected response:
```json
{
  "token": "eyJ...",
  "user": {
    "id": 1,
    "username": "testuser123",
    ...
  }
}
```

#### Test Login (curl)
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser123","password":"test12345"}'
```

### 4. Check Backend Logs
```bash
docker-compose logs backend --tail=50
```

Look for:
- HTTP POST requests to `/api/auth/register` or `/api/auth/login`
- Error messages
- Stack traces

### 5. Check Frontend Logs
```bash
docker-compose logs frontend --tail=50
```

### 6. Verify Services
```bash
docker-compose ps
```

All services should show "Up" and "healthy" status.

## Common Issues & Solutions

### Issue 1: CORS Error
**Error in browser:** "Access to fetch at 'http://localhost:8080/api/auth/register' from origin 'http://localhost:3000' has been blocked by CORS policy"

**Solution:**
```bash
# Check CORS configuration in backend logs
docker-compose logs backend | grep CORS

# Should show: "allowed_origins":["http://localhost:3000"]

# If not, rebuild backend:
docker-compose build backend
docker-compose restart backend
```

### Issue 2: Connection Refused
**Error:** "Failed to fetch" or "ERR_CONNECTION_REFUSED"

**Solution:**
```bash
# Check if backend is running
curl http://localhost:8080/healthz

# Should return: {"status":"ok",...}

# If not running:
docker-compose restart backend
```

### Issue 3: Timeout
**Error:** "timeout of 30000ms exceeded"

**Solution:**
```bash
# Check backend is responding
time curl http://localhost:8080/healthz

# Should respond in < 1 second

# If slow, check backend resources:
docker stats devopslab-backend
```

### Issue 4: Wrong API URL
**Check frontend environment:**
```bash
# Frontend should use: http://localhost:8080/api
# Check compiled frontend code or .env file
cat /Users/macbook/Documents/devops/devops-lab/frontend-vue/.env
```

Should contain:
```
VITE_API_URL=http://localhost:8080/api
```

If not:
```bash
echo 'VITE_API_URL=http://localhost:8080/api' > /Users/macbook/Documents/devops/devops-lab/frontend-vue/.env
docker-compose build frontend
docker-compose restart frontend
```

### Issue 5: Browser Cache
**Solution:** Hard refresh the browser
- Chrome/Edge: Ctrl+Shift+R (Windows) or Cmd+Shift+R (Mac)
- Firefox: Ctrl+F5 (Windows) or Cmd+Shift+R (Mac)
- Safari: Cmd+Option+R

Or clear browser cache:
- Open DevTools → Application/Storage tab → Clear storage

### Issue 6: Database Issues
**Error in backend logs:** "migration failed" or "database connection error"

**Solution:**
```bash
# Reset database
docker exec devopslab-postgres psql -U devopslab -d devopslab -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Restart backend to run migrations
docker-compose restart backend
```

## Quick Fix - Full Restart

If nothing works, try a complete restart:

```bash
# Stop all services
docker-compose down

# Rebuild all images
docker-compose build

# Start services
docker-compose up -d backend frontend

# Wait for healthy status
sleep 10

# Check status
docker-compose ps

# Check backend health
curl http://localhost:8080/healthz

# Open browser
open http://localhost:3000
```

## Testing Checklist

After any fix, test these:

- [ ] Backend health: `curl http://localhost:8080/healthz`
- [ ] Register via curl: Works?
- [ ] Login via curl: Works?
- [ ] Browser console: No errors?
- [ ] Browser network tab: Requests succeed?
- [ ] Frontend loads: http://localhost:3000
- [ ] Can switch between Login/Register tabs?
- [ ] Form submission: Gets response?

## Get More Info

To help diagnose, provide:

1. **Browser console output** (full error message)
2. **Network tab details** (failed request info)
3. **Backend logs** during the request:
   ```bash
   docker-compose logs backend --tail=100
   ```
4. **What happens when you submit the form?**
   - Does it show loading spinner?
   - Does it return immediately with error?
   - Does it timeout?

## Contact Support

If issue persists, create a GitHub issue with:
- Browser console screenshot
- Network tab screenshot
- Output of: `docker-compose ps`
- Output of: `docker-compose logs backend --tail=100`
