# React Native + Expo Skill Guide

## Project Layout (Expo Router)

```
app/
├── package.json
├── app.json
├── eas.json
├── tsconfig.json
├── app/                        # Expo Router file-based routing
│   ├── _layout.tsx             # Root layout
│   ├── (tabs)/
│   │   ├── _layout.tsx         # Tab bar layout
│   │   ├── index.tsx           # /
│   │   └── settings.tsx        # /settings
│   ├── (auth)/
│   │   ├── login.tsx           # /login
│   │   └── register.tsx        # /register
│   └── users/
│       ├── index.tsx           # /users
│       └── [id].tsx            # /users/:id
├── components/
├── hooks/
├── stores/
└── constants/
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "expo": "<version>",
    "expo-router": "<version>",
    "react": "<version>",
    "react-native": "<version>",
    "expo-status-bar": "<version>",
    "expo-image": "<version>",
    "zustand": "<version>"
  },
  "devDependencies": {
    "@babel/core": "<version>",
    "typescript": "<version>"
  }
}
```

## Root Layout

```typescript
// app/_layout.tsx
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useColorScheme } from 'react-native';

export default function RootLayout() {
  const colorScheme = useColorScheme();

  return (
    <>
      <StatusBar style={colorScheme === 'dark' ? 'light' : 'dark'} />
      <Stack>
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        <Stack.Screen name="(auth)/login" options={{ title: 'Login', presentation: 'modal' }} />
        <Stack.Screen name="users/[id]" options={{ title: 'User Detail' }} />
      </Stack>
    </>
  );
}
```

## Tab Layout

```typescript
// app/(tabs)/_layout.tsx
import { Tabs } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';

export default function TabLayout() {
  return (
    <Tabs screenOptions={{ tabBarActiveTintColor: '#007AFF' }}>
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="home" size={size} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="settings"
        options={{
          title: 'Settings',
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="settings" size={size} color={color} />
          ),
        }}
      />
    </Tabs>
  );
}
```

## Page with Dynamic Route

```typescript
// app/users/[id].tsx
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect, useState } from 'react';
import { View, Text, ActivityIndicator, StyleSheet } from 'react-native';

export default function UserDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUser(id).then(u => { setUser(u); setLoading(false); });
  }, [id]);

  if (loading) return <ActivityIndicator style={styles.center} />;
  if (!user) return <Text>User not found</Text>;

  return (
    <View style={styles.container}>
      <Text style={styles.title}>{user.name}</Text>
      <Text>{user.email}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 16 },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  title: { fontSize: 24, fontWeight: '700', marginBottom: 8 },
});
```

## StyleSheet.create

```typescript
import { StyleSheet, Platform } from 'react-native';

const styles = StyleSheet.create({
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    ...Platform.select({
      ios: {
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
      },
      android: { elevation: 3 },
    }),
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: '#1a1a1a',
  },
});
```

## FlatList (Performance)

```typescript
import { FlatList, View, Text, RefreshControl } from 'react-native';

function UserList({ users, onRefresh, refreshing }: Props) {
  return (
    <FlatList
      data={users}
      keyExtractor={item => item.id}
      renderItem={({ item }) => <UserCard user={item} />}
      ItemSeparatorComponent={() => <View style={{ height: 8 }} />}
      contentContainerStyle={{ padding: 16 }}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
      }
      ListEmptyComponent={<Text>No users found</Text>}
    />
  );
}
```

## expo-image

```typescript
import { Image } from 'expo-image';

<Image
  source={{ uri: user.avatar }}
  style={{ width: 48, height: 48, borderRadius: 24 }}
  placeholder={{ blurhash: user.avatarBlurhash }}
  contentFit="cover"
  transition={200}
/>
```

## Zustand State Store

```typescript
// stores/useUserStore.ts
import { create } from 'zustand';

interface User { id: string; name: string; email: string; }
interface UserStore {
  users: User[];
  loading: boolean;
  fetchUsers: () => Promise<void>;
  addUser: (user: User) => void;
}

export const useUserStore = create<UserStore>((set) => ({
  users: [],
  loading: false,
  fetchUsers: async () => {
    set({ loading: true });
    try {
      const users = await api.getUsers();
      set({ users, loading: false });
    } catch {
      set({ loading: false });
    }
  },
  addUser: (user) => set(state => ({ users: [...state.users, user] })),
}));
```

## EAS Build

```json
// eas.json
{
  "cli": { "version": ">= 12.0.0" },
  "build": {
    "development": {
      "developmentClient": true,
      "distribution": "internal"
    },
    "preview": {
      "distribution": "internal",
      "channel": "preview"
    },
    "production": {
      "channel": "production"
    }
  },
  "submit": {
    "production": {}
  }
}
```

```bash
# Build for internal testing
eas build --platform ios --profile preview

# OTA update (no store review)
eas update --branch preview --message "Fix login bug"

# Submit to stores
eas submit --platform ios --profile production
```

## Bare Workflow (Native Modules)

When you need native modules not in Expo Go:
```bash
npx expo prebuild          # Generates ios/ and android/ directories
npx expo run:ios           # Build and run on simulator
npx expo run:android       # Build and run on emulator
```

## Key Rules

- Use `StyleSheet.create` for all styles — never use inline objects (causes re-renders).
- Use `FlatList` for all lists; never `.map()` into a `ScrollView` for large data.
- Use `expo-image` instead of `<Image>` from React Native for better performance and caching.
- Use `useLocalSearchParams` (not `useSearchParams`) for Expo Router params.
- Use `keyExtractor` with stable unique IDs in FlatList — never use array index.
- Use `Platform.select` or `Platform.OS` for platform-specific styles/behavior.
- EAS Build for all production builds; OTA updates (`eas update`) for JS-only changes.
- Managed workflow (no native code) is preferred; use bare workflow only when required.
- All environment variables via `process.env.EXPO_PUBLIC_*` (exposed) or app.config.ts (server).
