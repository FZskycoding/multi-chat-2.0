export interface ChatRoom {
  id: string;
  name: string;
  creatorId: string;
  participants: string[];
  createdAt: string;
  updatedAt: string; // Add updatedAt
}

export interface User {
  id: string;
  username: string;
  email?: string; // 將 email 設為可選
}

// 定義訊息類型，與後端 models.Message 保持一致
export interface Message {
  id?: string; // 後端生成
  type?: "normal" | "system" | "room_state_update"; // 消息類型，新增 room_state_update
  senderId: string;
  senderUsername: string;
  roomId: string; // 聊天室ID
  roomName: string; // 聊天室名稱
  content: string;
  timestamp: string; // ISO 格式日期字串
}
