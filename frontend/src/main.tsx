import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom"; // 引入 BrowserRouter
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications"; // 引入 Notifications
import App from "./App.tsx";
import "@mantine/core/styles.css"; // 引入 Mantine 基礎樣式
import "@mantine/notifications/styles.css"; // 引入 Mantine Notifications 樣式

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider defaultColorScheme="light">
      {" "}
      {/* 可以設定預設主題色，例如 "light" 或 "dark" */}
      <Notifications /> {/* 放置 Notifications 組件 */}
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>
);
