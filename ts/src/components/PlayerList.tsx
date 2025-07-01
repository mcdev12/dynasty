import React, { useState } from 'react';
import { View, Text, ScrollView, TouchableOpacity, TextInput, StyleSheet } from 'react-native';
import { EnrichedPlayer, POSITION_COLORS } from '../types/draft';

interface PlayerListProps {
  players: EnrichedPlayer[];
  selectedPlayerId: string | null;
  onSelectPlayer: (playerId: string) => void;
  isUserTurn: boolean;
  onMakePick: () => void;
}

export default function PlayerList({
  players,
  selectedPlayerId,
  onSelectPlayer,
  isUserTurn,
  onMakePick,
}: PlayerListProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [positionFilter, setPositionFilter] = useState<string | null>(null);

  const filteredPlayers = players.filter((player) => {
    const matchesSearch = player.fullName.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesPosition = !positionFilter || player.position === positionFilter;
    return matchesSearch && matchesPosition;
  });

  const positions = ['QB', 'RB', 'WR', 'TE', 'K', 'DEF'];

  return (
    <View style={styles.container}>
      <View style={styles.searchContainer}>
        <TextInput
          style={styles.searchInput}
          placeholder="Search players..."
          placeholderTextColor="#4a5568"
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      <ScrollView horizontal style={styles.positionFilters} showsHorizontalScrollIndicator={false}>
        <TouchableOpacity
          style={[styles.filterButton, !positionFilter && styles.filterButtonActive]}
          onPress={() => setPositionFilter(null)}
        >
          <Text style={[styles.filterText, !positionFilter && styles.filterTextActive]}>All</Text>
        </TouchableOpacity>
        {positions.map((pos) => (
          <TouchableOpacity
            key={pos}
            style={[styles.filterButton, positionFilter === pos && styles.filterButtonActive]}
            onPress={() => setPositionFilter(pos)}
          >
            <Text style={[styles.filterText, positionFilter === pos && styles.filterTextActive]}>
              {pos}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <ScrollView style={styles.playerList}>
        {filteredPlayers.map((player) => (
          <TouchableOpacity
            key={player.id}
            style={[
              styles.playerRow,
              selectedPlayerId === player.id && styles.playerRowSelected,
            ]}
            onPress={() => onSelectPlayer(player.id)}
          >
            <View style={styles.rankBadge}>
              <Text style={styles.rankText}>{player.rank}</Text>
            </View>
            <View
              style={[
                styles.positionBadge,
                { backgroundColor: POSITION_COLORS[player.position as keyof typeof POSITION_COLORS] },
              ]}
            >
              <Text style={styles.positionText}>{player.position}</Text>
            </View>
            <View style={styles.playerInfo}>
              <Text style={styles.playerName}>{player.fullName}</Text>
              <Text style={styles.playerTeam}>
                {player.nflTeam || 'FA'} • {player.projection || 'N/A pts'}
              </Text>
            </View>
          </TouchableOpacity>
        ))}
      </ScrollView>

      {isUserTurn && selectedPlayerId && (
        <TouchableOpacity style={styles.makePickButton} onPress={onMakePick}>
          <Text style={styles.makePickButtonText}>Make Pick</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  searchContainer: {
    padding: 12,
  },
  searchInput: {
    backgroundColor: '#0f1419',
    borderRadius: 8,
    padding: 10,
    color: '#ffffff',
    fontSize: 14,
  },
  positionFilters: {
    paddingHorizontal: 12,
    marginBottom: 8,
    maxHeight: 40,
  },
  filterButton: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    marginRight: 8,
    borderRadius: 16,
    backgroundColor: '#2d3748',
  },
  filterButtonActive: {
    backgroundColor: '#00d4aa',
  },
  filterText: {
    color: '#a0aec0',
    fontSize: 12,
    fontWeight: '600',
  },
  filterTextActive: {
    color: '#0f1419',
  },
  playerList: {
    flex: 1,
  },
  playerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
  },
  playerRowSelected: {
    backgroundColor: 'rgba(0, 212, 170, 0.1)',
  },
  rankBadge: {
    width: 24,
    height: 24,
    borderRadius: 4,
    backgroundColor: '#4a5568',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 8,
  },
  rankText: {
    color: '#ffffff',
    fontSize: 11,
    fontWeight: 'bold',
  },
  positionBadge: {
    width: 20,
    height: 20,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 8,
  },
  positionText: {
    color: '#ffffff',
    fontSize: 10,
    fontWeight: 'bold',
  },
  playerInfo: {
    flex: 1,
  },
  playerName: {
    color: '#ffffff',
    fontSize: 13,
    fontWeight: 'bold',
  },
  playerTeam: {
    color: '#a0aec0',
    fontSize: 11,
    marginTop: 2,
  },
  makePickButton: {
    margin: 12,
    backgroundColor: '#00d4aa',
    borderRadius: 8,
    padding: 12,
    alignItems: 'center',
  },
  makePickButtonText: {
    color: '#0f1419',
    fontSize: 16,
    fontWeight: 'bold',
  },
});