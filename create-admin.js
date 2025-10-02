// Admin user yaratish uchun script
const { Client } = require('pg');

const pgClient = new Client({
  host: process.env.POSTGRES_HOST || 'localhost',
  port: parseInt(process.env.POSTGRES_PORT || '5432'),
  database: process.env.POSTGRES_DB || 'devopslab',
  user: process.env.POSTGRES_USER || 'devopslab',
  password: process.env.POSTGRES_PASSWORD || 'devopslab_secret'
});

async function createAdmin() {
  try {
    await pgClient.connect();
    console.log('✅ PostgreSQL connected');

    // Check if admin already exists
    const checkResult = await pgClient.query(
      'SELECT id, username, role FROM users WHERE username = $1',
      ['admin']
    );

    if (checkResult.rows.length > 0) {
      // Update existing user to admin
      await pgClient.query(
        'UPDATE users SET role = $1 WHERE username = $2',
        ['admin', 'admin']
      );
      console.log('✅ User "admin" updated to admin role');
    } else {
      // Create new admin user
      await pgClient.query(
        'INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4)',
        ['admin', 'admin@devopslab.com', 'admin123', 'admin']
      );
      console.log('✅ Admin user created');
      console.log('   Username: admin');
      console.log('   Password: admin123');
    }

    // Show all admins
    const adminsResult = await pgClient.query(
      'SELECT id, username, email, role FROM users WHERE role = $1',
      ['admin']
    );

    console.log('\n📋 Admin users:');
    adminsResult.rows.forEach(user => {
      console.log(`   - ${user.username} (${user.email}) - ID: ${user.id}`);
    });

    await pgClient.end();
  } catch (err) {
    console.error('❌ Error:', err.message);
    process.exit(1);
  }
}

createAdmin();
