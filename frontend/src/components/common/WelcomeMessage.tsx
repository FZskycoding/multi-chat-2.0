import React from "react";
import { Paper, rem, Title, Text } from "@mantine/core";
import { IconMessageCircle } from "@tabler/icons-react";

const WelcomeMessage: React.FC = () => {
  return (
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
  );
};

export default WelcomeMessage;
