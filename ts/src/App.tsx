import React from 'react';
import { SafeAreaView, StyleSheet, StatusBar } from 'react-native';
import DraftScreen from './screens/DraftScreen';

export default function App() {
  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="light-content" backgroundColor="#0f1419" />
      <DraftScreen />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f1419',
  },
});