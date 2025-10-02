const express = require('express');
const cors = require('cors');
const app = express();

app.use(cors());
app.use(express.json());

app.get('/healthz', (req, res) => res.send('ok'));

app.get('/api/tasks', (req, res) => {
  res.json([
    { id: 'nginx-health', title: 'Test 1: Nginx', order: 1, unlocked: true, completed: false },
    { id: 'docker-basics', title: 'Test 2: Docker', order: 2, unlocked: false, completed: false }
  ]);
});

app.listen(8080, () => console.log('Test backend on :8080'));
