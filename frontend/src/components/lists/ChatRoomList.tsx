// src/components/lists/ChatRoomList.tsx
import React from "react";
import {
  Stack,
  Text,
  Divider,
  UnstyledButton,
  Avatar,
  rem,
  Group,
  Menu,
  ActionIcon,
} from "@mantine/core";
import {
  IconMessageCircle,
  IconLogout,
  IconDotsVertical,
  IconUserPlus, 
} from "@tabler/icons-react";

import type { ChatRoom } from "../../types"; // 注意路徑，確保 ChatRoom 類型已定義

interface ChatRoomListProps {
  chatRooms: ChatRoom[];
  selectedRoomId: string | null;
  onSelectRoom: (room: ChatRoom) => void;
  onLeaveRoom: (room: ChatRoom) => void;
  // 新增 onInviteClick 屬性
  onInviteClick: (room: ChatRoom) => void;
}

const ChatRoomList: React.FC<ChatRoomListProps> = ({
  chatRooms,
  selectedRoomId,
  onSelectRoom,
  onLeaveRoom,
  onInviteClick, // 接收 onInviteClick
}) => {
  return (
    <div>
      <Text size="lg" fw={600} mb="md">
        聊天室列表
      </Text>
      <Divider mb="sm" />
      {chatRooms?.length === 0 ? (
        <Text c="dimmed" size="sm" mb="md">
          尚無聊天室。請從下方選擇使用者建立聊天室。
        </Text>
      ) : (
        <Stack gap="md">
          {chatRooms.map((room) => (
            <Group key={room.id} wrap="nowrap" justify="space-between">
              <UnstyledButton
                onClick={() => onSelectRoom(room)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  padding: rem(10),
                  borderRadius: "var(--mantine-radius-sm)",
                  backgroundColor:
                    selectedRoomId === room.id
                      ? "var(--mantine-color-blue-0)"
                      : "transparent",
                  flex: 1,
                }}
              >
                <Avatar color="blue" radius="xl">
                  <IconMessageCircle size={24} />
                </Avatar>
                <Text ml="md" fw={500}>
                  {room.name}
                </Text>
              </UnstyledButton>
              <Menu>
                <Menu.Target>
                  <ActionIcon variant="subtle" color="gray">
                    <IconDotsVertical size={16} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  {/* 邀請選項 */}
                  <Menu.Item
                    leftSection={<IconUserPlus size={14} />}
                    onClick={() => {
                      // 這裡可以觸發一個 props 傳入的 onInviteClick 函式
                      // 或者像現在這樣，先顯示一個通知
                      onInviteClick(room);
                    }}
                  >
                    邀請
                  </Menu.Item>
                  <Menu.Item
                    color="red"
                    leftSection={<IconLogout size={14} />}
                    onClick={() => onLeaveRoom(room)}
                  >
                    退出聊天室
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          ))}
        </Stack>
      )}
    </div>
  );
};

export default ChatRoomList;
