// src/App.tsx
import { useEffect } from "react";
import { Routes, Route, useNavigate, useLocation } from "react-router-dom";
import AuthPage from "./pages/AuthPage";
import HomePage from "./pages/HomePage";
import { isAuthenticated, saveUserSession } from "./utils/utils_auth";
import Cookies from "js-cookie"; 

function App() {
  const navigate = useNavigate();
  const location = useLocation();

  // 檢查登入狀態並導航
  useEffect(() => {
    // const params = new URLSearchParams(location.search);
    // const token = params.get("token");
    // const id = params.get("id");
    // const username = params.get("username");
    const userInfoCookie = Cookies.get("user_info");
    if (userInfoCookie) {
      try {
        const decodedCookie = decodeURIComponent(userInfoCookie)
        const userInfo = JSON.parse(decodedCookie);
        const { id, username } = userInfo;
        if (id && username) {
          // 如果 cookie 存在且內容有效，就儲存 session
          saveUserSession({ id, username: decodeURIComponent(username) }); // 對 username 再次解碼
          // 刪除 user_info cookie，因為它的任務已經完成
          Cookies.remove("user_info");
          // 清理 URL 並導航，這一步保持不變
          navigate("/home", { replace: true });
          return;
        }
      } catch (e) {
        console.error("Failed to parse user_info cookie", e);
        Cookies.remove("user_info"); // 如果解析失敗，也刪除壞掉的 cookie
      }
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
