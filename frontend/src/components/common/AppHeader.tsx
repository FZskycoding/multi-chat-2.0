import React from "react";
import { Group, Text, Button } from "@mantine/core";
import { IconLogout } from "@tabler/icons-react";

interface AppHeaderProps {
  username: string;
  onLogout: () => void;
}

const AppHeader: React.FC<AppHeaderProps> = ({ username, onLogout }) => {
  return (
    <Group justify="space-between" style={{ flex: 1 }}>
      <Text size="xl" fw={700}>
        GoChat
      </Text>
      <Group>
        <Text fw={500}>歡迎，{username}！</Text>
        <Button
          variant="light"
          onClick={onLogout}
          leftSection={<IconLogout size={16} />}
        >
          登出
        </Button>
      </Group>
    </Group>
  );
};

export default AppHeader;
