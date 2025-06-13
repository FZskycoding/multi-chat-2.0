// src/utils/auth.ts

interface UserSession {
  id: string;
  username: string;
  // 可以根據需求添加其他資訊，例如 token
}

const USER_SESSION_KEY = "user_session";

export const saveUserSession = (session: UserSession) => {
  localStorage.setItem(USER_SESSION_KEY, JSON.stringify(session));
};

export const getUserSession = (): UserSession | null => {
  const sessionString = localStorage.getItem(USER_SESSION_KEY);
  if (sessionString) {
    try {
      return JSON.parse(sessionString);
    } catch (e) {
      console.error("Failed to parse user session from localStorage", e);
      return null;
    }
  }
  return null;
};

export const clearUserSession = () => {
  localStorage.removeItem(USER_SESSION_KEY);
};

export const isAuthenticated = (): boolean => {
  return getUserSession() !== null;
};
