// src/api/user.ts
import { notifications } from "@mantine/notifications";
import { getUserSession } from "../utils/auth";

interface User {
  id: string;
  username: string;
  email: string;
}

//回傳一個裝著 User 陣列的 Promise
export const getAllUsers = async (): Promise<User[]> => {
  try {
    // 獲取用戶 session
    const session = getUserSession();
    if (!session?.token) {
      throw new Error("未登入或 token 無效");
    }

    const response = await fetch("http://localhost:8080/users", {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${session.token}`,
      },
    });

    if (!response.ok) {
      const errorData = await response.json();
      notifications.show({
        title: "載入使用者失敗",
        message: errorData.message || "無法載入所有使用者列表",
        color: "red",
        autoClose: 2000
      });
      return [];
    }

    const data: User[] = await response.json();
    return data;
  } catch (error) {
    console.error("獲取所有使用者請求錯誤:", error);
    notifications.show({
      title: "網路錯誤",
      message: "無法連線到伺服器，請檢查網路。",
      color: "red",
      autoClose: 2000
    });
    return [];
  }
};
