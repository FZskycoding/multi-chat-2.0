// src/components/chat/MessageInput.tsx
import React from "react";
import { Group, ActionIcon, TextInput, Button } from "@mantine/core";
import { IconSend, IconPhoto } from "@tabler/icons-react";

interface MessageInputProps {
  messageInput: string;
  onMessageInputChange: (value: string) => void;
  onSendMessage: () => void; // 新增這個 prop
  isDisabled?: boolean; // 可選，用於禁用輸入和發送
}

const MessageInput: React.FC<MessageInputProps> = ({
  messageInput,
  onMessageInputChange,
  onSendMessage, // 解構出來
  isDisabled = false, // 預設為 false
}) => {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      onSendMessage(); // 在這裡調用傳入的 onSendMessage
    }
  };

  return (
    <Group wrap="nowrap" align="flex-end">
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
        onChange={(event) => onMessageInputChange(event.currentTarget.value)}
        onKeyDown={handleKeyDown} // 使用新的 handleKeyDown 處理器
        size="md"
        disabled={isDisabled}
      />
      <Button
        size="md"
        onClick={onSendMessage} // 在這裡調用傳入的 onSendMessage
        leftSection={<IconSend size={18} />}
        disabled={isDisabled}
      >
        發送
      </Button>
    </Group>
  );
};

export default MessageInput;
