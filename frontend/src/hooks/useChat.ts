// src/hooks/useChat.ts
import { useState, useCallback } from "react";
import {
  getUserChatRooms,
  createChatRoom,
  leaveChatRoom,
} from "../api/api_chatroom";
import type { ChatRoom, User, Message } from "../types";
import { notifications } from "@mantine/notifications";

export const useChat = (
  userSession: ReturnType<typeof import("../utils/utils_auth").getUserSession>
) => {
  const [chatRooms, setChatRooms] = useState<ChatRoom[]>([]);
  const [selectedRoom, setSelectedRoom] = useState<ChatRoom | null>(null);
  const [messages, setMessages] = useState(new Map<string, Message[]>());

  const fetchUserChatRooms = useCallback(async () => {
    const rooms = await getUserChatRooms();
    if (!rooms) {
      setChatRooms([]);
      return;
    }
    const sortedRooms = rooms.sort(
      (a, b) =>
        new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
    );
    setChatRooms(sortedRooms);
  }, []);

  const fetchChatHistory = useCallback(
    async (roomId: string) => {
      if (!userSession?.token) return [];
      try {
        const response = await fetch(
          `http://localhost:8080/chat-history?roomId=${roomId}`,
          {
            headers: { Authorization: `Bearer ${userSession.token}` },
          }
        );
        if (!response.ok) throw new Error("Failed to fetch chat history");
        const data = await response.json();
        return data.messages || [];
      } catch (error) {
        console.error(`Error fetching chat history for room ${roomId}:`, error);
        return [];
      }
    },
    [userSession]
  );

  const handleSelectRoom = useCallback(
    async (room: ChatRoom) => {
      setSelectedRoom(room);
      const history = await fetchChatHistory(room.id);
      setMessages((prev) => new Map(prev).set(room.id, history));
    },
    [fetchChatHistory]
  );

  const startChatWithUser = useCallback(
    async (targetUser: User) => {
      if (!userSession) return;
      const room = await createChatRoom([userSession.id, targetUser.id]);
      if (!room) return;
      await fetchUserChatRooms();
      setSelectedRoom(room);
      const history = await fetchChatHistory(room.id);
      setMessages((prev) => new Map(prev).set(room.id, history));
    },
    [userSession, fetchUserChatRooms, fetchChatHistory]
  );

  const handleLeaveRoom = useCallback(
    async (room: ChatRoom) => {
      const success = await leaveChatRoom(room.id);
      if (success) {
        setChatRooms((prev) => prev.filter((r) => r.id !== room.id));
        if (selectedRoom?.id === room.id) {
          setSelectedRoom(null);
        }
        notifications.show({
          title: "已退出",
          message: `您已成功退出聊天室：${room.name}`,
          color: "blue",
        });
      }
    },
    [selectedRoom]
  );

  const exitChat = () => {
    setSelectedRoom(null);
  };

  const updateChatState = useCallback(
    (message: Message) => {
      // 更新聊天室列表順序和名稱
      setChatRooms((prevChatRooms) => {
        const roomExists = prevChatRooms.some(
          (room) => room.id === message.roomId
        );

        if (!roomExists && message.type !== "room_state_update") {
          fetchUserChatRooms();
          return prevChatRooms;
        }

        const updatedChatRooms = prevChatRooms.map((room) => {
          if (room.id === message.roomId) {
            return {
              ...room,
              name: message.roomName,
              updatedAt: message.timestamp,
            };
          }
          return room;
        });
        return updatedChatRooms.sort(
          (a, b) =>
            new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
        );
      });

      // 更新當前聊天室的標題
      setSelectedRoom((prev) =>
        prev && prev.id === message.roomId
          ? { ...prev, name: message.roomName }
          : prev
      );

      // 更新訊息列表
      if (message.type === "normal" || message.type === "system") {
        setMessages((prev) => {
          const newMap = new Map(prev);
          const roomMessages = newMap.get(message.roomId) || [];
          if (!roomMessages.some((msg) => msg.id === message.id)) {
            newMap.set(message.roomId, [...roomMessages, message]);
          }
          return newMap;
        });
      }
    },
    [fetchUserChatRooms]
  );

  return {
    chatRooms,
    selectedRoom,
    messages,
    handleSelectRoom,
    startChatWithUser,
    handleLeaveRoom,
    fetchUserChatRooms,
    exitChat,
    updateChatState,
    setChatRooms,
    setSelectedRoom,
  };
};
