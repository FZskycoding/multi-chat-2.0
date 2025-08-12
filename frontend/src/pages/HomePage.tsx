// src/pages/HomePage.tsx
import { useEffect, useState, useRef, useCallback } from "react";
import {
  AppShell,
  Burger,
  Group,
  ScrollArea,
  Divider,
  Paper,
  Title,
  Button,
  Stack,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useAuth } from "../hooks/useAuth";
import { useUsers } from "../hooks/useUsers";
import { useChat } from "../hooks/useChat";
import { useWebSocket } from "../hooks/useWebSocket";

import UserList from "../components/lists/UserList";
import ChatRoomList from "../components/lists/ChatRoomList";
import ChatMessages from "../components/chat/ChatMessages";
import MessageInput from "../components/chat/MessageInput";
import AppHeader from "../components/common/AppHeader";
import WelcomeMessage from "../components/common/WelcomeMessage";
import InviteUsersModal from "../components/modals/InviteUsersModal";
import type { ChatRoom } from "../types";

function HomePage() {
  const [opened, { toggle }] = useDisclosure();
  const { userSession, handleLogout } = useAuth();
  const { allUsers, fetchAllUsers } = useUsers(userSession);
  const {
    chatRooms,
    selectedRoom,
    messages,
    handleSelectRoom,
    startChatWithUser,
    handleLeaveRoom,
    fetchUserChatRooms,
    exitChat,
    updateChatState,
    setSelectedRoom,
  } = useChat(userSession);

  const { isConnected, sendMessage } = useWebSocket(
    userSession,
    updateChatState, // onMessageReceived
    handleLogout, // onForceLogout
    fetchUserChatRooms // onStateUpdate
  );

  const [messageInput, setMessageInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const [
    isInviteModalOpen,
    { open: openInviteModal, close: closeInviteModal },
  ] = useDisclosure(false);
  const [chatRoomToInvite, setChatRoomToInvite] = useState<ChatRoom | null>(null);

  useEffect(() => {
    if (userSession) {
      fetchUserChatRooms();
      fetchAllUsers();
    }
  }, [userSession, fetchUserChatRooms, fetchAllUsers]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSendMessage = useCallback(() => {
    if (!selectedRoom || messageInput.trim() === "") return;
    const messageToSend = {
      roomId: selectedRoom.id,
      content: messageInput,
      type: "normal",
    } as const;
    sendMessage(messageToSend);
    setMessageInput("");
  }, [selectedRoom, messageInput, sendMessage]);

  const handleInviteClick = useCallback(
    (room:ChatRoom) => {
      setChatRoomToInvite(room);
      openInviteModal();
    },
    [openInviteModal]
  );

  const handleInviteSuccess = useCallback(
    (updatedChatRoom:ChatRoom) => {
      fetchUserChatRooms();
      if (selectedRoom?.id === updatedChatRoom.id) {
        setSelectedRoom(updatedChatRoom);
      }
    },
    [selectedRoom, fetchUserChatRooms, setSelectedRoom]
  );

  if (!userSession) {
    return null; // 或者顯示一個載入中的畫面
  }

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
