// src/App.tsx
import { useEffect } from "react";
import { Routes, Route, useNavigate, useLocation } from "react-router-dom";
import AuthPage from "./pages/AuthPage";
import HomePage from "./pages/HomePage";
import { isAuthenticated, saveUserSession } from "./utils/utils_auth";

function App() {
  const navigate = useNavigate();
  const location = useLocation();

  // 檢查登入狀態並導航
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const token = params.get("token");
    const id = params.get("id");
    const username = params.get("username");

    if (token && id && username) {
      saveUserSession({ id, username, token });
      // 使用 replace: true 來替換歷史紀錄，避免使用者按上一頁回到帶有 token 的 URL
      navigate("/home", { replace: true });
      return; // 提早結束，避免執行後續邏輯
    }
    // --- 處理所有其他的路由情況 ---
    const isAuth = isAuthenticated();

    // 情況1: 如果使用者「未登入」，但他目前「不在登入頁」，就將他導向登入頁
    if (!isAuth && location.pathname !== "/auth") {
      navigate("/auth");
    }

    // 情況2: 如果使用者「已登入」，但他目前卻在「登入頁」，就將他導向首頁
    if (isAuth && location.pathname === "/auth") {
      navigate("/home");
    }
  }, [navigate, location]);

  

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
