// src/pages/HomePage.tsx
import React, { useEffect, useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  AppShell,
  Burger,
  Group,
  Text,
  Button,
  rem,
  Avatar,
  UnstyledButton,
  ScrollArea,
  Divider,
  Paper,
  Title,
  Stack,
  TextInput,
  ActionIcon,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { getUserSession, clearUserSession } from "../utils/auth";
import { getAllUsers } from "../api/user";
import {
  IconMessageCircle,
  IconLogout,
  IconUserCircle,
  IconSend,
  IconPhoto,
} from "@tabler/icons-react";

interface ChatRoom {
  id: string;
  name: string;
  creatorId: string;
  participants: User[];
  createdAt: string;
}

interface User {
  id: string;
  username: string;
  email?: string; // 將 email 設為可選
}

// 定義訊息類型，與後端 models.Message 保持一致
interface Message {
  id?: string; // 後端生成
  senderId: string;
  senderUsername: string;
  roomId: string; // 聊天室ID
  roomName: string; // 聊天室名稱
  content: string;
  timestamp: string; // ISO 格式日期字串
}

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

    const fetchAllUsers = async () => {
      const users = await getAllUsers();
      if (userSession) {
        setAllUsers(users.filter((u: User) => u.id !== userSession!.id));
      }
    };
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
    (roomId: string, roomName: string) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close(); // 關閉現有連線
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
  const fetchChatHistory = useCallback(async (roomId: string) => {
    try {
      const response = await fetch(
        `http://localhost:8080/chat-history?roomId=${roomId}`
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
  }, [notifications]); // 依賴 notifications

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

  // 生成唯一的聊天室 ID (基於兩個使用者 ID)
  const generateRoomId = (user1Id: string, user2Id: string): string => {
    const sortedIds = [user1Id, user2Id].sort();
    return `${sortedIds[0]}-${sortedIds[1]}`;
  };

  // 打開聊天室 (與特定使用者建立/加入聊天室)
  const startChatWithUser = async (targetUser: User) => {
    if (!userSession) return;

    const roomId = generateRoomId(userSession.id, targetUser.id);
    const roomName = `${userSession.username}、${targetUser.username} 的聊天室`;

    let existingRoom = chatRooms.find((room) => room.id === roomId);

    if (!existingRoom) {
      // 如果聊天室不存在，則建立一個新的
      const newRoom: ChatRoom = {
        id: roomId,
        name: roomName,
        creatorId: userSession.id,
        participants: [
          {
            id: userSession.id,
            username: userSession.username,
            email: userSession.email || "",
          },
          targetUser,
        ],
        createdAt: new Date().toISOString(),
      };
      setChatRooms((prev) => [...prev, newRoom]);
      existingRoom = newRoom; // 將新建立的房間賦值給 existingRoom
    }

    setSelectedRoom(existingRoom); // 設定選中的聊天室
    connectWebSocket(existingRoom.id, existingRoom.name); // 連接到該聊天室的 WebSocket

    // 獲取歷史聊天記錄
    const messages = await fetchChatHistory(existingRoom.id);
    setMessages((prevMessagesMap) => {
      const newMap = new Map(prevMessagesMap);
      newMap.set(existingRoom.id, messages);
      return newMap;
    });

    notifications.show({
      title: "進入聊天室",
      message: `你已進入聊天室：${existingRoom.name}`,
      color: "blue",
      autoClose: 1500,
    });
  };

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
          <Group justify="space-between" style={{ flex: 1 }}>
            <Text size="xl" fw={700}>
              GoChat
            </Text>
            <Group>
              <Text fw={500}>歡迎，{userSession.username}！</Text>
              <Button
                variant="light"
                onClick={handleLogout}
                leftSection={<IconLogout size={16} />}
              >
                登出
              </Button>
            </Group>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <ScrollArea h="calc(100vh - var(--app-shell-header-height) - var(--app-shell-footer-height, 0px))">
          <Stack gap="md">
            {/* 移除「建立新聊天室」按鈕，因為現在是點擊使用者建立 */}

            <div>
              <Text size="lg" fw={600} mb="md">
                聊天室列表
              </Text>
              <Divider mb="sm" />
              {chatRooms.length === 0 ? (
                <Text c="dimmed" size="sm" mb="md">
                  尚無聊天室。請從下方選擇使用者建立聊天室。
                </Text>
              ) : (
                <Stack gap="md">
                  {chatRooms.map((room) => (
                    <UnstyledButton
                      key={room.id}
                      onClick={() => handleSelectRoom(room)} // 呼叫新的處理函式
                      style={{
                        display: "flex",
                        alignItems: "center",
                        padding: rem(10),
                        borderRadius: "var(--mantine-radius-sm)",
                        backgroundColor:
                          selectedRoom?.id === room.id
                            ? "var(--mantine-color-blue-0)"
                            : "transparent",
                      }}
                    >
                      <Avatar color="blue" radius="xl">
                        <IconMessageCircle size={24} />
                      </Avatar>
                      <Text ml="md" fw={500}>
                        {room.name}
                      </Text>
                    </UnstyledButton>
                  ))}
                </Stack>
              )}
            </div>

            <div>
              <Text size="lg" fw={600} mb="md">
                所有使用者
              </Text>
              <Divider mb="sm" />
              {allUsers.length === 0 ? (
                <Text c="dimmed">沒有其他使用者。</Text>
              ) : (
                <Stack gap="md">
                  {allUsers.map((user) => (
                    <UnstyledButton
                      key={user.id}
                      onClick={() => startChatWithUser(user)} // 點擊使用者建立/加入聊天室
                      style={{
                        display: "flex",
                        alignItems: "center",
                        padding: rem(10),
                        borderRadius: "var(--mantine-radius-sm)",
                        backgroundColor: "transparent",
                      }}
                    >
                      <Avatar color="gray" radius="xl">
                        <IconUserCircle size={24} />
                      </Avatar>
                      <Text ml="md" fw={500}>
                        {user.username}
                      </Text>
                    </UnstyledButton>
                  ))}
                </Stack>
              )}
            </div>
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
              <Button variant="light" color="red" onClick={exitChat}>
                退出聊天室
              </Button>
            </Group>
            <Divider mb="md" />
            {/* 聊天訊息顯示區域 */}
            <ScrollArea style={{ flex: 1, marginBottom: rem(10) }}>
              <Stack>
                {selectedRoom &&
                  messages.get(selectedRoom.id)?.map((msg, index) => (
                    <Group
                      key={index}
                      justify={
                        msg.senderId === userSession.id
                          ? "flex-end"
                          : "flex-start"
                      }
                    >
                      <Paper
                        p="xs"
                        radius="md"
                        shadow="xs"
                        bg={
                          msg.senderId === userSession.id
                            ? "#c3efab" //淺綠色
                            : "#cde2ff" //淺藍色
                        }
                        style={{ maxWidth: "70%" }}
                      >
                        <Text size="xs" c="dark">
                          {msg.senderUsername} (
                          {new Date(msg.timestamp).toLocaleTimeString()})
                        </Text>
                        <Text>{msg.content}</Text>
                      </Paper>
                    </Group>
                  ))}
                <div ref={messagesEndRef} /> {/* 滾動錨點 */}
              </Stack>
            </ScrollArea>

            {/* 訊息輸入框和發送按鈕 */}
            <Group wrap="nowrap" align="flex-end">
              <ActionIcon
                size="xl"
                variant="light"
                color="gray"
                aria-label="Upload Image"
                disabled
              >
                <IconPhoto size={24} />
              </ActionIcon>
              <TextInput
                style={{ flex: 1 }}
                placeholder="輸入訊息..."
                value={messageInput}
                onChange={(event) => setMessageInput(event.currentTarget.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    handleSendMessage();
                  }
                }}
                size="md"
              />
              <Button
                size="md"
                onClick={handleSendMessage}
                leftSection={<IconSend size={18} />}
              >
                發送
              </Button>
            </Group>
          </Paper>
        ) : (
          // 首頁內容 - 提示選擇聊天室
          <Paper
            p="xl"
            shadow="sm"
            radius="md"
            style={{
              height: "calc(100vh - 100px)",
              display: "flex",
              flexDirection: "column",
              justifyContent: "center",
              alignItems: "center",
            }}
          >
            <Title order={2} ta="center" mb="md">
              歡迎使用 GoChat
            </Title>
            <Text c="dimmed" ta="center">
              您可以從左側選擇使用者來建立或加入聊天室。
            </Text>
            <IconMessageCircle
              size={100}
              color="var(--mantine-color-gray-4)"
              style={{ marginTop: rem(40) }}
            />
          </Paper>
        )}
      </AppShell.Main>
    </AppShell>
  );
}

export default HomePage;
