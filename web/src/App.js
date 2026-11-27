import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import Layout from './components/Layout';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Groups from './pages/Groups';
import Dashboard from './pages/Dashboard';
import ProjectLayout from './pages/ProjectLayout';
import Board from './pages/Board';
import ProjectGroups from './pages/ProjectGroups';
import Activity from './pages/Activity';
import ProjectUsers from './pages/ProjectUsers';
import InvitePage from './pages/InvitePage';
import LandingPage from './landing/LandingPage';

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />
          <Route path="/invite" element={<InvitePage />} />
          <Route element={<Layout />}>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/groups" element={<Groups />} />
            <Route path="/projects/:id" element={<ProjectLayout />}>
              <Route index element={<Board />} />
              <Route path="tasks/:taskId" element={<Board />} />
              <Route path="groups" element={<ProjectGroups />} />
              <Route path="users" element={<ProjectUsers />} />
              <Route path="activity" element={<Activity />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
