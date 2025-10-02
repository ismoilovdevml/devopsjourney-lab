const http = require('http');
const url = require('url');
const crypto = require('crypto');
const Docker = require('dockerode');
const { Client } = require('pg');

const docker = new Docker({ socketPath: '/var/run/docker.sock' });

const pgClient = new Client({
  host: process.env.POSTGRES_HOST || 'localhost',
  port: parseInt(process.env.POSTGRES_PORT || '5432'),
  database: process.env.POSTGRES_DB || 'devopslab',
  user: process.env.POSTGRES_USER || 'devopslab',
  password: process.env.POSTGRES_PASSWORD || 'devopslab_secret'
});

// In-memory fallback storage - DEFAULT MODE
let dbConnected = false;
const inMemoryUsers = new Map(); // username -> user object
const inMemorySessions = new Map(); // session_id -> session object
const inMemoryProgress = new Map(); // user_id -> array of progress
let userIdCounter = 1;

// Connect to PostgreSQL
pgClient.connect().then(() => {
  console.log('✅ PostgreSQL connected');
  dbConnected = true;
  initDatabase();
}).catch(err => {
  console.error('❌ PostgreSQL:', err.message);
  console.log('⚠️  Using in-memory storage (data will be lost on restart)');
  dbConnected = false;
});

async function initDatabase() {
  if (!dbConnected) return;

  await pgClient.query(`
    CREATE TABLE IF NOT EXISTS users (
      id SERIAL PRIMARY KEY,
      username VARCHAR(100) UNIQUE NOT NULL,
      email VARCHAR(100) UNIQUE NOT NULL,
      password VARCHAR(255) NOT NULL,
      role VARCHAR(50) DEFAULT 'student',
      created_at TIMESTAMP DEFAULT NOW(),
      last_login TIMESTAMP,
      is_online BOOLEAN DEFAULT FALSE
    )
  `);
  await pgClient.query(`
    CREATE TABLE IF NOT EXISTS sessions (
      session_id VARCHAR(100) PRIMARY KEY,
      user_id INTEGER,
      task_id VARCHAR(100) NOT NULL,
      container_id VARCHAR(100) NOT NULL,
      status VARCHAR(50) DEFAULT 'active',
      created_at TIMESTAMP DEFAULT NOW(),
      expires_at TIMESTAMP NOT NULL
    )
  `);
  await pgClient.query(`
    CREATE TABLE IF NOT EXISTS progress (
      id SERIAL PRIMARY KEY,
      user_id INTEGER,
      task_id VARCHAR(100) NOT NULL,
      score INTEGER DEFAULT 0,
      total INTEGER DEFAULT 0,
      passed BOOLEAN DEFAULT FALSE,
      completed_at TIMESTAMP DEFAULT NOW()
    )
  `);
  await pgClient.query(`
    CREATE TABLE IF NOT EXISTS user_activity_logs (
      id SERIAL PRIMARY KEY,
      user_id INTEGER,
      username VARCHAR(100),
      action VARCHAR(50) NOT NULL,
      ip_address VARCHAR(50),
      user_agent TEXT,
      created_at TIMESTAMP DEFAULT NOW()
    )
  `);
  await pgClient.query(`
    CREATE TABLE IF NOT EXISTS container_stats (
      id SERIAL PRIMARY KEY,
      session_id VARCHAR(100),
      container_id VARCHAR(100),
      cpu_usage DECIMAL,
      memory_usage BIGINT,
      memory_limit BIGINT,
      network_rx BIGINT,
      network_tx BIGINT,
      recorded_at TIMESTAMP DEFAULT NOW()
    )
  `);
  console.log('✅ Database initialized');
}

const sessions = new Map();
const containers = new Map();
let sessionCounter = 0;

const taskDetails = {
  'nginx-health': {
    id:'nginx-health',title:'Test 1: Nginx Web Server',order:1,difficulty:1,unlocked:true,completed:false,points:25,
    description:'Nginx web server o\'rnatish va sozlash.',
    hints:['apt-get update','nginx o\'rnatish','index.html yaratish','/var/www/html/ papka','nginx start']
  },
  'docker-basics': {
    id:'docker-basics',title:'Test 2: Docker Management',order:2,difficulty:2,unlocked:true,completed:false,points:35,
    description:'Docker container yaratish va boshqarish.',
    hints:['docker pull','docker run','-d flag','--name','docker ps']
  },
  'systemd-service': {
    id:'systemd-service',title:'Test 3: Systemd Service',order:3,difficulty:2,unlocked:true,completed:false,points:30,
    description:'Systemd service yaratish va sozlash.',
    hints:['Service file','ExecStart','daemon-reload','enable','start']
  },
  'user-management': {
    id:'user-management',title:'Test 4: User Management',order:4,difficulty:3,unlocked:true,completed:false,points:35,
    description:'User va permission boshqaruvi.',
    hints:['useradd','groupadd','usermod','chmod','chown']
  },
  'firewall-config': {
    id:'firewall-config',title:'Test 5: Firewall Config',order:5,difficulty:3,unlocked:true,completed:false,points:40,
    description:'UFW firewall sozlash va boshqarish.',
    hints:['ufw install','enable','allow','deny','status']
  },
  'cron-backup': {
    id:'cron-backup',title:'Test 6: Cron & Backup',order:6,difficulty:3,unlocked:true,completed:false,points:45,
    description:'Cron job va backup skripting.',
    hints:['backup.sh','tar','crontab -e','format','chmod +x']
  }
};

const server = http.createServer(async (req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  const parsedUrl = url.parse(req.url, true);
  const path = parsedUrl.pathname;

  if (path === '/healthz') {
    res.writeHead(200);
    res.end('ok');
    return;
  }

  if (path === '/api/tasks') {
    res.writeHead(200, {'Content-Type': 'application/json'});
    res.end(JSON.stringify(Object.values(taskDetails)));
    return;
  }

  if (path.startsWith('/api/tasks/') && req.method === 'GET') {
    const taskId = path.split('/')[3];
    const task = taskDetails[taskId];
    if (task) {
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(task));
    } else {
      res.writeHead(404, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Task not found'}));
    }
    return;
  }

  if (path === '/api/auth/register' && req.method === 'POST') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      try {
        // Validate body is not empty
        if (!body || body.trim() === '') {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Invalid request format: empty body'}));
          return;
        }

        // Parse JSON
        let data;
        try {
          data = JSON.parse(body);
        } catch (parseErr) {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Invalid request format: malformed JSON'}));
          return;
        }

        // Validate required fields
        if (!data.username || !data.email || !data.password) {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({
            error: 'Invalid request format: missing required fields',
            required: ['username', 'email', 'password']
          }));
          return;
        }

        // Insert user (DB or in-memory)
        let user;
        if (dbConnected) {
          const result = await pgClient.query(
            'INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, username, email, role',
            [data.username, data.email, data.password]
          );
          user = result.rows[0];
        } else {
          // In-memory storage
          if (inMemoryUsers.has(data.username)) {
            res.writeHead(400, {'Content-Type': 'application/json'});
            res.end(JSON.stringify({error: 'Username already exists'}));
            return;
          }

          user = {
            id: userIdCounter++,
            username: data.username,
            email: data.email,
            password: data.password,
            role: 'student'
          };
          inMemoryUsers.set(data.username, user);
        }

        res.writeHead(201, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          token: 'jwt-' + Date.now(),
          user: user
        }));
      } catch (err) {
        console.error('Register error:', err);
        res.writeHead(400, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: err.message || 'Registration failed'}));
      }
    });
    return;
  }

  if (path === '/api/auth/login' && req.method === 'POST') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      console.log('📝 LOGIN request received');
      try {
        // Validate body is not empty
        if (!body || body.trim() === '') {
          console.log('❌ Empty body');
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Invalid request format: empty body'}));
          return;
        }

        // Parse JSON
        let data;
        try {
          data = JSON.parse(body);
          console.log('📦 Login data:', { username: data.username, hasPassword: !!data.password });
        } catch (parseErr) {
          console.log('❌ JSON parse error');
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Invalid request format: malformed JSON'}));
          return;
        }

        // Validate required fields
        if (!data.username || !data.password) {
          console.log('❌ Missing fields');
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({
            error: 'Invalid request format: missing required fields',
            required: ['username', 'password']
          }));
          return;
        }

        // Find user with password
        let user;
        if (dbConnected) {
          const result = await pgClient.query(
            'SELECT id, username, email, password, role FROM users WHERE username = $1',
            [data.username]
          );

          if (result.rows.length === 0) {
            res.writeHead(404, {'Content-Type': 'application/json'});
            res.end(JSON.stringify({
              error: 'User not found',
              message: 'Iltimos avval ro\'yxatdan o\'ting.'
            }));
            return;
          }

          user = result.rows[0];
        } else {
          // In-memory storage
          user = inMemoryUsers.get(data.username);
          if (!user) {
            res.writeHead(404, {'Content-Type': 'application/json'});
            res.end(JSON.stringify({
              error: 'User not found',
              message: 'Iltimos avval ro\'yxatdan o\'ting.'
            }));
            return;
          }
        }

        // Validate password
        if (user.password !== data.password) {
          console.log('❌ Invalid password for user:', data.username);
          res.writeHead(401, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({
            error: 'Invalid password',
            message: 'Parol noto\'g\'ri.'
          }));
          return;
        }

        console.log('✅ Login successful:', data.username);

        // Update last login and online status
        if (dbConnected) {
          await pgClient.query(
            'UPDATE users SET last_login = NOW(), is_online = TRUE WHERE id = $1',
            [user.id]
          );
          // Log activity
          await pgClient.query(
            'INSERT INTO user_activity_logs (user_id, username, action, ip_address, user_agent) VALUES ($1, $2, $3, $4, $5)',
            [user.id, user.username, 'login', req.socket.remoteAddress, req.headers['user-agent']]
          );
        }

        // Remove password from response
        const { password, ...userWithoutPassword } = user;

        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          token: 'jwt-' + Date.now(),
          user: userWithoutPassword
        }));
      } catch (err) {
        console.error('Login error:', err);
        res.writeHead(500, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: err.message || 'Login failed'}));
      }
    });
    return;
  }

  if (path === '/api/progress' && req.method === 'GET') {
    const authHeader = req.headers.authorization;
    if (!authHeader) {
      res.writeHead(401, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Unauthorized'}));
      return;
    }

    // Simple token parsing (jwt-{timestamp})
    pgClient.query(
      `SELECT task_id, MAX(score) as score, MAX(total) as total,
       BOOL_OR(passed) as passed, MAX(completed_at) as completed_at
       FROM progress
       GROUP BY task_id
       ORDER BY MAX(completed_at) DESC`
    ).then(result => {
      const totalScore = result.rows.reduce((sum, row) => sum + (parseInt(row.score) || 0), 0);
      const totalPossible = Object.values(taskDetails).reduce((sum, task) => sum + task.points, 0);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        progress: result.rows,
        totalScore: totalScore,
        totalPossible: totalPossible,
        percentage: Math.round((totalScore / totalPossible) * 100)
      }));
    }).catch(err => {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    });
    return;
  }

  if (path === '/api/sessions' && req.method === 'POST') {
    console.log('🔵 POST /api/sessions - Session creation request received');
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      try {
        console.log('📦 Request body:', body);
        const data = JSON.parse(body);
        console.log('✅ Parsed data:', data);
        const sessionId = 'session-' + (++sessionCounter) + '-' + Date.now();
        const expiresAt = new Date(Date.now() + 3600000);

        // Extract user ID from auth token
        const authHeader = req.headers.authorization;
        console.log('🔑 Auth header:', authHeader);
        let userId = null;

        if (authHeader && authHeader.startsWith('Bearer jwt-')) {
          // Simple user lookup - in production use proper JWT
          const username = req.headers['x-username'];
          if (username) {
            const userResult = await pgClient.query(
              'SELECT id FROM users WHERE username = $1',
              [username]
            );
            if (userResult.rows.length > 0) {
              userId = userResult.rows[0].id;
            }
          }
        }

        console.log('🔥 Creating container for user:', userId);

        let imageName = 'devopslab:latest';
        try {
          await docker.getImage(imageName).inspect();
        } catch {
          imageName = 'ubuntu:22.04';
        }

        const container = await docker.createContainer({
          Image: imageName,
          name: `devops-lab-${sessionId}`,
          Tty: true,
          OpenStdin: true,
          Cmd: ['/bin/bash'],
          Env: ['DEBIAN_FRONTEND=noninteractive', 'TERM=xterm-256color'],
          HostConfig: {
            Memory: 512 * 1024 * 1024,
            CpuShares: 512,
            AutoRemove: false
          }
        });

        await container.start();
        console.log('✅ Container:', container.id.substring(0, 12));

        const session = {
          sessionId: sessionId,
          taskId: data.taskId,
          containerId: container.id,
          userId: userId,
          status: 'active',
          createdAt: new Date().toISOString(),
          expiresAt: expiresAt.toISOString(),
          ttlSeconds: 3600
        };

        sessions.set(sessionId, session);
        containers.set(sessionId, container);

        await pgClient.query(
          'INSERT INTO sessions (session_id, user_id, task_id, container_id, expires_at) VALUES ($1, $2, $3, $4, $5)',
          [sessionId, userId, data.taskId, container.id, expiresAt]
        );

        // Auto cleanup after 1 hour
        setTimeout(async () => {
          console.log('⏰ Auto-cleaning session:', sessionId);
          await cleanupSession(sessionId);
        }, 3600000);

        res.writeHead(201, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(session));
      } catch (err) {
        console.error('❌ Container error:', err.message);
        res.writeHead(500, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: err.message}));
      }
    });
    return;
  }

  if (path.startsWith('/api/sessions/') && req.method === 'DELETE') {
    const sessionId = path.split('/')[3];
    const session = sessions.get(sessionId);

    cleanupSession(sessionId).then(async () => {
      // Log logout activity
      if (session && session.userId && dbConnected) {
        await pgClient.query(
          'UPDATE users SET is_online = FALSE WHERE id = $1',
          [session.userId]
        );
        await pgClient.query(
          'INSERT INTO user_activity_logs (user_id, action) VALUES ($1, $2)',
          [session.userId, 'logout']
        );
      }

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({message: 'Session deleted'}));
    }).catch(err => {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    });
    return;
  }

  // Admin API: Get all users
  if (path === '/api/admin/users' && req.method === 'GET') {
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      const result = await pgClient.query(`
        SELECT u.id, u.username, u.email, u.role, u.created_at, u.last_login, u.is_online,
               COUNT(DISTINCT s.session_id) as session_count,
               COALESCE(SUM(p.score), 0) as total_score
        FROM users u
        LEFT JOIN sessions s ON u.id = s.user_id AND s.status = 'active'
        LEFT JOIN progress p ON u.id = p.user_id
        GROUP BY u.id
        ORDER BY u.created_at DESC
      `);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(result.rows));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Get all sessions
  if (path === '/api/admin/sessions' && req.method === 'GET') {
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      const result = await pgClient.query(`
        SELECT s.session_id, s.user_id, u.username, s.task_id, s.container_id,
               s.status, s.created_at, s.expires_at,
               EXTRACT(EPOCH FROM (s.expires_at - NOW())) as remaining_seconds
        FROM sessions s
        LEFT JOIN users u ON s.user_id = u.id
        WHERE s.status = 'active'
        ORDER BY s.created_at DESC
      `);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(result.rows));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Get user activity logs
  if (path === '/api/admin/activity' && req.method === 'GET') {
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      const result = await pgClient.query(`
        SELECT id, user_id, username, action, ip_address, user_agent, created_at
        FROM user_activity_logs
        ORDER BY created_at DESC
        LIMIT 100
      `);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(result.rows));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Get system statistics
  if (path === '/api/admin/stats' && req.method === 'GET') {
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      const usersResult = await pgClient.query('SELECT COUNT(*) as total, COUNT(CASE WHEN is_online THEN 1 END) as online FROM users');
      const sessionsResult = await pgClient.query('SELECT COUNT(*) as total FROM sessions WHERE status = \'active\'');
      const progressResult = await pgClient.query('SELECT COUNT(*) as completed_tasks, SUM(score) as total_score FROM progress WHERE passed = TRUE');
      const activityResult = await pgClient.query(`
        SELECT action, COUNT(*) as count
        FROM user_activity_logs
        WHERE created_at > NOW() - INTERVAL '24 hours'
        GROUP BY action
      `);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        users: {
          total: parseInt(usersResult.rows[0].total),
          online: parseInt(usersResult.rows[0].online),
          offline: parseInt(usersResult.rows[0].total) - parseInt(usersResult.rows[0].online)
        },
        sessions: {
          active: parseInt(sessionsResult.rows[0].total)
        },
        progress: {
          completedTasks: parseInt(progressResult.rows[0].completed_tasks || 0),
          totalScore: parseInt(progressResult.rows[0].total_score || 0)
        },
        activity24h: activityResult.rows
      }));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Get container resources
  if (path === '/api/admin/resources' && req.method === 'GET') {
    try {
      const containersInfo = [];

      for (const [sessionId, container] of containers.entries()) {
        const session = sessions.get(sessionId);
        try {
          const stats = await container.stats({ stream: false });
          const cpuDelta = stats.cpu_stats.cpu_usage.total_usage - stats.precpu_stats.cpu_usage.total_usage;
          const systemDelta = stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
          const cpuPercent = (cpuDelta / systemDelta) * stats.cpu_stats.online_cpus * 100;

          const memUsage = stats.memory_stats.usage;
          const memLimit = stats.memory_stats.limit;
          const memPercent = (memUsage / memLimit) * 100;

          containersInfo.push({
            sessionId: sessionId,
            containerId: container.id.substring(0, 12),
            userId: session?.userId,
            taskId: session?.taskId,
            cpu: cpuPercent.toFixed(2),
            memory: {
              usage: memUsage,
              limit: memLimit,
              percent: memPercent.toFixed(2)
            },
            network: {
              rx: stats.networks?.eth0?.rx_bytes || 0,
              tx: stats.networks?.eth0?.tx_bytes || 0
            }
          });

          // Save to database
          if (dbConnected) {
            await pgClient.query(
              'INSERT INTO container_stats (session_id, container_id, cpu_usage, memory_usage, memory_limit, network_rx, network_tx) VALUES ($1, $2, $3, $4, $5, $6, $7)',
              [sessionId, container.id, cpuPercent, memUsage, memLimit, stats.networks?.eth0?.rx_bytes || 0, stats.networks?.eth0?.tx_bytes || 0]
            );
          }
        } catch (err) {
          console.error('Error getting stats for container:', sessionId, err.message);
        }
      }

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(containersInfo));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Update user (username, password, role)
  if (path.startsWith('/api/admin/users/') && req.method === 'PUT') {
    const userId = path.split('/')[4];
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      try {
        const data = JSON.parse(body);
        const updates = [];
        const values = [];
        let paramCount = 1;

        if (data.username) {
          updates.push(`username = $${paramCount++}`);
          values.push(data.username);
        }
        if (data.email) {
          updates.push(`email = $${paramCount++}`);
          values.push(data.email);
        }
        if (data.password) {
          updates.push(`password = $${paramCount++}`);
          values.push(data.password);
        }
        if (data.role) {
          updates.push(`role = $${paramCount++}`);
          values.push(data.role);
        }

        if (updates.length === 0) {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'No fields to update'}));
          return;
        }

        values.push(userId);
        const query = `UPDATE users SET ${updates.join(', ')} WHERE id = $${paramCount} RETURNING id, username, email, role`;

        const result = await pgClient.query(query, values);

        if (result.rows.length === 0) {
          res.writeHead(404, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'User not found'}));
          return;
        }

        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          message: 'User updated successfully',
          user: result.rows[0]
        }));
      } catch (err) {
        res.writeHead(500, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: err.message}));
      }
    });
    return;
  }

  // Admin API: Delete user
  if (path.startsWith('/api/admin/users/') && req.method === 'DELETE') {
    const userId = path.split('/')[4];
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      // First, delete user's sessions
      await pgClient.query('DELETE FROM sessions WHERE user_id = $1', [userId]);

      // Delete user's progress
      await pgClient.query('DELETE FROM progress WHERE user_id = $1', [userId]);

      // Delete user's activity logs
      await pgClient.query('DELETE FROM user_activity_logs WHERE user_id = $1', [userId]);

      // Finally, delete the user
      const result = await pgClient.query('DELETE FROM users WHERE id = $1 RETURNING username', [userId]);

      if (result.rows.length === 0) {
        res.writeHead(404, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: 'User not found'}));
        return;
      }

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        message: 'User deleted successfully',
        username: result.rows[0].username
      }));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Get user details
  if (path.startsWith('/api/admin/users/') && path.split('/').length === 5 && req.method === 'GET') {
    const userId = path.split('/')[4];
    if (!dbConnected) {
      res.writeHead(503, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: 'Database not connected'}));
      return;
    }

    try {
      // Get user info
      const userResult = await pgClient.query(
        'SELECT id, username, email, role, created_at, last_login, is_online FROM users WHERE id = $1',
        [userId]
      );

      if (userResult.rows.length === 0) {
        res.writeHead(404, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: 'User not found'}));
        return;
      }

      const user = userResult.rows[0];

      // Get user's sessions
      const sessionsResult = await pgClient.query(
        'SELECT session_id, task_id, container_id, status, created_at, expires_at FROM sessions WHERE user_id = $1 ORDER BY created_at DESC',
        [userId]
      );

      // Get user's progress
      const progressResult = await pgClient.query(
        'SELECT task_id, score, total, passed, completed_at FROM progress WHERE user_id = $1 ORDER BY completed_at DESC',
        [userId]
      );

      // Get user's activity
      const activityResult = await pgClient.query(
        'SELECT action, ip_address, created_at FROM user_activity_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20',
        [userId]
      );

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        user: user,
        sessions: sessionsResult.rows,
        progress: progressResult.rows,
        activity: activityResult.rows
      }));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  // Admin API: Force terminate session
  if (path.startsWith('/api/admin/sessions/') && path.endsWith('/terminate') && req.method === 'POST') {
    const sessionId = path.split('/')[4];

    try {
      await cleanupSession(sessionId);

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        message: 'Session terminated successfully',
        sessionId: sessionId
      }));
    } catch (err) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: err.message}));
    }
    return;
  }

  if (path.startsWith('/api/check/') && req.method === 'POST') {
    const sessionId = path.split('/')[3];
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      const session = sessions.get(sessionId);
      const container = containers.get(sessionId);

      console.log('🔍 Check requested for session:', sessionId);
      console.log('   Session exists:', !!session);
      console.log('   Container exists:', !!container);

      if (!session || !container) {
        res.writeHead(404, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: 'Session not found'}));
        return;
      }

      // Get taskId from request body (current task from frontend)
      let taskId = session.taskId; // default to session task
      try {
        const data = JSON.parse(body);
        if (data.taskId) {
          taskId = data.taskId;
          console.log('📝 Checking task from frontend:', taskId);
        }
      } catch (err) {
        // ignore parse errors, use default
      }

      const task = taskDetails[taskId];
      if (!task) {
        res.writeHead(404, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: 'Task not found'}));
        return;
      }
      console.log('   Task:', task.id);

    if (task.id === 'nginx-health') {
      checkNginxTask(container, task).then(async results => {
        console.log('✅ Check completed:', results.length, 'checks');
        const passedCount = results.filter(r => r.ok).length;
        const score = Math.floor((passedCount / results.length) * task.points);
        const passed = passedCount === results.length;

        // Save progress to database
        if (session.userId) {
          try {
            await pgClient.query(
              'INSERT INTO progress (user_id, task_id, score, total, passed) VALUES ($1, $2, $3, $4, $5)',
              [session.userId, task.id, score, task.points, passed]
            );
            console.log('💾 Progress saved for user:', session.userId);
          } catch (err) {
            console.error('❌ Failed to save progress:', err.message);
          }
        }

        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          passed: passed,
          score: score,
          total: task.points,
          results: results
        }));
      }).catch(err => {
        console.error('❌ Check error:', err.message);
        res.writeHead(500, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({error: err.message}));
      });
    } else {
      const results = task.hints.map((hint, i) => ({
        name: `${i + 1}. ${hint}`,
        ok: false,
        output: 'Not yet implemented'
      }));

      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        passed: false,
        score: 0,
        total: task.points,
        results: results
      }));
    }
    });
    return;
  }

  res.writeHead(404);
  res.end('Not Found');
});

async function cleanupSession(sessionId) {
  const container = containers.get(sessionId);
  if (container) {
    try {
      console.log('🗑️  Stopping container:', sessionId);
      await container.stop({ t: 5 });
      await container.remove();
      console.log('✅ Container removed');
    } catch (err) {
      console.error('❌ Cleanup error:', err.message);
    }
    containers.delete(sessionId);
    sessions.delete(sessionId);

    // Remove from database
    try {
      await pgClient.query('DELETE FROM sessions WHERE session_id = $1', [sessionId]);
      console.log('💾 Session removed from database');
    } catch (err) {
      console.error('❌ Database cleanup error:', err.message);
    }
  }
}

async function checkNginxTask(container, task) {
  console.log('🔍 Starting nginx validation...');
  const results = [];

  // Check 1: nginx installed
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
    console.log('   Check 1 (nginx installed):', hasNginx);
    results.push({
      name: '1. nginx o\'rnatilgan',
      ok: hasNginx,
      output: hasNginx ? '✓ /usr/sbin/nginx' : '✗ nginx not found'
    });
  } catch (err) {
    console.error('   Check 1 error:', err.message);
    results.push({ name: '1. nginx o\'rnatilgan', ok: false, output: 'Error: ' + err.message });
  }

  // Check 2: index.html exists
  try {
    const exec2 = await container.exec({
      Cmd: ['test', '-f', '/var/www/html/index.html'],
      AttachStdout: true
    });
    const stream2 = await exec2.start({ hijack: true });
    await new Promise(resolve => stream2.on('end', resolve));
    const info2 = await exec2.inspect();

    const hasFile = info2.ExitCode === 0;
    console.log('   Check 2 (index.html exists):', hasFile);
    results.push({
      name: '2. index.html fayl yaratilgan',
      ok: hasFile,
      output: hasFile ? '✓ /var/www/html/index.html' : '✗ File not found'
    });
  } catch (err) {
    console.error('   Check 2 error:', err.message);
    results.push({ name: '2. index.html fayl yaratilgan', ok: false, output: 'Error: ' + err.message });
  }

  // Check 3: index.html content
  try {
    const exec3 = await container.exec({
      Cmd: ['cat', '/var/www/html/index.html'],
      AttachStdout: true,
      AttachStderr: true
    });
    const stream3 = await exec3.start({ hijack: true, stdin: false });
    let content = '';
    stream3.on('data', chunk => content += chunk.toString());
    await new Promise(resolve => stream3.on('end', resolve));

    const hasContent = content.length > 0;
    console.log('   Check 3 (file has content):', hasContent, content.substring(0, 50));
    results.push({
      name: '3. /var/www/html/ papkaga joylashtirilgan',
      ok: hasContent,
      output: hasContent ? '✓ Content: ' + content.substring(0, 30) + '...' : '✗ Empty file'
    });
  } catch (err) {
    console.error('   Check 3 error:', err.message);
    results.push({ name: '3. /var/www/html/ papkaga joylashtirilgan', ok: false, output: 'Error: ' + err.message });
  }

  // Check 4: nginx service running
  try {
    const exec4 = await container.exec({
      Cmd: ['service', 'nginx', 'status'],
      AttachStdout: true,
      AttachStderr: true
    });
    const stream4 = await exec4.start({ hijack: true, stdin: false });
    let status = '';
    stream4.on('data', chunk => status += chunk.toString());
    await new Promise(resolve => stream4.on('end', resolve));

    const isRunning = status.includes('running') || status.includes('active');
    console.log('   Check 4 (nginx running):', isRunning);
    results.push({
      name: '4. nginx service ishga tushirilgan',
      ok: isRunning,
      output: isRunning ? '✓ nginx is running' : '✗ Service not running'
    });
  } catch (err) {
    console.error('   Check 4 error:', err.message);
    results.push({ name: '4. nginx service ishga tushirilgan', ok: false, output: 'Error: ' + err.message });
  }

  // Check 5: apt-get update was run
  try {
    const exec5 = await container.exec({
      Cmd: ['test', '-f', '/var/lib/apt/lists/lock'],
      AttachStdout: true
    });
    const stream5 = await exec5.start({ hijack: true });
    await new Promise(resolve => stream5.on('end', resolve));
    const info5 = await exec5.inspect();

    const aptUpdated = info5.ExitCode === 0;
    console.log('   Check 5 (apt updated):', aptUpdated);
    results.push({
      name: '5. apt-get update bajarilgan',
      ok: aptUpdated,
      output: aptUpdated ? '✓ Package lists updated' : '✗ Run apt-get update'
    });
  } catch (err) {
    console.error('   Check 5 error:', err.message);
    results.push({ name: '5. apt-get update bajarilgan', ok: false, output: 'Error: ' + err.message });
  }

  console.log('✅ Validation complete:', results.filter(r => r.ok).length, '/', results.length);
  return results;
}

server.on('upgrade', (request, socket, head) => {
  const pathname = url.parse(request.url).pathname;
  const query = url.parse(request.url, true).query;

  if (pathname === '/ws') {
    const sessionId = query.sessionId;

    if (!sessionId || !sessions.has(sessionId)) {
      socket.write('HTTP/1.1 404 Not Found\r\n\r\n');
      socket.destroy();
      return;
    }

    const acceptKey = request.headers['sec-websocket-key'];
    const hash = crypto.createHash('sha1')
      .update(acceptKey + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11')
      .digest('base64');

    socket.write(
      'HTTP/1.1 101 Switching Protocols\r\n' +
      'Upgrade: websocket\r\n' +
      'Connection: Upgrade\r\n' +
      `Sec-WebSocket-Accept: ${hash}\r\n\r\n`
    );

    const session = sessions.get(sessionId);
    const container = containers.get(sessionId);

    if (!container) {
      sendWS(socket, '\r\n\x1b[31mError: Container not found\x1b[0m\r\n');
      socket.end();
      return;
    }

    container.exec({
      Cmd: ['/bin/bash'],
      AttachStdin: true,
      AttachStdout: true,
      AttachStderr: true,
      Tty: true
    }, (err, exec) => {
      if (err) {
        sendWS(socket, '\r\n\x1b[31mExec error\x1b[0m\r\n');
        socket.end();
        return;
      }

      exec.start({ Tty: true, stdin: true, hijack: true }, (err, stream) => {
        if (err) {
          sendWS(socket, '\r\n\x1b[31mStream error\x1b[0m\r\n');
          socket.end();
          return;
        }

        sendWS(socket, `\r\n\x1b[32m✓ Connected to Container\x1b[0m\r\n`);
        sendWS(socket, `\x1b[36mTask: ${taskDetails[session.taskId].title}\x1b[0m\r\n`);
        sendWS(socket, '\x1b[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m\r\n\r\n');

        stream.on('data', (chunk) => sendWS(socket, chunk.toString('utf8')));

        socket.on('data', (buffer) => {
          try {
            const frame = parseWSFrame(buffer);
            if (!frame) return;
            if (frame.opcode === 0x8) {
              stream.end();
              socket.end();
              return;
            }
            if (frame.opcode === 0x1 || frame.opcode === 0x2) {
              stream.write(frame.payload);
            }
          } catch (err) {
            console.error('WS error:', err.message);
          }
        });

        stream.on('end', () => socket.end());
        socket.on('close', () => stream.end());
      });
    });
  } else {
    socket.destroy();
  }
});

function parseWSFrame(buffer) {
  if (buffer.length < 2) return null;
  const opcode = buffer[0] & 0x0f;
  const masked = buffer[1] & 0x80;
  let len = buffer[1] & 0x7f;
  let maskStart = 2;

  if (len === 126) {
    if (buffer.length < 4) return null;
    len = buffer.readUInt16BE(2);
    maskStart = 4;
  } else if (len === 127) {
    if (buffer.length < 10) return null;
    len = Number(buffer.readBigUInt64BE(2));
    maskStart = 10;
  }

  if (!masked) return { opcode, payload: Buffer.alloc(0) };

  const mask = buffer.slice(maskStart, maskStart + 4);
  const payload = buffer.slice(maskStart + 4, maskStart + 4 + len);
  const decoded = Buffer.alloc(len);
  for (let i = 0; i < len; i++) {
    decoded[i] = payload[i] ^ mask[i % 4];
  }

  return { opcode, payload: decoded };
}

function sendWS(socket, message) {
  const payload = Buffer.from(message);
  const len = payload.length;
  let frame;

  if (len < 126) {
    frame = Buffer.allocUnsafe(2 + len);
    frame[0] = 0x81;
    frame[1] = len;
    payload.copy(frame, 2);
  } else if (len < 65536) {
    frame = Buffer.allocUnsafe(4 + len);
    frame[0] = 0x81;
    frame[1] = 126;
    frame.writeUInt16BE(len, 2);
    payload.copy(frame, 4);
  } else {
    frame = Buffer.allocUnsafe(10 + len);
    frame[0] = 0x81;
    frame[1] = 127;
    frame.writeBigUInt64BE(BigInt(len), 2);
    payload.copy(frame, 10);
  }

  socket.write(frame);
}

server.listen(8080, () => {
  console.log('✅ DevOps Lab Backend v2.0 PRODUCTION');
  console.log('   HTTP: http://localhost:8080');
  console.log('   Features: Real Validation + Auto Cleanup + Colors');
});
