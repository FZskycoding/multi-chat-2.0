// src/api/auth.ts
import { notifications } from "@mantine/notifications";

const API_BASE_URL = "http://localhost:8080";

interface RegisterPayload {
  email: string;
  username: string;
  password: string;
}

interface LoginPayload {
  email: string;
  password: string;
}

interface AuthResponse {
  message: string;
  id?: string;
  username?: string;
}

// 註冊
export async function register(
  payload: RegisterPayload
): Promise<AuthResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    const data = await response.json();

    if (!response.ok) {
      notifications.show({
        title: "註冊失敗",
        message: data.message || "發生未知錯誤",
        color: "red",
        autoClose: 2000
      });
      throw new Error(data.message || "註冊失敗");
    }

    // 顯示註冊成功通知
    notifications.show({
      title: "註冊成功",
      message: "帳號已建立，請登入！",
      color: "green",
      autoClose: 1500,
    });
    return data;
  } catch (error) {
    // `TypeError` 通常表示網路錯誤 (例如 CORS 錯誤、離線等)
    if (error instanceof TypeError) {
      notifications.show({
        title: "網路錯誤",
        message: "無法連線到伺服器，請檢查網路。",
        color: "red",
        autoClose: 2000,
      });
    }
    throw error;
  }
}

//登入
export async function login(payload: LoginPayload): Promise<AuthResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    const data = await response.json();

    if (!response.ok) {
      notifications.show({
        title: "登入失敗",
        message: data.message || "電子郵件或密碼不正確",
        color: "red",
        autoClose: 2000,
      });
      throw new Error(data.message || "登入失敗");
    }

    notifications.show({
      title: "登入成功",
      message: `歡迎回來，${data.username}！`,
      color: "green",
      autoClose: 1500,
    });
    return data;
  } catch (error) {
    // 只有當它是真正的網路連線錯誤 (例如 fetch 失敗，沒有收到 response) 時才顯示
    if (error instanceof TypeError) {
      notifications.show({
        title: "網路錯誤",
        message: "無法連線到伺服器，請檢查網路。",
        color: "red",
        autoClose: 2000
      });
    }
    throw error; 
  }
}
