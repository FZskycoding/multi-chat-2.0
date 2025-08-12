// src/hooks/useAuth.ts
import { useState, useCallback, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { getUserSession, clearUserSession } from "../utils/utils_auth";

export const useAuth = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [userSession, setUserSession] = useState(getUserSession());

  useEffect(() => {
    setUserSession(getUserSession());
  }, [location]);

  const handleLogout = useCallback(() => {
    // WebSocket 的關閉邏輯應該由 useWebSocket Hook 自己處理

    clearUserSession();
    setUserSession(null);
    navigate("/auth");
  }, [navigate]);

  return { userSession, handleLogout };
};
