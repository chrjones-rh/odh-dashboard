import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import NotFound from './components/NotFound';
import MainPage from './pages/MainPage';
import GuardrailsPage from './pages/GuardrailsPage/GuardrailsPage';

const AppRoutes: React.FC = () => (
  <Routes>
    <Route path="/" element={<Navigate to="/main-view" replace />} />
    <Route path="/main-view/*" element={<MainPage />} />
    <Route path="/guardrails" element={<GuardrailsPage />} />
    <Route path="*" element={<NotFound />} />
  </Routes>
);

export default AppRoutes;
