// src/pages/HomePage.tsx
import React, { useEffect, useState, useRef } from "react";
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
  TextInput, // 引入 TextInput
  ActionIcon, // 引入 ActionIcon
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { getUserSession, clearUserSession } from "../utils/auth";
import { getAllUsers } from "../api/user"; // 引入 getAllUsers
import {
  IconMessageCircle,
  IconLogout,
  IconUserCircle,
  IconSend, // 引入發送圖示
  IconPhoto, // 引入圖片圖示
} from "@tabler/icons-react";

interface User {
  id: string;
  username: string;
  email: string;
}

// 定義訊息類型，與後端 models.Message 保持一致
interface Message {
  id?: string; // 後端生成
  senderId: string;
  senderUsername: string;
  recipientId?: string; // 可選，一對一聊天時使用
  content: string;
  timestamp: string; // ISO 格式日期字串
}

function HomePage() {
  const navigate = useNavigate();
  const [userSession, setUserSession] = useState(getUserSession());
  const [opened, { toggle }] = useDisclosure();
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null); // WebSocket 實例
  const [messageInput, setMessageInput] = useState(""); // 訊息輸入框內容
  const [messages, setMessages] = useState<Map<string, Message[]>>(new Map()); // 聊天訊息列表，按使用者ID分組

  const messagesEndRef = useRef<HTMLDivElement>(null); // 用於自動滾動到底部

  // 滾動到最新訊息
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    if (!userSession) {
      navigate("/auth");
      return;
    }

    // 獲取所有使用者列表
    const fetchAllUsers = async () => {
      const users = await getAllUsers(); // 調用新的 API 函數
      if (userSession) { // 確保 userSession 存在
        setAllUsers(users.filter((u: User) => u.id !== userSession!.id)); // 使用非空斷言
      }
    };
    fetchAllUsers();

    // 建立 WebSocket 連線
    const websocketUrl = `ws://localhost:8080/ws?userId=${userSession.id}&username=${userSession.username}`;
    const newWs = new WebSocket(websocketUrl);

    newWs.onopen = () => {
      console.log("WebSocket 連線成功！");
      notifications.show({
        title: "連線成功",
        message: "已成功連接到聊天伺服器。",
        color: "green",
      });
    };

    newWs.onmessage = (event) => {
      const receivedMessage: Message = JSON.parse(event.data);
      console.log("收到訊息:", receivedMessage);

      setMessages((prevMessagesMap) => {
        const newMap = new Map(prevMessagesMap);
        const currentUserId = userSession!.id;

        // 判斷訊息是發送給誰的，或者誰發送的
        let chatPartnerId: string | null = null;

        // 如果訊息有明確的接收者且是當前使用者
        if (receivedMessage.recipientId === currentUserId) {
          chatPartnerId = receivedMessage.senderId;
        }
        // 如果訊息是當前使用者發送的
        else if (receivedMessage.senderId === currentUserId) {
          chatPartnerId = receivedMessage.recipientId || null; // 如果是廣播訊息，這裡可能為空
        }
        // 如果訊息沒有明確的接收者，且不是當前使用者發送的，則可能是廣播訊息
        // 或者其他不相關的訊息，這裡我們暫時忽略
        else {
          // 如果是廣播訊息，可以考慮一個特殊的 chatPartnerId，例如 "broadcast"
          // 但目前我們的後端邏輯是針對一對一訊息的，所以這裡只處理有明確 senderId/recipientId 的情況
          return prevMessagesMap;
        }

        if (chatPartnerId) {
          const existingMessages = newMap.get(chatPartnerId) || [];
          // 避免重複添加訊息，特別是歷史訊息在連線時會一次性發送
          const isDuplicate = existingMessages.some(
            (msg) => msg.id === receivedMessage.id
          );
          if (!isDuplicate) {
            newMap.set(chatPartnerId, [...existingMessages, receivedMessage]);
          }
        }
        return newMap;
      });
    };

    newWs.onclose = (event) => {
      console.log("WebSocket 連線關閉:", event);
      notifications.show({
        title: "連線關閉",
        message: "與聊天伺服器的連線已斷開。",
        color: "orange",
      });
      setWs(null); // 清除 WebSocket 實例
    };

    newWs.onerror = (error) => {
      console.error("WebSocket 連線錯誤:", error);
      notifications.show({
        title: "連線錯誤",
        message: "WebSocket 連線發生錯誤。",
        color: "red",
      });
    };

    setWs(newWs);

    // 清理函數：組件卸載時關閉 WebSocket 連線
    return () => {
      if (newWs.readyState === WebSocket.OPEN) {
        newWs.close();
      }
    };
  }, [userSession, navigate]);

  // 訊息列表更新時滾動到底部
  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleLogout = () => {
    clearUserSession();
    setUserSession(null);
    navigate("/auth");
  };

  const startChat = (user: User) => {
    setSelectedUser(user);
    // 不再清空訊息列表，而是根據 selectedUser 篩選顯示
    notifications.show({
      title: "進入聊天室",
      message: `你已進入與 ${user.username} 的聊天室`,
      color: "blue",
    });
  };

  const exitChat = () => {
    setSelectedUser(null);
    // 退出聊天室時不清空訊息，保持在 Map 中
    notifications.show({
      title: "退出聊天室",
      message: "你已回到首頁",
      color: "blue",
    });
  };

  const handleSendMessage = () => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      notifications.show({
        title: "連線錯誤",
        message: "WebSocket 未連線或已關閉，無法發送訊息。",
        color: "red",
      });
      return;
    }

    if (messageInput.trim() === "") {
      return; // 不發送空訊息
    }

    const messageToSend: Partial<Message> = {
      content: messageInput,
    };

    // 如果有選中的聊天對象，則設定 recipientId
    if (selectedUser) {
      messageToSend.recipientId = selectedUser.id;
    }

    ws.send(JSON.stringify(messageToSend));

    // 手動將自己發送的訊息添加到訊息列表中，以便立即顯示
    // 注意：這裡的 timestamp 應該與後端生成的一致，但前端先用當前時間顯示
    const selfMessage: Message = {
      //使用"!"告訴程式碼變數不會是 null 或 undefined
      senderId: userSession!.id,
      senderUsername: userSession!.username,
      recipientId: selectedUser?.id, // 如果有接收者
      content: messageInput,
      timestamp: new Date().toISOString(), // 使用 ISO 格式
    };
    setMessages((prevMessagesMap) => {
      const newMap = new Map(prevMessagesMap);
      const chatPartnerId = selectedUser?.id || ""; // 如果是廣播訊息，這裡可能為空
      const existingMessages = newMap.get(chatPartnerId) || [];
      newMap.set(chatPartnerId, [...existingMessages, selfMessage]);
      return newMap;
    });

    setMessageInput(""); // 清空輸入框
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
          <Text size="lg" fw={600} mb="md">
            所有使用者
          </Text>
          <Divider mb="sm" />
          {allUsers.length === 0 ? (
            <Text c="dimmed">沒有其他使用者。</Text>
          ) : (
            <Stack>
              {allUsers.map((user) => (
                <UnstyledButton
                  key={user.id}
                  onClick={() => startChat(user)}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    padding: rem(10),
                    borderRadius: "var(--mantine-radius-sm)",
                    backgroundColor:
                      selectedUser?.id === user.id
                        ? "var(--mantine-color-blue-0)"
                        : "transparent",
                  }}
                >
                  <Avatar color="blue" radius="xl">
                    <IconUserCircle size={(24)} />
                  </Avatar>
                  <Text ml="md" fw={500}>
                    {user.username}
                  </Text>
                </UnstyledButton>
              ))}
            </Stack>
          )}
        </ScrollArea>
      </AppShell.Navbar>

      <AppShell.Main>
        {selectedUser ? (
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
              <Title order={3}>與 {selectedUser.username} 的聊天室</Title>
              <Button variant="light" color="red" onClick={exitChat}>
                退出聊天室
              </Button>
            </Group>
            <Divider mb="md" />
            {/* 聊天訊息顯示區域 */}
            <ScrollArea style={{ flex: 1, marginBottom: rem(10) }}>
              <Stack>
                {selectedUser &&
                  messages.get(selectedUser.id)?.map((msg, index) => (
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
                            ? "var(--mantine-color-blue-light)"
                            : "var(--mantine-color-gray-2)"
                        }
                        style={{ maxWidth: "70%" }}
                      >
                        <Text size="xs" c="dimmed">
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
              {/* 預留圖片上傳按鈕位置 */}
              <ActionIcon
                size="xl"
                variant="light"
                color="gray"
                aria-label="Upload Image"
                disabled // 暫時禁用
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
                    event.preventDefault(); // 防止換行
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
          // 首頁內容 - 提示選擇使用者
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
              選擇一位使用者開始聊天
            </Title>
            <Text c="dimmed" ta="center">
              從左側導航欄點擊一位使用者來進入聊天室。
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
