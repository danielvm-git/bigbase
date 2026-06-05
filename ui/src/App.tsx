import { Routes, Route } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import DataStudioPage from './pages/DataStudioPage'
import SqlEditorPage from './pages/SqlEditorPage'
import UsersPage from './pages/UsersPage'
import GitReposPage from './pages/GitReposPage'
import DeployPage from './pages/DeployPage'
import MessagingPage from './pages/MessagingPage'
import MessagingDetailPage from './pages/MessagingDetailPage'
import StoragePage from './pages/StoragePage'
import FunctionsPage from './pages/FunctionsPage'
import FunctionDetailPage, { FunctionLogsRedirect } from './pages/FunctionDetailPage'
import ForgePage from './pages/ForgePage'
import CiciPage from './pages/CiciPage'
import CreateSitePage from './pages/CreateSitePage'
import MonitoringPage from './pages/MonitoringPage'
import EventsPage from './pages/EventsPage'
import SiteDetailPage from './pages/SiteDetailPage'
import RealtimePage from './pages/RealtimePage'
import SettingsPage from './pages/SettingsPage'
import NotFoundPage from './pages/NotFoundPage'
import Layout from './Layout'

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<DashboardPage />} />
        <Route path="data" element={<DataStudioPage />} />
        <Route path="sql" element={<SqlEditorPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="repos" element={<GitReposPage />} />
        <Route path="deploy" element={<DeployPage />} />
        <Route path="deploy/new" element={<CreateSitePage />} />
        <Route path="deploy/:siteId" element={<SiteDetailPage />} />
        <Route path="messaging" element={<MessagingPage />} />
        <Route path="messaging/:id" element={<MessagingDetailPage />} />
        <Route path="storage" element={<StoragePage />} />
        <Route path="functions" element={<FunctionsPage />} />
        <Route path="functions/:id" element={<FunctionDetailPage />} />
        <Route path="functions/:id/logs" element={<FunctionLogsRedirect />} />
        <Route path="forge" element={<ForgePage />} />
        <Route path="cici" element={<CiciPage />} />
        <Route path="monitoring" element={<MonitoringPage />} />
        <Route path="events" element={<EventsPage />} />
        <Route path="realtime" element={<RealtimePage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
      <Route path="login" element={<LoginPage />} />
    </Routes>
  )
}

export default App
