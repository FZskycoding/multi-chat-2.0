// src/pages/HomePage.tsx
import { useEffect, useState, useRef, useCallback } from "react";
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
  createChatRoom,
  getUserChatRooms,
  leaveChatRoom,
} from "../api/api_chatroom";
import UserList from "../components/lists/UserList";
import ChatRoomList from "../components/lists/ChatRoomList";
import ChatMessages from "../components/chat/ChatMessages";
import MessageInput from "../components/chat/MessageInput";
import AppHeader from "../components/common/AppHeader";
import WelcomeMessage from "../components/common/WelcomeMessage";
import InviteUsersModal from "../components/modals/InviteUsersModal";

function HomePage() {
  const navigate = useNavigate();
  const [userSession, setUserSession] = useState(getUserSession());
  const [opened, { toggle }] = useDisclosure();
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [chatRooms, setChatRooms] = useState<ChatRoom[]>([]);
  const [selectedRoom, setSelectedRoom] = useState<ChatRoom | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [messageInput, setMessageInput] = useState("");
  const [messages, setMessages] = useState(new Map<string, Message[]>());

  const [
    isInviteModalOpen,
    { open: openInviteModal, close: closeInviteModal },
  ] = useDisclosure(false);
  const [chatRoomToInvite, setChatRoomToInvite] = useState<ChatRoom | null>(
    null
  );

  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const fetchUserChatRooms = useCallback(async () => {
    try {
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
    } catch (error) {
      console.error("Error fetching user chat rooms:", error);
      setChatRooms([]);
    }
  }, []);

  const fetchAllUsers = useCallback(async () => {
    if (userSession) {
      const users = await getAllUsers();
      setAllUsers(users.filter((u: User) => u.id !== userSession.id));
    }
  }, [userSession]);

  useEffect(() => {
    if (!userSession) {
      navigate("/auth");
      return;
    }
    if (ws.current && ws.current.readyState < 2) {
      return;
    }
    const websocketUrl = `ws://localhost:8080/ws?userId=${userSession.id}&username=${userSession.username}`;
    const newWs = new WebSocket(websocketUrl);
    newWs.onopen = () => setIsConnected(true);
    newWs.onclose = () => setIsConnected(false);
    newWs.onerror = () => setIsConnected(false);

    newWs.onmessage = (event: MessageEvent) => {
      const receivedMessage: Message = JSON.parse(event.data);
      if (receivedMessage.type === "room_state_update") {
        fetchUserChatRooms();
      }
      setChatRooms((prevChatRooms) => {
        const roomExists = prevChatRooms.some(
          (room) => room.id === receivedMessage.roomId
        );
        if (!roomExists && receivedMessage.type !== "room_state_update") {
          fetchUserChatRooms();
          return prevChatRooms;
        }
        const updatedChatRooms = prevChatRooms.map((room) => {
          if (room.id === receivedMessage.roomId) {
            return {
              ...room,
              name: receivedMessage.roomName,
              updatedAt: receivedMessage.timestamp,
            };
          }
          return room;
        });
        return updatedChatRooms.sort(
          (a, b) =>
            new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
        );
      });
      setSelectedRoom((prevSelectedRoom) =>
        prevSelectedRoom && prevSelectedRoom.id === receivedMessage.roomId
          ? { ...prevSelectedRoom, name: receivedMessage.roomName }
          : prevSelectedRoom
      );
      if (
        receivedMessage.type === "normal" ||
        receivedMessage.type === "system"
      ) {
        setMessages((prevMessagesMap) => {
          const newMap = new Map(prevMessagesMap);
          const roomMessages = newMap.get(receivedMessage.roomId) || [];
          if (!roomMessages.some((msg) => msg.id === receivedMessage.id)) {
            newMap.set(receivedMessage.roomId, [
              ...roomMessages,
              receivedMessage,
            ]);
          }
          return newMap;
        });
      }
    };
    ws.current = newWs;
    return () => {
      if (newWs && newWs.readyState < 2) {
        newWs.close();
      }
    };
  }, [userSession, navigate, fetchUserChatRooms]);

  useEffect(() => {
    if (userSession) {
      fetchAllUsers();
      fetchUserChatRooms();
    }
  }, [userSession, fetchAllUsers, fetchUserChatRooms]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleLogout = () => {
    if (ws.current) {
      ws.current.close();
    }
    clearUserSession();
    setUserSession(null);
    navigate("/auth");
  };

  const fetchChatHistory = useCallback(
    async (roomId: string) => {
      if (!userSession) return [];
      try {
        const response = await fetch(
          `http://localhost:8080/chat-history?roomId=${roomId}`,
          {
            headers: { Authorization: `Bearer ${userSession.token}` },
          }
        );
        if (!response.ok) throw new Error("Failed to fetch chat history");
        return response.json().then((data) => data.messages || []);
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
      try {
        const room = await createChatRoom([userSession.id, targetUser.id]);
        if (!room) return;
        fetchUserChatRooms();
        setSelectedRoom(room);
        const history = await fetchChatHistory(room.id);
        setMessages((prev) => new Map(prev).set(room.id, history));
      } catch (error) {
        console.error("Error starting chat:", error);
      }
    },
    [userSession, fetchUserChatRooms, fetchChatHistory]
  );

  const handleInviteClick = useCallback((room: ChatRoom) => {
    setChatRoomToInvite(room);
    openInviteModal();
  }, []);

  const handleInviteSuccess = useCallback(
    (updatedChatRoom: ChatRoom) => {
      fetchUserChatRooms();
      if (selectedRoom?.id === updatedChatRoom.id) {
        setSelectedRoom(updatedChatRoom);
      }
    },
    [selectedRoom, fetchUserChatRooms]
  );

  // 【最終修正】處理退出聊天室的邏輯
  const handleLeaveRoom = useCallback(
    async (room: ChatRoom) => {
      const success = await leaveChatRoom(room.id);
      if (success) {
        // API 呼叫成功後，立即從前端 state 中移除該聊天室
        setChatRooms((prevRooms) => prevRooms.filter((r) => r.id !== room.id));

        // 如果退出的聊天室是當前選中的，則清空選中狀態
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
    [selectedRoom] // 依賴項保持不變
  );

  const exitChat = () => {
    setSelectedRoom(null);
  };

  const handleSendMessage = useCallback(() => {
    if (!isConnected || !ws.current) {
      notifications.show({
        title: "錯誤",
        message: "WebSocket 未連線。",
        color: "red",
      });
      return;
    }
    if (!selectedRoom || messageInput.trim() === "") return;
    const messageToSend: Partial<Message> = {
      roomId: selectedRoom.id,
      content: messageInput,
      type: "normal",
    };
    ws.current.send(JSON.stringify(messageToSend));
    setMessageInput("");
  }, [isConnected, selectedRoom, messageInput]);

  if (!userSession) return <Text>重定向中...</Text>;

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{ width: 300, breakpoint: "sm", collapsed: { mobile: !opened } }}
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
            />
            <UserList users={allUsers} onStartChat={startChatWithUser} />
          </Stack>
        </ScrollArea>
      </AppShell.Navbar>

      <AppShell.Main>
        {selectedRoom ? (
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
              messagesEndRef={messagesEndRef}
            />
            <MessageInput
              messageInput={messageInput}
              onMessageInputChange={setMessageInput}
              onSendMessage={handleSendMessage}
              isDisabled={!isConnected}
            />
          </Paper>
        ) : (
          <WelcomeMessage />
        )}
      </AppShell.Main>

      {chatRoomToInvite && (
        <InviteUsersModal
          opened={isInviteModalOpen}
          onClose={closeInviteModal}
          chatRoom={chatRoomToInvite}
          allUsers={allUsers}
          onInviteSuccess={handleInviteSuccess}
        />
      )}
    </AppShell>
  );
}

export default HomePage;
