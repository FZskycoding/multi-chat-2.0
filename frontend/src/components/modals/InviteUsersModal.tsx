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
} from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import type { User, ChatRoom } from "../../types"; // 引入 User 和 ChatRoom 類型

interface InviteUsersModalProps {
  opened: boolean;
  onClose: () => void;
  chatRoom: ChatRoom; // 要邀請用戶進入的聊天室
  allUsers: User[]; // 所有註冊用戶的列表
  onInvite: (selectedUserIds: string[], roomId: string) => void; // 假設的邀請處理函數 (暫時不實作)
}

const InviteUsersModal: React.FC<InviteUsersModalProps> = ({
  opened,
  onClose,
  chatRoom,
  allUsers,
  onInvite, // 暫時未實作功能
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  // 過濾掉已經在聊天室內的用戶，並根據搜尋詞過濾
  const availableUsers = useMemo(() => {
    if (!allUsers || !chatRoom || !chatRoom.participants) {
      return [];
    }

    // 確保 participantIds 是有效的字串集合
    // 這裡增加一層過濾，確保每個 participant 都有有效的 id
    const participantIds = new Set(
      chatRoom.participants.filter(
        (id) => typeof id === "string" && id.length > 0 // 確保 ID 是有效的字串
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
          onClick={() => {
            // 目前只觸發提示，功能尚未實作
            onInvite(selectedUserIds, chatRoom.id);
            // 清空選中狀態和搜尋詞
            setSelectedUserIds([]);
            setSearchTerm("");
          }}
          disabled={selectedUserIds.length === 0} // 沒有選中用戶時禁用
        >
          邀請 ({selectedUserIds.length})
        </Button>
      </Group>
    </Modal>
  );
};

export default InviteUsersModal;
