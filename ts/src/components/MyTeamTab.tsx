import React from 'react';
import { View, Text, ScrollView, StyleSheet } from 'react-native';

interface MyTeamTabProps {
  userTeamId?: string;
}

export default function MyTeamTab({ userTeamId }: MyTeamTabProps) {
  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerText}>My Draft Picks</Text>
      </View>
      
      <View style={styles.emptyState}>
        <Text style={styles.emptyText}>No players drafted yet</Text>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
  },
  headerText: {
    color: '#ffffff',
    fontSize: 16,
    fontWeight: 'bold',
  },
  emptyState: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    color: '#a0aec0',
    fontSize: 14,
  },
});