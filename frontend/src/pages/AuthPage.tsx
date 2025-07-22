// src/pages/AuthPage.tsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Anchor,
  Group,
  Stack,
  Flex,
} from "@mantine/core";
import { useForm } from "@mantine/form";
import { register, login } from "../api/api_auth";
import { saveUserSession } from "../utils/utils_auth";

function AuthPage() {
  const [isRegister, setIsRegister] = useState(false);
  const navigate = useNavigate();

  const registerForm = useForm({
    initialValues: {
      email: "",
      username: "",
      password: "",
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : "無效的電子郵件"),
      username: (value) => (value.length >= 3 ? null : "使用者名稱至少3個字元"),
      password: (value) => (value.length >= 6 ? null : "密碼至少6個字元"),
    },
  });

  const loginForm = useForm({
    initialValues: {
      email: "",
      password: "",
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : "無效的電子郵件"),
      password: (value) => (value.length >= 6 ? null : "密碼至少6個字元"),
    },
  });

  const handleRegister = async (values: typeof registerForm.values) => {
    try {
      const response = await register(values);
      if (response.id) {
        // 註冊成功後，自動切換到登入介面
        setIsRegister(false);
        loginForm.setFieldValue("email", values.email); // 填充已註冊的 Email
        loginForm.setFieldValue("password", ""); // 顯式清空密碼欄位
      }
    } catch (error) {
      console.error("註冊處理失敗:", error);
      // 錯誤訊息已由 api/auth.ts 中的 notifications 處理
    }
  };

  const handleLogin = async (values: typeof loginForm.values) => {
    try {
      const response = await login(values);
      if (response.id && response.username && response.token) {
        saveUserSession({
          id: response.id,
          username: response.username,
          token: response.token,
        });
        navigate("/home"); // 登入成功後導航到首頁
      }
    } catch (error) {
      console.error("登入處理失敗:", error);
      // 錯誤訊息已由 api/auth.ts 中的 notifications 處理
    }
  };

  return (
    <Flex
      mih="100vh"
      justify="center"
      align="center"
      bg="var(--mantine-color-gray-0)" // 輕微的背景色
    >
      <Paper radius="md" p="xl" withBorder w={400}>
        <Title order={2} size="h1" fw={900} ta="center" mt="md" mb={30}>
          {isRegister ? "註冊帳號" : "登入您的帳號"}
        </Title>

        {isRegister ? (
          <form onSubmit={registerForm.onSubmit(handleRegister)}>
            <Stack>
              <TextInput
                label="電子郵件"
                placeholder="您的電子郵件"
                value={registerForm.values.email}
                onChange={(event) =>
                  registerForm.setFieldValue("email", event.currentTarget.value)
                }
                error={registerForm.errors.email}
                radius="md"
              />
              <TextInput
                label="使用者名稱"
                placeholder="您的使用者名稱"
                value={registerForm.values.username}
                onChange={(event) =>
                  registerForm.setFieldValue(
                    "username",
                    event.currentTarget.value
                  )
                }
                error={registerForm.errors.username}
                radius="md"
              />
              <PasswordInput
                label="密碼"
                placeholder="您的密碼"
                value={registerForm.values.password}
                onChange={(event) =>
                  registerForm.setFieldValue(
                    "password",
                    event.currentTarget.value
                  )
                }
                error={registerForm.errors.password}
                radius="md"
              />
            </Stack>

            <Group justify="space-between" mt="xl">
              <Anchor
                component="button"
                type="button"
                c="dimmed"
                onClick={() => setIsRegister(false)}
                size="xs"
              >
                已經有帳號了？立即登入
              </Anchor>
              <Button type="submit" radius="md">
                註冊
              </Button>
            </Group>
          </form>
        ) : (
          <form onSubmit={loginForm.onSubmit(handleLogin)}>
            <Stack>
              <TextInput
                label="電子郵件"
                placeholder="您的電子郵件"
                value={loginForm.values.email}
                onChange={(event) =>
                  loginForm.setFieldValue("email", event.currentTarget.value)
                }
                error={loginForm.errors.email}
                radius="md"
              />
              <PasswordInput
                label="密碼"
                placeholder="您的密碼"
                value={loginForm.values.password}
                onChange={(event) =>
                  loginForm.setFieldValue("password", event.currentTarget.value)
                }
                error={loginForm.errors.password}
                radius="md"
              />
            </Stack>

            <Group justify="space-between" mt="xl">
              <Anchor
                component="button"
                type="button"
                c="dimmed"
                onClick={() => setIsRegister(true)}
                size="xs"
              >
                還沒有帳號？立即註冊
              </Anchor>
              <Button type="submit" radius="md">
                登入
              </Button>
            </Group>
          </form>
        )}
      </Paper>
    </Flex>
  );
}

export default AuthPage;
