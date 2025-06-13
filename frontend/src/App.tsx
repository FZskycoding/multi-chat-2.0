// src/App.tsx
import React, { useEffect } from "react";
import { Routes, Route, useNavigate } from "react-router-dom";
import AuthPage from "./pages/AuthPage";
import HomePage from "./pages/HomePage";
import { isAuthenticated } from "./utils/auth";

function App() {
  const navigate = useNavigate();

  // 檢查登入狀態並導航
  useEffect(() => {
    if (!isAuthenticated()) {
      navigate("/auth");
    } else {
      navigate("/home");
    }
  }, [navigate]); // 依賴 navigate 確保只運行一次

  return (
    <Routes>
      <Route path="/auth" element={<AuthPage />} />
      <Route path="/home" element={<HomePage />} />
      {/* 任意未匹配路徑導向到認證頁面 (可選，但有助於初始導向) */}
      <Route path="*" element={<AuthPage />} />
    </Routes>
  );
}

export default App;
