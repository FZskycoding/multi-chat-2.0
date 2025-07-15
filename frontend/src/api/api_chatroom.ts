// frontend/src/api/chatroom.ts
import { notifications } from "@mantine/notifications";
import { getUserSession } from "../utils/utils_auth"; // 確保導入正確的路徑
import type { ChatRoom } from "../types/index";
const API_BASE_URL = "http://localhost:8080";

/**
 * 獲取當前使用者所有聊天室的列表
 * @returns {Promise<ChatRoom[]>} 聊天室列表
 */
export async function getUserChatRooms(): Promise<ChatRoom[]> {
  const userSession = getUserSession();
  if (!userSession || !userSession.token) {
    notifications.show({
      title: "錯誤",
      message: "未登入或缺少認證資訊，無法獲取聊天室列表。",
      color: "red",
    });
    return [];
  }

  try {
    const response = await fetch("http://localhost:8080/user-chatrooms", {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${userSession.token}`,
      },
    });

    // 當伺服器連線失敗時
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "無法獲取聊天室列表");
    }

    const data: ChatRoom[] = await response.json();
    return data;
  } catch (error: unknown) {
    // 將 any 改為 unknown
    console.error("Error fetching user chat rooms:", error);
    let errorMessage = "獲取聊天室列表失敗";
    if (error instanceof Error) {
      // 判斷是否為 Error 實例
      errorMessage = `獲取聊天室列表失敗: ${error.message}`;
    }
    notifications.show({
      title: "錯誤",
      message: errorMessage,
      color: "red",
    });
    return [];
  }
}

/**
 * 更新聊天室
 * @param {string} roomId - 聊天室 ID
 * @param {string[]} participantIds - 新的參與者列表
 * @returns {Promise<ChatRoom | null>} 更新後的聊天室物件
 */
export async function updateChatRoom(
  roomId: string,
  participantIds: string[]
): Promise<ChatRoom | null> {
  const userSession = getUserSession();
  if (!userSession || !userSession.token) {
    notifications.show({
      title: "錯誤",
      message: "未登入或缺少認證資訊，無法更新聊天室。",
      color: "red",
    });
    return null;
  }

  try {
    const response = await fetch(`http://localhost:8080/chatrooms/${roomId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${userSession.token}`,
      },
      body: JSON.stringify({ participantIds }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "無法更新聊天室");
    }

    const data: ChatRoom = await response.json();
    return data;
  } catch (error: unknown) {
    console.error("Error updating chat room:", error);
    let errorMessage = "更新聊天室失敗";
    if (error instanceof Error) {
      errorMessage = `更新聊天室失敗: ${error.message}`;
    }
    notifications.show({
      title: "錯誤",
      message: errorMessage,
      color: "red",
    });
    return null;
  }
}

/**
 * 創建或獲取一個聊天室
 * @param {string[]} participantIds - 參與者的 ID 列表 (例如 [使用者A的ID, 使用者B的ID])
 * @param {string} name - 聊天室名稱
 * @returns {Promise<ChatRoom | null>} 創建或獲取的聊天室物件
 */
/**
 * 退出聊天室
 * @param {string} roomId - 要退出的聊天室ID
 * @returns {Promise<boolean>} 退出成功返回true，失敗返回false
 */
export async function leaveChatRoom(roomId: string): Promise<boolean> {
  const userSession = getUserSession();
  if (!userSession || !userSession.token) {
    notifications.show({
      title: "錯誤",
      message: "未登入或缺少認證資訊，無法退出聊天室。",
      color: "red",
    });
    return false;
  }

  try {
    const response = await fetch(`http://localhost:8080/chatrooms/${roomId}/leave`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${userSession.token}`,
      }
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "無法退出聊天室");
    }

    return true;
  } catch (error: unknown) {
    console.error("Error leaving chat room:", error);
    let errorMessage = "退出聊天室失敗";
    if (error instanceof Error) {
      errorMessage = `退出聊天室失敗: ${error.message}`;
    }
    notifications.show({
      title: "錯誤",
      message: errorMessage,
      color: "red",
    });
    return false;
  }
}

export async function createOrGetChatRoom(
  participantIds: string[],
  name: string
): Promise<ChatRoom | null> {
  const userSession = getUserSession();
  if (!userSession || !userSession.token) {
    notifications.show({
      title: "錯誤",
      message: "未登入或缺少認證資訊，無法建立聊天室。",
      color: "red",
    });
    return null;
  }

  try {
    const response = await fetch("http://localhost:8080/chatrooms", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${userSession.token}`,
      },
      body: JSON.stringify({
        name,
        participantIds,
      }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "無法建立或獲取聊天室");
    }

    const data: ChatRoom = await response.json();
    return data;
  } catch (error: unknown) {
    // 將 any 改為 unknown
    console.error("Error creating or getting chat room:", error);
    let errorMessage = "建立/獲取聊天室失敗";
    if (error instanceof Error) {
      // 判斷是否為 Error 實例
      errorMessage = `建立/獲取聊天室失敗: ${error.message}`;
    }
    notifications.show({
      title: "錯誤",
      message: errorMessage,
      color: "red",
    });
    return null;
  }
}

// 新增邀請參與者到聊天室的 API 函數
export const addParticipantsToChatRoom = async (
  roomId: string,
  newParticipantIds: string[]
): Promise<ChatRoom> => {
  // 從 getUserSession 獲取整個使用者會話物件，其中包含 token
  const userSession = getUserSession();

  // 檢查 userSession 和 token 是否存在
  if (!userSession || !userSession.token) {
    // 抛出更具描述性的錯誤
    throw new Error("Authentication token not found. Please log in.");
  }

  const response = await fetch(
    `${API_BASE_URL}/chatrooms/${roomId}/participants`,
    {
      method: "PUT", // 後端定義的是 PUT 方法
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${userSession.token}`, // <-- 使用從 userSession 獲取的 token
      },
      body: JSON.stringify({ newParticipantIds }), // 注意這裡的 key 必須與後端 AddParticipantsRequest 匹配
    }
  );

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.message || "Failed to add participants.");
  }

  return response.json(); // 後端應返回更新後的 ChatRoom 物件
};
