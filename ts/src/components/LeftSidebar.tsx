import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import PlayerList from './PlayerList';
import MyTeamTab from './MyTeamTab';
import { EnrichedPlayer, Team, CurrentPickState } from '../types/draft';

interface LeftSidebarProps {
  availablePlayers: EnrichedPlayer[];
  teams: Team[];
  onMakePick: (playerId: string) => void;
  currentPick?: CurrentPickState;
  userTeamId?: string;
}

type TabType = 'players' | 'myteam';

export default function LeftSidebar({ 
  availablePlayers, 
  teams, 
  onMakePick, 
  currentPick,
  userTeamId 
}: LeftSidebarProps) {
  const [activeTab, setActiveTab] = useState<TabType>('players');
  const [selectedPlayerId, setSelectedPlayerId] = useState<string | null>(null);

  const isUserTurn = currentPick && currentPick.teamId === userTeamId;

  const handleMakePick = () => {
    if (selectedPlayerId && isUserTurn) {
      onMakePick(selectedPlayerId);
      setSelectedPlayerId(null);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.tabContainer}>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'players' && styles.activeTab]}
          onPress={() => setActiveTab('players')}
        >
          <Text style={[styles.tabText, activeTab === 'players' && styles.activeTabText]}>
            Players
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'myteam' && styles.activeTab]}
          onPress={() => setActiveTab('myteam')}
        >
          <Text style={[styles.tabText, activeTab === 'myteam' && styles.activeTabText]}>
            My Team
          </Text>
        </TouchableOpacity>
      </View>

      <View style={styles.content}>
        {activeTab === 'players' && (
          <PlayerList
            players={availablePlayers}
            selectedPlayerId={selectedPlayerId}
            onSelectPlayer={setSelectedPlayerId}
            isUserTurn={isUserTurn || false}
            onMakePick={handleMakePick}
          />
        )}
        {activeTab === 'myteam' && (
          <MyTeamTab userTeamId={userTeamId} />
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    width: 300,
    backgroundColor: '#1a1f2e',
    borderRightWidth: 1,
    borderRightColor: '#2d3748',
  },
  tabContainer: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
  },
  tab: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
  },
  activeTab: {
    borderBottomWidth: 2,
    borderBottomColor: '#00d4aa',
  },
  tabText: {
    color: '#a0aec0',
    fontSize: 14,
    fontWeight: '600',
  },
  activeTabText: {
    color: '#00d4aa',
  },
  content: {
    flex: 1,
  },
});