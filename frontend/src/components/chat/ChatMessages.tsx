// src/components/chat/ChatMessages.tsx
import React, { useEffect } from "react";
import { ScrollArea, Stack, Group, Paper, Text, rem } from "@mantine/core";
import type { Message } from "../../types"; // 注意路徑，確保 Message 類型已定義

interface ChatMessagesProps {
  messages: Message[];
  userSessionId: string; // 用來判斷是自己的訊息還是別人的訊息
  messagesEndRef: React.RefObject<HTMLDivElement|null>; // 從父組件傳入的 ref
}

const ChatMessages: React.FC<ChatMessagesProps> = ({
  messages,
  userSessionId,
  messagesEndRef,
}) => {
  // 當訊息更新時，自動滾動到最新訊息
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, messagesEndRef]); // 依賴 messages 和 messagesEndRef

  return (
    <ScrollArea style={{ flex: 1, marginBottom: rem(10) }}>
      <Stack>
        {messages.map((msg, index) => (
          <Group
            key={msg.id || index} // 使用 msg.id 作為 key，如果沒有則用 index (雖然不推薦，但暫時作為 fallback)
            justify={
              msg.senderId === userSessionId
                ? "flex-end" // 自己的訊息靠右
                : "flex-start" // 其他人的訊息靠左
            }
          >
            {msg.type === "system" ? (
              <Paper
                p="xs"
                radius="md"
                w="20%" // 系統訊息可以佔用較少寬度並居中
                bg="var(--mantine-color-gray-1)"
                style={{ textAlign: "center", margin: "0 auto" }}
              >
                <Text size="sm" c="dimmed">
                  {msg.content}
                </Text>
                <Text size="xs" c="dimmed">
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </Text>
              </Paper>
            ) : (
              <Paper
                p="xs"
                radius="md"
                shadow="xs"
                bg={
                  msg.senderId === userSessionId
                    ? "#c3efab" // 淺綠色：自己的訊息
                    : "#cde2ff" // 淺藍色：其他人的訊息
                }
                style={{ maxWidth: "70%" }} // 訊息最大寬度
              >
                <Text size="xs" c="dark">
                  {msg.senderUsername} (
                  {new Date(msg.timestamp).toLocaleTimeString()})
                </Text>
                <Text>{msg.content}</Text>
              </Paper>
            )}
          </Group>
        ))}
        <div ref={messagesEndRef} /> {/* 滾動錨點 */}
      </Stack>
    </ScrollArea>
  );
};

export default ChatMessages;
