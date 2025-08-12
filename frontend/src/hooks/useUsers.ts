// src/hooks/useUsers.ts
import { useState, useEffect, useCallback } from "react";
import { getAllUsers } from "../api/api_user";
import type { User } from "../types";

export const useUsers = (currentUser: User | null) => {
  const [allUsers, setAllUsers] = useState<User[]>([]);

  const fetchAllUsers = useCallback(async () => {
    if (currentUser) {
      const users = await getAllUsers();
      setAllUsers(users.filter((u: User) => u.id !== currentUser.id));
    }
  }, [currentUser]);

  useEffect(() => {
    fetchAllUsers();
  }, [fetchAllUsers]);

  return { allUsers, fetchAllUsers };
};
