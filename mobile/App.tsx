import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { DarkTheme, DefaultTheme, NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { SQLiteProvider } from 'expo-sqlite';
import { StatusBar } from 'expo-status-bar';
import { BookOpen, PenLine, Settings } from 'lucide-react-native';
import { Component, type PropsWithChildren, Suspense } from 'react';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { copy } from './src/copy';
import { DesktopScreen, SettingsScreen } from './src/desktop-screens';
import { DraftScreen } from './src/draft-screen';
import { PublicationScreen } from './src/publication-screen';
import { ArticleScreen, SubscribeScreen, SubscriptionsScreen } from './src/reader-screens';
import { ReadingScreen } from './src/reading-screen';
import type { Routes, Tabs } from './src/routes';
import { initializeStorage } from './src/storage';
import { usePalette } from './src/theme';
import { Button, Loading, Notice, Page, Screen } from './src/ui';
import { WritingScreen } from './src/writing-screen';

const Stack = createNativeStackNavigator<Routes>();
const Tab = createBottomTabNavigator<Tabs>();

function Home() {
  const { colors } = usePalette();
  return <Tab.Navigator screenOptions={{
    headerShown: false,
    tabBarActiveTintColor: colors.accent,
    tabBarInactiveTintColor: colors.muted,
    tabBarStyle: { backgroundColor: colors.paper, borderTopColor: colors.line },
    tabBarLabelStyle: { fontSize: 12, fontWeight: '600' },
    tabBarHideOnKeyboard: true,
  }}>
    <Tab.Screen name="Writing" component={WritingScreen} options={{
      title: copy.writing, tabBarIcon: ({ color, size }) => <PenLine size={size} color={color} />,
    }} />
    <Tab.Screen name="Reading" component={ReadingScreen} options={{
      title: copy.reading, tabBarIcon: ({ color, size }) => <BookOpen size={size} color={color} />,
    }} />
    <Tab.Screen name="Settings" component={SettingsScreen} options={{
      title: copy.settings, tabBarIcon: ({ color, size }) => <Settings size={size} color={color} />,
    }} />
  </Tab.Navigator>;
}

class StorageBoundary extends Component<PropsWithChildren, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  render() {
    if (this.state.failed) return <Screen tab><Page>
      <Notice text={copy.storageError} error />
      <Button label={copy.retry} onPress={() => this.setState({ failed: false })} />
    </Page></Screen>;
    return this.props.children;
  }
}

export default function App() {
  const { colors, isDark } = usePalette();
  const defaultTheme = isDark ? DarkTheme : DefaultTheme;
  const theme = {
    ...defaultTheme,
    colors: {
      ...defaultTheme.colors,
      primary: colors.accent, background: colors.background, card: colors.paper,
      text: colors.text, border: colors.line, notification: colors.accent,
    },
  };
  return <SafeAreaProvider>
    <StatusBar style={isDark ? 'light' : 'dark'} />
    <StorageBoundary>
      <Suspense fallback={<Screen tab><Page><Loading /></Page></Screen>}>
        <SQLiteProvider databaseName="kinpage.db" onInit={initializeStorage} useSuspense>
          <NavigationContainer theme={theme}>
            <Stack.Navigator screenOptions={{
              headerStyle: { backgroundColor: colors.background },
              headerTintColor: colors.text,
              headerBackButtonDisplayMode: 'minimal',
              contentStyle: { backgroundColor: colors.background },
            }}>
              <Stack.Screen name="Home" component={Home} options={{ headerShown: false }} />
              <Stack.Screen name="Draft" component={DraftScreen} options={{ title: copy.draft }} />
              <Stack.Screen name="Publish" component={PublicationScreen} options={{ title: copy.publish }} />
              <Stack.Screen name="Article" component={ArticleScreen} options={{ title: copy.article }} />
              <Stack.Screen name="Subscribe" component={SubscribeScreen} options={{ title: copy.addFeed }} />
              <Stack.Screen name="Subscriptions" component={SubscriptionsScreen} options={{ title: copy.manageFeeds }} />
              <Stack.Screen name="Desktop" component={DesktopScreen} options={{ title: copy.desktop }} />
            </Stack.Navigator>
          </NavigationContainer>
        </SQLiteProvider>
      </Suspense>
    </StorageBoundary>
  </SafeAreaProvider>;
}
