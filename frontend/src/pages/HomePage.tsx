// src/pages/HomePage.tsx
import React, { useEffect, useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import type { ChatRoom, User, Message } from "../types/index";
import {
  AppShell,
  Burger,
  Group,
  Text,
  Button,
  ScrollArea,
  Divider,
  Paper,
  Title,
  Stack,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { getUserSession, clearUserSession } from "../utils/utils_auth";
import { getAllUsers } from "../api/api_user";
import {
  createOrGetChatRoom,
  getUserChatRooms,
  leaveChatRoom,
} from "../api/api_chatroom";
import UserList from "../components/lists/UserList";
import ChatRoomList from "../components/lists/ChatRoomList";
import ChatMessages from "../components/chat/ChatMessages";
import MessageInput from "../components/chat/MessageInput";
import AppHeader from "../components/common/AppHeader";
import WelcomeMessage from "../components/common/WelcomeMessage";
import InviteUsersModal from "../components/modals/InviteUsersModal"; // 引入新的邀請 Modal 組件

function HomePage() {
  const navigate = useNavigate();
  const [userSession, setUserSession] = useState(getUserSession());
  const [opened, { toggle }] = useDisclosure();
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [chatRooms, setChatRooms] = useState<ChatRoom[]>([]); // 使用者已加入的聊天室列表
  const [selectedRoom, setSelectedRoom] = useState<ChatRoom | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [messageInput, setMessageInput] = useState("");
  const [messages, setMessages] = useState<Map<string, Message[]>>(new Map()); // 聊天訊息列表，按聊天室ID分組
  // 邀請 Modal 相關狀態
  const [
    isInviteModalOpen,
    { open: openInviteModal, close: closeInviteModal },
  ] = useDisclosure(false);
  const [chatRoomToInvite, setChatRoomToInvite] = useState<ChatRoom | null>(
    null
  );

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const notificationShownRef = useRef(false);

  // 滾動到最新訊息
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  // 獲取所有使用者列表
  useEffect(() => {
    if (!userSession) {
      navigate("/auth");
      return;
    }

    // 獲取所有使用者
    const fetchAllUsers = async () => {
      const users = await getAllUsers();
      if (userSession) {
        setAllUsers(users.filter((u: User) => u.id !== userSession!.id));
      }
    };

    // 獲取使用者的聊天室列表
    const fetchUserChatRooms = async () => {
      try {
        const rooms = await getUserChatRooms();
        setChatRooms(rooms || []); // 確保總是設置為陣列
      } catch (error) {
        console.error("Error fetching chat rooms:", error);
        setChatRooms([]); // 錯誤時設置為空陣列
      }
    };

    fetchUserChatRooms();
    fetchAllUsers();
  }, [userSession, navigate]);

  // 訊息列表更新時滾動到底部
  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // WebSocket 訊息處理
  const handleWsMessage = useCallback((event: MessageEvent) => {
    const receivedMessage: Message = JSON.parse(event.data);
    // console.log("receivedMessage: ", receivedMessage);
    setMessages((prevMessagesMap) => {
      const newMap = new Map(prevMessagesMap);
      const roomMessages = newMap.get(receivedMessage.roomId) || [];
      const isDuplicate = roomMessages.some(
        (msg) => msg.id === receivedMessage.id
      );

      if (!isDuplicate) {
        newMap.set(receivedMessage.roomId, [...roomMessages, receivedMessage]);
      }
      // console.log("更新後的 messages Map:", newMap);
      return newMap;
    });
  }, []);

  // WebSocket 連線關閉處理
  const handleWsClose = useCallback((event: CloseEvent) => {
    console.log("WebSocket 連線關閉:", event);
    notifications.show({
      title: "連線關閉",
      message: "與聊天伺服器的連線已斷開。",
      color: "orange",
      autoClose: 1500,
    });
    setWs(null);
    notificationShownRef.current = false;
  }, []);

  // WebSocket 錯誤處理
  const handleWsError = useCallback((error: Event) => {
    console.error("WebSocket 連線錯誤:", error);
    notifications.show({
      title: "連線錯誤",
      message: "WebSocket 連線發生錯誤。",
      color: "red",
      autoClose: 2000,
    });
  }, []);

  // 建立 WebSocket 連線
  const connectWebSocket = useCallback(
    async (roomId: string, roomName: string) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        await new Promise((resolve) => {
          ws.onclose = resolve;
          ws.close();
        });
      }

      const websocketUrl = `ws://localhost:8080/ws?userId=${
        userSession!.id
      }&username=${
        userSession!.username
      }&roomId=${roomId}&roomName=${encodeURIComponent(roomName)}`;
      const newWs = new WebSocket(websocketUrl);

      newWs.onopen = () => {
        console.log("WebSocket 連線成功！");
        if (!notificationShownRef.current) {
          notifications.show({
            title: "連線成功",
            message: "已成功連接到聊天伺服器。",
            color: "green",
            autoClose: 1500,
          });
          notificationShownRef.current = true;
        }
      };
      newWs.onmessage = handleWsMessage;
      newWs.onclose = handleWsClose;
      newWs.onerror = handleWsError;

      setWs(newWs);

      return () => {
        if (newWs.readyState === WebSocket.OPEN) {
          newWs.close();
        }
        notificationShownRef.current = false;
      };
    },
    [userSession, ws, handleWsMessage, handleWsClose, handleWsError]
  );

  const handleLogout = () => {
    clearUserSession();
    setUserSession(null);
    navigate("/auth");
  };

  // 獲取歷史聊天記錄
  const fetchChatHistory = useCallback(
    async (roomId: string) => {
      try {
        const response = await fetch(
          `http://localhost:8080/chat-history?roomId=${roomId}`,
          {
            headers: {
              Authorization: `Bearer ${userSession!.token}`,
            },
          }
        );
        if (!response.ok) {
          throw new Error("Failed to fetch chat history");
        }
        const data = await response.json();
        return data.messages;
      } catch (error) {
        console.error(`Error fetching chat history for room ${roomId}:`, error);
        notifications.show({
          title: "錯誤",
          message: `無法獲取聊天室 ${roomId} 的聊天記錄`,
          color: "red",
          autoClose: 2000,
        });
        return [];
      }
    },
    [userSession]
  );

  // 處理點擊聊天室列表項目
  const handleSelectRoom = useCallback(
    async (room: ChatRoom) => {
      setSelectedRoom(room);
      connectWebSocket(room.id, room.name); // 重新連線到該聊天室的 WebSocket

      // 同時，重新獲取該聊天室的歷史訊息
      const messages = await fetchChatHistory(room.id);
      setMessages((prevMessagesMap) => {
        const newMap = new Map(prevMessagesMap);
        newMap.set(room.id, messages);
        return newMap;
      });

      notifications.show({
        title: "進入聊天室",
        message: `你已進入聊天室：${room.name}`,
        color: "blue",
        autoClose: 1500,
      });
    },
    [setSelectedRoom, connectWebSocket, fetchChatHistory, setMessages]
  );

  // 打開聊天室 (與特定使用者建立/加入聊天室)
  const startChatWithUser = async (targetUser: User) => {
    if (!userSession) return;

    try {
      const roomName = `${userSession.username}、${targetUser.username} 的聊天室`;
      // 使用 API 創建或獲取聊天室
      const room = await createOrGetChatRoom(
        [userSession.id, targetUser.id],
        roomName
      );

      if (!room) {
        notifications.show({
          title: "錯誤",
          message: "無法創建或獲取聊天室",
          color: "red",
          autoClose: 2000,
        });
        return;
      }

      // 更新聊天室列表
      setChatRooms((prev) => {
        // 檢查是否已存在相同 ID 的聊天室
        const exists = prev.some((r) => r.id === room.id);
        if (!exists) {
          return [...prev, room];
        }
        return prev;
      });

      setSelectedRoom(room); // 設定選中的聊天室
      connectWebSocket(room.id, room.name); // 連接到該聊天室的 WebSocket

      // 獲取歷史聊天記錄
      const messages = await fetchChatHistory(room.id);
      setMessages((prevMessagesMap) => {
        const newMap = new Map(prevMessagesMap);
        newMap.set(room.id, messages);
        return newMap;
      });

      notifications.show({
        title: "進入聊天室",
        message: `你已進入聊天室：${room.name}`,
        color: "blue",
        autoClose: 1500,
      });
    } catch (error) {
      console.error("Error creating/getting chatroom:", error);
      notifications.show({
        title: "錯誤",
        message: "創建或加入聊天室時發生錯誤",
        color: "red",
        autoClose: 2000,
      });
    }
  };

  // 處理邀請按鈕點擊事件
  const handleInviteClick = useCallback(
    (room: ChatRoom) => {
      setChatRoomToInvite(room); // 設定當前要邀請的聊天室
      openInviteModal(); // 開啟邀請 Modal
    },
    [openInviteModal]
  );

  // 處理退出聊天室
  const handleLeaveRoom = useCallback(
    async (room: ChatRoom) => {
      try {
        const success = await leaveChatRoom(room.id);

        if (success) {
          // 關閉 WebSocket 連接
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.close();
          }

          // 從聊天室列表中移除
          setChatRooms((prev) => prev.filter((r) => r.id !== room.id));

          // 如果正在查看此聊天室，則導向首頁
          if (selectedRoom?.id === room.id) {
            setSelectedRoom(null);
          }

          // 顯示通知
          notifications.show({
            title: "已退出聊天室",
            message: `您已成功退出聊天室：${room.name}`,
            color: "blue",
            autoClose: 1500,
          });
        }
      } catch (error) {
        console.error("Error leaving room:", error);
      }
    },
    [ws, selectedRoom]
  );

  // 關閉聊天室
  const exitChat = () => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.close(); // 關閉當前 WebSocket 連線
    }

    setSelectedRoom(null);
    notifications.show({
      title: "退出聊天室",
      message: "你已回到首頁",
      color: "blue",
      autoClose: 1500,
    });
  };

  // 使用者發送訊息
  const handleSendMessage = () => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      notifications.show({
        title: "連線錯誤",
        message: "WebSocket 未連線或已關閉，無法發送訊息。",
        color: "red",
        autoClose: 2000,
      });
      return;
    }

    if (messageInput.trim() === "") {
      return;
    }

    if (!selectedRoom) {
      notifications.show({
        title: "錯誤",
        message: "請先選擇一個聊天室。",
        color: "red",
        autoClose: 2000,
      });
      return;
    }

    const messageToSend: Partial<Message> = {
      roomId: selectedRoom.id,
      roomName: selectedRoom.name,
      content: messageInput,
    };

    ws.send(JSON.stringify(messageToSend));
    setMessageInput("");
  };

  if (!userSession) {
    return <Text>重定向中...</Text>;
  }

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 300,
        breakpoint: "sm",
        collapsed: { mobile: !opened },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md">
          <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
          <AppHeader username={userSession.username} onLogout={handleLogout} />
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <ScrollArea h="calc(100vh - var(--app-shell-header-height) - var(--app-shell-footer-height, 0px))">
          <Stack gap="md">
            

            <ChatRoomList
              chatRooms={chatRooms}
              selectedRoomId={selectedRoom?.id || null}
              onSelectRoom={handleSelectRoom}
              onLeaveRoom={handleLeaveRoom}
              onInviteClick={handleInviteClick}
              // userSession={userSession} // 如果 ChatRoomList 內部需要 userSession 的細節，可以傳遞
            />

            <UserList users={allUsers} onStartChat={startChatWithUser} />
          </Stack>
        </ScrollArea>
      </AppShell.Navbar>

      <AppShell.Main>
        {selectedRoom ? (
          // 聊天室介面
          <Paper
            p="md"
            shadow="sm"
            radius="md"
            style={{
              height: "calc(100vh - 100px)",
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Group justify="space-between" align="center" mb="md">
              <Title order={3}>聊天室：{selectedRoom.name}</Title>
              <Button variant="light" color="green" onClick={exitChat}>
                回到首頁
              </Button>
            </Group>
            <Divider mb="md" />

            <ChatMessages
              messages={messages.get(selectedRoom.id) || []}
              userSessionId={userSession.id}
              messagesEndRef={messagesEndRef} // 傳遞 ref
            />

            <MessageInput
              messageInput={messageInput}
              onMessageInputChange={setMessageInput}
              onSendMessage={handleSendMessage}
              isDisabled={
                !selectedRoom || !ws || ws.readyState !== WebSocket.OPEN
              }
            />
          </Paper>
        ) : (
          // 首頁內容 - 提示選擇聊天室
          <WelcomeMessage />
        )}
      </AppShell.Main>
      {/* 邀請用戶 Modal */}
      {chatRoomToInvite && ( // 確保有選中的聊天室才渲染 Modal
        <InviteUsersModal
          opened={isInviteModalOpen}
          onClose={closeInviteModal}
          chatRoom={chatRoomToInvite}
          allUsers={allUsers} // 傳遞所有用戶列表
          // 目前不傳遞邀請相關的處理函數，因為功能還沒實現
          onInvite={() => {
            notifications.show({
              title: "提示",
              message: "邀請功能正在開發中...",
              color: "yellow",
              autoClose: 1500,
            });
            closeInviteModal();
          }}
        />
      )}
    </AppShell>
  );
}

export default HomePage;
