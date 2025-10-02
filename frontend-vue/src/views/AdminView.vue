<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 text-white p-6">
    <ToastNotification />
    <!-- Header -->
    <div class="max-w-7xl mx-auto mb-8">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-4xl font-bold bg-gradient-to-r from-purple-400 to-pink-600 bg-clip-text text-transparent">
            Admin Dashboard
          </h1>
          <p class="text-gray-400 mt-2">DevOps Lab - Real-time Monitoring & Management</p>
        </div>
        <div class="flex items-center gap-3">
          <ThemeToggle />
          <router-link to="/lab" class="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors">
            ← Back to Lab
          </router-link>
          <button
            @click="refreshAll"
            :disabled="loading"
            class="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg transition-colors disabled:opacity-50"
          >
            <RefreshCw :class="{ 'animate-spin': loading }" class="w-5 h-5" />
            Refresh
          </button>
        </div>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <div class="bg-slate-800/50 backdrop-blur-sm border border-purple-500/20 rounded-xl p-6">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm">Total Users</p>
            <p class="text-3xl font-bold mt-1">{{ stats.users.total }}</p>
          </div>
          <Users class="w-12 h-12 text-purple-500" />
        </div>
      </div>

      <div class="bg-slate-800/50 backdrop-blur-sm border border-green-500/20 rounded-xl p-6">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm">Online Users</p>
            <p class="text-3xl font-bold mt-1 text-green-400">{{ stats.users.online }}</p>
          </div>
          <Activity class="w-12 h-12 text-green-500" />
        </div>
      </div>

      <div class="bg-slate-800/50 backdrop-blur-sm border border-blue-500/20 rounded-xl p-6">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm">Active Sessions</p>
            <p class="text-3xl font-bold mt-1 text-blue-400">{{ stats.sessions.active }}</p>
          </div>
          <Server class="w-12 h-12 text-blue-500" />
        </div>
      </div>

      <div class="bg-slate-800/50 backdrop-blur-sm border border-pink-500/20 rounded-xl p-6">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm">Completed Tasks</p>
            <p class="text-3xl font-bold mt-1 text-pink-400">{{ stats.progress.completedTasks }}</p>
          </div>
          <CheckCircle class="w-12 h-12 text-pink-500" />
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="max-w-7xl mx-auto mb-6">
      <div class="flex gap-4 border-b border-gray-700">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="[
            'px-4 py-2 font-medium transition-colors',
            activeTab === tab.id
              ? 'text-purple-400 border-b-2 border-purple-400'
              : 'text-gray-400 hover:text-gray-200'
          ]"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <!-- Tab Content -->
    <div class="max-w-7xl mx-auto">
      <!-- Users Tab -->
      <div v-if="activeTab === 'users'" class="bg-slate-800/50 backdrop-blur-sm border border-purple-500/20 rounded-xl overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead class="bg-slate-900/50">
              <tr>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">ID</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Username</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Email</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Role</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Status</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Sessions</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Score</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Last Login</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-700">
              <tr v-for="user in users" :key="user.id" class="hover:bg-slate-700/30 transition-colors">
                <td class="px-6 py-4 text-sm">{{ user.id }}</td>
                <td class="px-6 py-4 text-sm font-medium">{{ user.username }}</td>
                <td class="px-6 py-4 text-sm text-gray-400">{{ user.email }}</td>
                <td class="px-6 py-4 text-sm">
                  <span :class="[
                    'px-2 py-1 rounded-full text-xs font-medium',
                    user.role === 'admin' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'
                  ]">
                    {{ user.role }}
                  </span>
                </td>
                <td class="px-6 py-4 text-sm">
                  <span :class="[
                    'flex items-center gap-2',
                    user.is_online ? 'text-green-400' : 'text-gray-500'
                  ]">
                    <span class="w-2 h-2 rounded-full" :class="user.is_online ? 'bg-green-400 animate-pulse' : 'bg-gray-500'"></span>
                    {{ user.is_online ? 'Online' : 'Offline' }}
                  </span>
                </td>
                <td class="px-6 py-4 text-sm">{{ user.session_count }}</td>
                <td class="px-6 py-4 text-sm font-semibold text-purple-400">{{ user.total_score }}</td>
                <td class="px-6 py-4 text-sm text-gray-400">{{ formatDate(user.last_login) }}</td>
                <td class="px-6 py-4 text-sm">
                  <div class="flex items-center gap-2">
                    <button
                      @click="viewUserDetails(user)"
                      class="p-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-all hover:scale-110"
                      title="View Details"
                    >
                      <Eye class="w-4 h-4" />
                    </button>
                    <button
                      @click="editUser(user)"
                      class="p-2 bg-yellow-600 hover:bg-yellow-700 rounded-lg transition-all hover:scale-110"
                      title="Edit User"
                    >
                      <Edit class="w-4 h-4" />
                    </button>
                    <button
                      @click="confirmDeleteUser(user)"
                      class="p-2 bg-red-600 hover:bg-red-700 rounded-lg transition-all hover:scale-110"
                      title="Delete User"
                    >
                      <Trash2 class="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Sessions Tab -->
      <div v-if="activeTab === 'sessions'" class="bg-slate-800/50 backdrop-blur-sm border border-purple-500/20 rounded-xl overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead class="bg-slate-900/50">
              <tr>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Session ID</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">User</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Task</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Container</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Status</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Remaining</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Created</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-700">
              <tr v-for="session in sessions" :key="session.session_id" class="hover:bg-slate-700/30 transition-colors">
                <td class="px-6 py-4 text-sm font-mono text-purple-400">{{ session.session_id.substring(0, 20) }}...</td>
                <td class="px-6 py-4 text-sm font-medium">{{ session.username || 'N/A' }}</td>
                <td class="px-6 py-4 text-sm">{{ session.task_id }}</td>
                <td class="px-6 py-4 text-sm font-mono text-blue-400">{{ session.container_id.substring(0, 12) }}</td>
                <td class="px-6 py-4 text-sm">
                  <span class="px-2 py-1 bg-green-500/20 text-green-400 rounded-full text-xs font-medium">
                    {{ session.status }}
                  </span>
                </td>
                <td class="px-6 py-4 text-sm text-gray-400">{{ formatDuration(session.remaining_seconds) }}</td>
                <td class="px-6 py-4 text-sm text-gray-400">{{ formatDate(session.created_at) }}</td>
                <td class="px-6 py-4 text-sm">
                  <button
                    @click="confirmTerminateSession(session)"
                    class="p-2 bg-red-600 hover:bg-red-700 rounded-lg transition-all hover:scale-110"
                    title="Terminate Session"
                  >
                    <XCircle class="w-4 h-4" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Activity Tab -->
      <div v-if="activeTab === 'activity'" class="bg-slate-800/50 backdrop-blur-sm border border-purple-500/20 rounded-xl overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead class="bg-slate-900/50">
              <tr>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">ID</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Username</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Action</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Details</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">IP Address</th>
                <th class="px-6 py-4 text-left text-sm font-semibold text-gray-300">Time</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-700">
              <tr v-for="log in activityLogs" :key="log.id" class="hover:bg-slate-700/30 transition-colors">
                <td class="px-6 py-4 text-sm">{{ log.id }}</td>
                <td class="px-6 py-4 text-sm font-medium">{{ log.username || 'N/A' }}</td>
                <td class="px-6 py-4 text-sm">
                  <span :class="[
                    'px-2 py-1 rounded-full text-xs font-medium',
                    log.action === 'login' ? 'bg-green-500/20 text-green-400' :
                    log.action.startsWith('admin') ? 'bg-red-500/20 text-red-400' :
                    'bg-blue-500/20 text-blue-400'
                  ]">
                    {{ log.action }}
                  </span>
                </td>
                <td class="px-6 py-4 text-sm text-gray-400 truncate max-w-xs">{{ log.details || 'N/A' }}</td>
                <td class="px-6 py-4 text-sm font-mono text-gray-400">{{ log.ip_address || 'N/A' }}</td>
                <td class="px-6 py-4 text-sm text-gray-400">{{ formatDate(log.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Resources Tab -->
      <div v-if="activeTab === 'resources'" class="space-y-6">
        <div v-if="resources.length === 0" class="text-center py-12 text-gray-400">
          <Server class="w-16 h-16 mx-auto mb-4 opacity-50" />
          <p>No active containers / Coming soon...</p>
        </div>
      </div>
    </div>

    <!-- User Details Modal -->
    <div v-if="showUserModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="showUserModal = false">
      <div class="bg-slate-800 rounded-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto shadow-2xl">
        <div class="p-6 border-b border-gray-700 sticky top-0 bg-slate-800 z-10">
          <div class="flex items-center justify-between">
            <h2 class="text-2xl font-bold">User Details: {{ selectedUser?.user?.username }}</h2>
            <button @click="showUserModal = false" class="p-2 hover:bg-slate-700 rounded-lg transition-colors">
              <X class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div class="p-6 space-y-6" v-if="selectedUser">
          <!-- User Info -->
          <div class="bg-slate-900/50 rounded-lg p-4 border border-gray-700">
            <h3 class="font-semibold mb-3 flex items-center gap-2">
              <Users class="w-5 h-5" />
              Basic Information
            </h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <p class="text-sm text-gray-400">Username</p>
                <p class="font-medium">{{ selectedUser.user.username }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-400">Email</p>
                <p class="font-medium">{{ selectedUser.user.email }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-400">Role</p>
                <p class="font-medium">
                  <span :class="[
                    'px-2 py-1 rounded-full text-xs',
                    selectedUser.user.role === 'admin' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'
                  ]">
                    {{ selectedUser.user.role }}
                  </span>
                </p>
              </div>
              <div>
                <p class="text-sm text-gray-400">Created</p>
                <p class="font-medium">{{ formatDate(selectedUser.user.created_at) }}</p>
              </div>
            </div>
          </div>

          <!-- Sessions -->
          <div class="bg-slate-900/50 rounded-lg p-4 border border-gray-700">
            <h3 class="font-semibold mb-3 flex items-center gap-2">
              <Server class="w-5 h-5" />
              Sessions ({{ selectedUser.sessions.length }})
            </h3>
            <div class="space-y-2 max-h-60 overflow-y-auto">
              <div v-for="sess in selectedUser.sessions" :key="sess.session_id" class="bg-slate-800 p-3 rounded border border-gray-700">
                <div class="flex justify-between text-sm">
                  <span class="font-mono text-purple-400">{{ sess.session_id || sess.id }}</span>
                  <span :class="sess.status === 'active' ? 'text-green-400' : 'text-gray-400'">{{ sess.status }}</span>
                </div>
                <div class="text-xs text-gray-400 mt-1">
                  Task: {{ sess.task_id }} | Created: {{ formatDate(sess.created_at) }}
                </div>
              </div>
              <div v-if="selectedUser.sessions.length === 0" class="text-center text-gray-500 py-4">
                No sessions
              </div>
            </div>
          </div>

          <!-- Progress -->
          <div class="bg-slate-900/50 rounded-lg p-4 border border-gray-700">
            <h3 class="font-semibold mb-3 flex items-center gap-2">
              <CheckCircle class="w-5 h-5" />
              Progress ({{ selectedUser.progress.length }} tasks)
            </h3>
            <div class="space-y-2 max-h-60 overflow-y-auto">
              <div v-for="prog in selectedUser.progress" :key="prog.id" class="bg-slate-800 p-3 rounded border border-gray-700">
                <div class="flex justify-between text-sm">
                  <span class="font-medium">{{ prog.task_id }}</span>
                  <span :class="prog.passed ? 'text-green-400' : 'text-red-400'">
                    {{ prog.score }}/{{ prog.total }} {{ prog.passed ? '✓' : '✗' }}
                  </span>
                </div>
                <div class="text-xs text-gray-400 mt-1">
                  Completed: {{ formatDate(prog.completed_at) }}
                </div>
              </div>
              <div v-if="selectedUser.progress.length === 0" class="text-center text-gray-500 py-4">
                No progress yet
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit User Modal -->
    <div v-if="showEditModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="showEditModal = false">
      <div class="bg-slate-800 rounded-xl max-w-md w-full shadow-2xl">
        <div class="p-6 border-b border-gray-700">
          <div class="flex items-center justify-between">
            <h2 class="text-2xl font-bold flex items-center gap-2">
              <Edit class="w-6 h-6" />
              Edit User: {{ editingUser?.username }}
            </h2>
            <button @click="showEditModal = false" class="p-2 hover:bg-slate-700 rounded-lg transition-colors">
              <X class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div class="p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Username</label>
            <input
              v-model="editForm.username"
              type="text"
              class="w-full px-4 py-2 bg-slate-900 border border-gray-700 rounded-lg focus:border-purple-500 focus:outline-none transition-colors"
              placeholder="Enter username"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Email</label>
            <input
              v-model="editForm.email"
              type="email"
              class="w-full px-4 py-2 bg-slate-900 border border-gray-700 rounded-lg focus:border-purple-500 focus:outline-none transition-colors"
              placeholder="Enter email"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Password (leave empty to keep current)</label>
            <input
              v-model="editForm.password"
              type="password"
              class="w-full px-4 py-2 bg-slate-900 border border-gray-700 rounded-lg focus:border-purple-500 focus:outline-none transition-colors"
              placeholder="Enter new password"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Role</label>
            <select
              v-model="editForm.role"
              class="w-full px-4 py-2 bg-slate-900 border border-gray-700 rounded-lg focus:border-purple-500 focus:outline-none transition-colors"
            >
              <option value="student">Student</option>
              <option value="admin">Admin</option>
            </select>
          </div>

          <div class="flex gap-3 pt-4">
            <button
              @click="saveUser"
              :disabled="saving"
              class="flex-1 px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg transition-colors disabled:opacity-50 font-semibold"
            >
              {{ saving ? 'Saving...' : 'Save Changes' }}
            </button>
            <button
              @click="showEditModal = false"
              class="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors font-semibold"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirmation Dialog -->
    <div v-if="showConfirmDialog" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="showConfirmDialog = false">
      <div class="bg-slate-800 rounded-xl max-w-md w-full shadow-2xl">
        <div class="p-6">
          <div class="flex items-center gap-3 mb-4">
            <div class="p-3 bg-yellow-500/20 rounded-full">
              <AlertTriangle class="w-8 h-8 text-yellow-500" />
            </div>
            <h2 class="text-xl font-bold">{{ confirmDialog.title }}</h2>
          </div>
          <p class="text-gray-300 mb-6 leading-relaxed">{{ confirmDialog.message }}</p>
          <div class="flex gap-3">
            <button
              @click="confirmDialog.onConfirm"
              :disabled="saving"
              class="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg transition-colors disabled:opacity-50 font-semibold"
            >
              {{ saving ? 'Processing...' : 'Confirm' }}
            </button>
            <button
              @click="showConfirmDialog = false"
              class="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors font-semibold"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Users, Activity, Server, CheckCircle, RefreshCw, Eye, Edit, Trash2, XCircle, X, AlertTriangle } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { api } from '@/services/api'
import ToastNotification from '@/components/ToastNotification.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'

const { success, error } = useToast()

const activeTab = ref('users')
const loading = ref(false)
const saving = ref(false)

const stats = ref({
  users: { total: 0, online: 0, offline: 0 },
  sessions: { active: 0 },
  progress: { completedTasks: 0, totalScore: 0 },
  activity24h: []
})

const users = ref<any[]>([])
const sessions = ref<any[]>([])
const activityLogs = ref<any[]>([])
const resources = ref<any[]>([])

const showUserModal = ref(false)
const showEditModal = ref(false)
const showConfirmDialog = ref(false)
const selectedUser = ref<any>(null)
const editingUser = ref<any>(null)
const editForm = ref({
  username: '',
  email: '',
  password: '',
  role: 'student'
})
const confirmDialog = ref({
  title: '',
  message: '',
  onConfirm: () => {}
})

const tabs = [
  { id: 'users', label: 'Users' },
  { id: 'sessions', label: 'Active Sessions' },
  { id: 'activity', label: 'Activity Logs' },
  { id: 'resources', label: 'Resources' }
]

const fetchStats = async () => {
  try {
    stats.value = await api.adminStats()
  } catch (err) {
    console.error('Failed to fetch stats:', err)
  }
}

const fetchUsers = async () => {
  try {
    users.value = await api.adminUsers()
  } catch (err) {
    console.error('Failed to fetch users:', err)
  }
}

const fetchSessions = async () => {
  try {
    sessions.value = await api.adminSessions()
  } catch (err) {
    console.error('Failed to fetch sessions:', err)
  }
}

const fetchActivity = async () => {
  try {
    activityLogs.value = await api.adminActivity()
  } catch (err) {
    console.error('Failed to fetch activity:', err)
  }
}

const fetchResources = async () => {
  try {
    resources.value = await api.adminResources()
  } catch (err) {
    console.error('Failed to fetch resources:', err)
  }
}

const viewUserDetails = async (user: any) => {
  try {
    selectedUser.value = await api.adminUserDetails(user.id)
    showUserModal.value = true
  } catch (err) {
    console.error('Failed to fetch user details:', err)
    error('Failed to load user details')
  }
}

const editUser = (user: any) => {
  editingUser.value = user
  editForm.value = {
    username: user.username,
    email: user.email,
    password: '',
    role: user.role
  }
  showEditModal.value = true
}

const saveUser = async () => {
  if (!editingUser.value) return

  saving.value = true
  try {
    const updateData: any = {
      username: editForm.value.username,
      email: editForm.value.email,
      role: editForm.value.role
    }

    if (editForm.value.password) {
      updateData.password = editForm.value.password
    }

    await api.adminUpdateUser(editingUser.value.id, updateData)

    showEditModal.value = false
    await refreshAll()
    success('User updated successfully')
  } catch (err: any) {
    console.error('Failed to update user:', err)
    error('Failed to update user: ' + (err.response?.data?.error || err.message))
  } finally {
    saving.value = false
  }
}

const confirmDeleteUser = (user: any) => {
  confirmDialog.value = {
    title: 'Delete User',
    message: `Are you sure you want to delete user "${user.username}"? This action cannot be undone and will delete all user data including sessions, progress, and activity logs.`,
    onConfirm: () => deleteUser(user)
  }
  showConfirmDialog.value = true
}

const deleteUser = async (user: any) => {
  saving.value = true
  try {
    await api.adminDeleteUser(user.id)
    showConfirmDialog.value = false
    await refreshAll()
    success('User deleted successfully')
  } catch (err: any) {
    console.error('Failed to delete user:', err)
    error('Failed to delete user: ' + (err.response?.data?.error || err.message))
  } finally {
    saving.value = false
  }
}

const confirmTerminateSession = (session: any) => {
  confirmDialog.value = {
    title: 'Terminate Session',
    message: `Are you sure you want to terminate session for user "${session.username}"? This will stop their container immediately.`,
    onConfirm: () => terminateSession(session)
  }
  showConfirmDialog.value = true
}

const terminateSession = async (session: any) => {
  saving.value = true
  try {
    await api.adminTerminateSession(session.session_id)
    showConfirmDialog.value = false
    await refreshAll()
    success('Session terminated successfully')
  } catch (err: any) {
    console.error('Failed to terminate session:', err)
    error('Failed to terminate session: ' + (err.response?.data?.error || err.message))
  } finally {
    saving.value = false
  }
}

const refreshAll = async () => {
  loading.value = true
  await Promise.all([
    fetchStats(),
    fetchUsers(),
    fetchSessions(),
    fetchActivity(),
    fetchResources()
  ])
  loading.value = false
}

const formatDate = (date: string) => {
  if (!date) return 'N/A'
  return new Date(date).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatDuration = (seconds: number) => {
  if (!seconds || seconds < 0) return 'Expired'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}m ${secs}s`
}

let refreshInterval: number | null = null

onMounted(() => {
  refreshAll()
  refreshInterval = setInterval(refreshAll, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>
