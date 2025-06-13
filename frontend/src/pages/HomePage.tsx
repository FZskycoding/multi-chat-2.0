// src/pages/HomePage.tsx
import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  AppShell,
  Burger,
  Group,
  Skeleton,
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
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { getUserSession, clearUserSession } from "../utils/auth";
import {
  IconMessageCircle,
  IconLogout,
  IconUserCircle,
} from "@tabler/icons-react"; // Mantine icons 建議使用 @tabler/icons-react

interface User {
  id: string;
  username: string;
  email: string; // 假設後端提供
}

function HomePage() {
  const navigate = useNavigate();
  const [userSession, setUserSession] = useState(getUserSession());
  const [opened, { toggle }] = useDisclosure();
  const [allUsers, setAllUsers] = useState<User[]>([]); // 存放所有使用者
  const [selectedUser, setSelectedUser] = useState<User | null>(null); // 被選中的聊天對象

  useEffect(() => {
    if (!userSession) {
      navigate("/auth"); // 如果沒有登入會話，重定向到登入頁
    } else {
      // 在這裡發送請求獲取所有使用者列表
      // 由於目前後端沒有提供獲取所有使用者的 API，這裡先用模擬資料
      // 實際專案中你需要新增一個 Go 後端 API 來獲取所有使用者
      const fetchAllUsers = async () => {
        try {
          // ⚠️ 注意：這裡需要替換為你的 Go 後端獲取所有使用者的 API 端點
          // 範例：
          const response = await fetch("http://localhost:8080/users", {
            // 假設有一個 /users 接口
            headers: {
              "Content-Type": "application/json",
              // 如果有 token 認證，這裡需要加上 Authorization Header
              // 'Authorization': `Bearer ${userSession.token}`
            },
          });
          const data = await response.json();
          if (response.ok) {
            // 過濾掉當前登入的使用者
            setAllUsers(data.filter((u: User) => u.id !== userSession?.id));
          } else {
            notifications.show({
              title: "載入使用者失敗",
              message: data.message || "無法載入所有使用者列表",
              color: "red",
            });
          }
        } catch (error) {
          console.error("獲取所有使用者請求錯誤:", error);
          notifications.show({
            title: "網路錯誤",
            message: "無法連線到伺服器，請檢查網路。",
            color: "red",
          });
          // 模擬資料，以便開發階段看到效果
          setAllUsers(
            [
              {
                id: "mock-user-1",
                username: "ChatBuddy1",
                email: "chat1@example.com",
              },
              {
                id: "mock-user-2",
                username: "AwesomeUser",
                email: "awesome@example.com",
              },
              {
                id: "mock-user-3",
                username: "TesterLin",
                email: "tester@example.com",
              },
            ].filter((u) => u.id !== userSession.id)
          );
        }
      };
      fetchAllUsers();
    }
  }, [userSession, navigate]);

  const handleLogout = () => {
    clearUserSession();
    setUserSession(null);
    navigate("/auth");
  };

  const startChat = (user: User) => {
    setSelectedUser(user);
    notifications.show({
      title: "進入聊天室",
      message: `你已進入與 ${user.username} 的聊天室`,
      color: "blue",
    });
  };

  const exitChat = () => {
    setSelectedUser(null);
    notifications.show({
      title: "退出聊天室",
      message: "你已回到首頁",
      color: "blue",
    });
  };

  if (!userSession) {
    return <Text>重定向中...</Text>; // 或者顯示一個載入畫面
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
            <Text c="dimmed">
              這裡是與 {selectedUser.username} 的聊天內容，功能尚未實作。
            </Text>
            {/* 未來聊天訊息將會在這裡顯示 */}
            <Skeleton height={200} mt="md" radius="sm" />
            <Skeleton height={40} mt="md" width="70%" radius="sm" />
            {/* 訊息輸入框等 */}
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
