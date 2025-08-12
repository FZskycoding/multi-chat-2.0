// src/hooks/useWebSocket.ts
import { useState, useEffect, useRef, useCallback } from "react";
import { notifications } from "@mantine/notifications";
import { WEBSOCKET_URL } from "../config";
import type { Message } from "../types";

export const useWebSocket = (
  userSession: ReturnType<typeof import("../utils/utils_auth").getUserSession>,
  onMessageReceived: (message: Message) => void,
  onForceLogout: () => void,
  onStateUpdate: () => void
) => {
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!userSession?.token) {
      if (ws.current) ws.current.close();
      return;
    }

    const websocketUrl = `${WEBSOCKET_URL}?token=${userSession.token}`;
    const newWs = new WebSocket(websocketUrl);

    newWs.onopen = () => setIsConnected(true);
    newWs.onclose = () => setIsConnected(false);
    newWs.onerror = () => setIsConnected(false);

    newWs.onmessage = (event: MessageEvent) => {
      const receivedMessage: Message = JSON.parse(event.data);

      if (receivedMessage.type === "force_logout") {
        notifications.show({
          title: "登出通知",
          message:
            receivedMessage.content || "您的帳號已在另一台裝置登入，您已被登出",
          color: "orange",
          autoClose: 5000,
        });
        onForceLogout();
        return;
      }

      if (receivedMessage.type === "room_state_update") {
        onStateUpdate();
      }

      onMessageReceived(receivedMessage);
    };

    ws.current = newWs;

    return () => {
      if (newWs.readyState < 2) {
        newWs.close();
      }
    };
  }, [userSession, onMessageReceived, onForceLogout, onStateUpdate]);

  const sendMessage = useCallback(
    (messageToSend: Partial<Message>) => {
      if (!isConnected || !ws.current) {
        notifications.show({
          title: "錯誤",
          message: "WebSocket 未連線。",
          color: "red",
        });
        return;
      }
      ws.current.send(JSON.stringify(messageToSend));
    },
    [isConnected]
  );

  return { isConnected, sendMessage };
};
