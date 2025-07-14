// src/components/lists/UserList.tsx
import React from "react";
import {
  Stack,
  Text,
  Divider,
  UnstyledButton,
  Avatar,
  rem,
} from "@mantine/core";
import { IconUserCircle } from "@tabler/icons-react";
import type { User } from "../../types"; // 注意路徑

interface UserListProps {
  users: User[];
  onStartChat: (user: User) => void;
}

const UserList: React.FC<UserListProps> = ({ users, onStartChat }) => {
  return (
    <div>
      <Text size="lg" fw={600} mb="md">
        所有使用者
      </Text>
      <Divider mb="sm" />
      {users.length === 0 ? (
        <Text c="dimmed">沒有其他使用者。</Text>
      ) : (
        <Stack gap="md">
          {users.map((user) => (
            <UnstyledButton
              key={user.id}
              onClick={() => onStartChat(user)}
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
  );
};

export default UserList;
