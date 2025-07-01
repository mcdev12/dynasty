import React from 'react';
import { View, Text, ScrollView, StyleSheet, Animated } from 'react-native';
import { Team, DraftPick, CurrentPickState, DraftSettings, DraftType, POSITION_COLORS } from '../types/draft';

interface DraftGridProps {
  teams: Team[];
  picks: DraftPick[];
  currentPick?: CurrentPickState;
  settings: DraftSettings;
  draftType: DraftType;
}

export default function DraftGrid({ teams, picks, currentPick, settings, draftType }: DraftGridProps) {
  const pulseAnim = React.useRef(new Animated.Value(1)).current;

  React.useEffect(() => {
    Animated.loop(
      Animated.sequence([
        Animated.timing(pulseAnim, {
          toValue: 1.05,
          duration: 1000,
          useNativeDriver: true,
        }),
        Animated.timing(pulseAnim, {
          toValue: 1,
          duration: 1000,
          useNativeDriver: true,
        }),
      ])
    ).start();
  }, []);

  const getPickForCell = (round: number, teamId: string): DraftPick | null => {
    return picks.find(p => p.round === round && p.teamId === teamId) || null;
  };

  const isCurrentPickCell = (round: number, teamId: string): boolean => {
    if (!currentPick) return false;
    return currentPick.round === round && currentPick.teamId === teamId;
  };

  return (
    <ScrollView horizontal style={styles.container}>
      <View style={styles.grid}>
        {/* Header Row */}
        <View style={styles.headerRow}>
          <View style={styles.roundHeader} />
          {teams.map((team) => (
            <View key={team.id} style={[styles.teamHeader, { borderBottomColor: team.color }]}>
              <Text style={styles.teamAvatar}>{team.avatar}</Text>
              <Text style={styles.teamName} numberOfLines={1}>{team.name}</Text>
            </View>
          ))}
        </View>

        {/* Grid Rows */}
        <ScrollView style={styles.gridBody}>
          {Array.from({ length: settings.rounds }, (_, i) => i + 1).map((round) => (
            <View key={round} style={styles.row}>
              <View style={styles.roundCell}>
                <Text style={styles.roundText}>R{round}</Text>
              </View>
              {teams.map((team) => {
                const pick = getPickForCell(round, team.id);
                const isCurrentPick = isCurrentPickCell(round, team.id);

                return (
                  <Animated.View
                    key={`${round}-${team.id}`}
                    style={[
                      styles.pickCell,
                      isCurrentPick && styles.currentPickCell,
                      isCurrentPick && { transform: [{ scale: pulseAnim }] },
                      pick && { backgroundColor: `${team.color}26` }, // 15% opacity
                    ]}
                  >
                    {isCurrentPick && !pick && (
                      <>
                        <Text style={styles.onClockText}>ON CLOCK</Text>
                        <View style={styles.pulsingDot} />
                      </>
                    )}
                    {pick && (
                      <>
                        <Text style={styles.playerName} numberOfLines={1}>
                          {pick.playerId || 'Player'}
                        </Text>
                        <View style={styles.pickInfo}>
                          <View style={[styles.positionBadge, { backgroundColor: POSITION_COLORS.QB }]}>
                            <Text style={styles.positionText}>TBD</Text>
                          </View>
                        </View>
                      </>
                    )}
                    {!pick && !isCurrentPick && (
                      <Text style={styles.emptyPick}>—</Text>
                    )}
                  </Animated.View>
                );
              })}
            </View>
          ))}
        </ScrollView>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#131920',
  },
  grid: {
    flex: 1,
  },
  headerRow: {
    flexDirection: 'row',
    backgroundColor: '#0f1419',
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
    zIndex: 100,
  },
  roundHeader: {
    width: 60,
    height: 80,
  },
  teamHeader: {
    width: 140,
    height: 80,
    padding: 10,
    alignItems: 'center',
    justifyContent: 'center',
    borderBottomWidth: 3,
  },
  teamAvatar: {
    fontSize: 24,
    marginBottom: 4,
  },
  teamName: {
    color: '#ffffff',
    fontSize: 12,
    fontWeight: 'bold',
  },
  gridBody: {
    flex: 1,
  },
  row: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
  },
  roundCell: {
    width: 60,
    height: 60,
    backgroundColor: '#1a1f2e',
    alignItems: 'center',
    justifyContent: 'center',
    borderRightWidth: 1,
    borderRightColor: '#2d3748',
  },
  roundText: {
    color: '#a0aec0',
    fontSize: 14,
    fontWeight: 'bold',
  },
  pickCell: {
    width: 140,
    minHeight: 60,
    padding: 8,
    borderRightWidth: 1,
    borderRightColor: '#2d3748',
    justifyContent: 'center',
  },
  currentPickCell: {
    borderWidth: 2,
    borderColor: '#22c55e',
  },
  onClockText: {
    color: '#22c55e',
    fontSize: 10,
    fontWeight: 'bold',
    textAlign: 'center',
  },
  pulsingDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#22c55e',
    alignSelf: 'center',
    marginTop: 4,
  },
  playerName: {
    color: '#ffffff',
    fontSize: 12,
    fontWeight: 'bold',
    marginBottom: 4,
  },
  pickInfo: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  positionBadge: {
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  positionText: {
    color: '#ffffff',
    fontSize: 10,
    fontWeight: 'bold',
  },
  emptyPick: {
    color: '#4a5568',
    fontSize: 16,
    textAlign: 'center',
  },
});