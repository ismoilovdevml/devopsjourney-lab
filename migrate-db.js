// Database migration - yangi column'larni qo'shish
const { Client } = require('pg');

const pgClient = new Client({
  host: process.env.POSTGRES_HOST || 'localhost',
  port: parseInt(process.env.POSTGRES_PORT || '5432'),
  database: process.env.POSTGRES_DB || 'devopslab',
  user: process.env.POSTGRES_USER || 'devopslab',
  password: process.env.POSTGRES_PASSWORD || 'devopslab_secret'
});

async function migrate() {
  try {
    await pgClient.connect();
    console.log('✅ PostgreSQL connected');

    // Add columns to users table
    console.log('\n📝 Adding columns to users table...');
    try {
      await pgClient.query('ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login TIMESTAMP');
      console.log('  ✓ last_login column added');
    } catch (err) {
      console.log('  - last_login:', err.message);
    }

    try {
      await pgClient.query('ALTER TABLE users ADD COLUMN IF NOT EXISTS is_online BOOLEAN DEFAULT FALSE');
      console.log('  ✓ is_online column added');
    } catch (err) {
      console.log('  - is_online:', err.message);
    }

    // Create user_activity_logs table
    console.log('\n📝 Creating user_activity_logs table...');
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
    console.log('  ✓ user_activity_logs table created');

    // Create container_stats table
    console.log('\n📝 Creating container_stats table...');
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
    console.log('  ✓ container_stats table created');

    // Show tables
    console.log('\n📋 Current tables:');
    const result = await pgClient.query(`
      SELECT table_name
      FROM information_schema.tables
      WHERE table_schema = 'public'
      ORDER BY table_name
    `);
    result.rows.forEach(row => console.log('  -', row.table_name));

    console.log('\n✅ Migration completed successfully!');

    await pgClient.end();
  } catch (err) {
    console.error('❌ Migration error:', err.message);
    process.exit(1);
  }
}

migrate();
