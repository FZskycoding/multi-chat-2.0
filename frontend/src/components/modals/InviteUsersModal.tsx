// src/components/modals/InviteUsersModal.tsx
import React, { useState, useMemo } from "react";
import {
  Modal,
  TextInput,
  ScrollArea,
  Stack,
  Text,
  Avatar,
  Group,
  Button,
  Divider,
  Loader,
} from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import type { User, ChatRoom } from "../../types"; // 引入 User 和 ChatRoom 類型
import { addParticipantsToChatRoom } from "../../api/api_chatroom";

interface InviteUsersModalProps {
  opened: boolean;
  onClose: () => void;
  chatRoom: ChatRoom; // 要邀請用戶進入的聊天室
  allUsers: User[]; // 所有註冊用戶的列表
  onInviteSuccess: (updatedChatRoom: ChatRoom) => void;
}

const InviteUsersModal: React.FC<InviteUsersModalProps> = ({
  opened,
  onClose,
  chatRoom,
  allUsers,
  onInviteSuccess, 
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // 過濾掉已經在聊天室內的用戶，並根據搜尋詞過濾
  const availableUsers = useMemo(() => {
    if (!allUsers || !chatRoom || !chatRoom.participants) {
      return [];
    }

    const participantIds = new Set(
      chatRoom.participants.filter(
        (id) => typeof id === "string" && id.length > 0 
      )
    );

    return allUsers.filter((user) => {
      // 確保 user 有有效的 id 屬性
      const isUserInChatRoom =
        typeof user.id === "string" &&
        user.id.length > 0 &&
        participantIds.has(user.id);
      const matchesSearchTerm = user.username
        .toLowerCase()
        .includes(searchTerm.toLowerCase());

      return !isUserInChatRoom && matchesSearchTerm;
    });
  }, [allUsers, chatRoom, searchTerm]);

  // 模擬點擊選中用戶，目前只用於介面展示選中狀態
  const handleUserSelect = (userId: string) => {
    setSelectedUserIds((prev) =>
      prev.includes(userId)
        ? prev.filter((id) => id !== userId)
        : [...prev, userId]
    );
  };

  const handleInvite = async () => {
    if (selectedUserIds.length === 0) {
      notifications.show({
        title: "提示",
        message: "請選擇至少一位用戶進行邀請。",
        color: "yellow",
      });
      return;
    }

    setIsLoading(true);
    try {
      // 呼叫後端 API
      const updatedRoom = await addParticipantsToChatRoom(
        chatRoom.id,
        selectedUserIds
      );

      notifications.show({
        title: "成功",
        message: "邀請已送出，用戶已加入聊天室。",
        color: "green",
      });
      onInviteSuccess(updatedRoom); // 傳遞更新後的聊天室資訊給父組件
      onClose(); // 關閉 Modal
      // 清空選中狀態和搜尋詞
      setSelectedUserIds([]);
      setSearchTerm("");
    } catch (error: unknown) {
      console.error("邀請失敗:", error);
      let errorMessage = "邀請失敗";
      if (error instanceof Error) {
        errorMessage = `邀請失敗: ${error.message}`;
      }
      notifications.show({
        title: "錯誤",
        message: errorMessage,
        color: "red",
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={`邀請成員至：${chatRoom.name}`}
      size="lg"
      overlayProps={{
        backgroundOpacity: 0.55,
        blur: 3,
      }}
    >
      <TextInput
        placeholder="搜尋用戶名稱"
        leftSection={<IconSearch size={16} />}
        value={searchTerm}
        onChange={(event) => setSearchTerm(event.currentTarget.value)}
        mb="md"
      />

      <Divider my="sm" label="可邀請用戶" labelPosition="center" />

      {availableUsers.length === 0 ? (
        <Text c="dimmed" ta="center" py="xl">
          沒有符合條件的用戶可供邀請。
        </Text>
      ) : (
        <ScrollArea h={300} type="auto" offsetScrollbars scrollbarSize={8}>
          <Stack gap="xs">
            {availableUsers.map((user) => (
              <Group
                key={user.id}
                p="xs"
                style={{
                  cursor: "pointer",
                  borderRadius: "var(--mantine-radius-sm)",
                  backgroundColor: selectedUserIds.includes(user.id)
                    ? "var(--mantine-color-blue-light)"
                    : "transparent",
                  "&:hover": {
                    backgroundColor: "var(--mantine-color-gray-0)",
                  },
                }}
                onClick={() => handleUserSelect(user.id)}
              >
                <Avatar color="cyan" radius="xl">
                  {user.username.charAt(0).toUpperCase()}
                </Avatar>
                <Text fw={500}>{user.username}</Text>
              </Group>
            ))}
          </Stack>
        </ScrollArea>
      )}

      <Divider my="sm" />

      <Group justify="flex-end" mt="md">
        <Button variant="default" onClick={onClose}>
          取消
        </Button>
        <Button
          onClick={handleInvite}
          disabled={selectedUserIds.length === 0 || isLoading}
        >
          {isLoading ? (
            <Loader size="sm" color="white" />
          ) : (
            `邀請 (${selectedUserIds.length})`
          )}
        </Button>
      </Group>
    </Modal>
  );
};

export default InviteUsersModal;
