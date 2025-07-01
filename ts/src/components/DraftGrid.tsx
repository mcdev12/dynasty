import React from 'react';
import { View, Text, ScrollView, StyleSheet, Animated, Platform } from 'react-native';
import { Team, DraftPick, CurrentPickState, DraftSettings, DraftType, POSITION_COLORS, Player } from '../types/draft';

interface DraftGridProps {
  teams: Team[];
  picks: DraftPick[];
  currentPick?: CurrentPickState;
  settings: DraftSettings;
  draftType: DraftType;
  playersById: Map<string, Player>;
}

export default function DraftGrid({ teams, picks, currentPick, settings, draftType, playersById }: DraftGridProps) {
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

  // Use native web table for better layout on web
  if (Platform.OS === 'web') {
    return (
      <div style={{ 
        flex: 1, 
        overflow: 'auto', 
        backgroundColor: '#131920',
        fontFamily: 'system-ui, -apple-system, sans-serif'
      }}>
        <table style={{ 
          borderCollapse: 'collapse', 
          width: '100%',
          minWidth: 'fit-content'
        }}>
          {/* Header Row */}
          <thead>
            <tr style={{ backgroundColor: '#0f1419', borderBottom: '1px solid #2d3748' }}>
              <th style={{ 
                width: '60px', 
                height: '80px',
                backgroundColor: '#1a1f2e',
                border: '1px solid #2d3748',
                color: '#a0aec0',
                fontSize: '14px',
                fontWeight: 'bold'
              }}>
                Round
              </th>
              {teams.map((team) => (
                <th key={team.id} style={{ 
                  width: '160px', 
                  height: '80px',
                  padding: '10px',
                  textAlign: 'center',
                  backgroundColor: '#0f1419',
                  borderBottom: `3px solid ${team.color}`,
                  border: '1px solid #2d3748',
                  color: '#ffffff',
                  fontSize: '12px',
                  fontWeight: 'bold'
                }}>
                  <div style={{ fontSize: '24px', marginBottom: '4px' }}>{team.avatar}</div>
                  <div>{team.name}</div>
                </th>
              ))}
            </tr>
          </thead>
          
          {/* Grid Body */}
          <tbody>
            {Array.from({ length: settings.rounds }, (_, i) => i + 1).map((round) => (
              <tr key={round} style={{ borderBottom: '1px solid #2d3748' }}>
                <td style={{ 
                  width: '60px',
                  height: '60px',
                  backgroundColor: '#1a1f2e',
                  textAlign: 'center',
                  border: '1px solid #2d3748',
                  color: '#a0aec0',
                  fontSize: '14px',
                  fontWeight: 'bold'
                }}>
                  R{round}
                </td>
                {teams.map((team) => {
                  const pick = getPickForCell(round, team.id);
                  const isCurrentPick = isCurrentPickCell(round, team.id);

                  return (
                    <td key={`${round}-${team.id}`} style={{ 
                      width: '160px',
                      minHeight: '60px',
                      padding: '8px',
                      border: '1px solid #2d3748',
                      backgroundColor: pick ? `${team.color}26` : 'transparent',
                      borderColor: isCurrentPick ? '#22c55e' : '#2d3748',
                      borderWidth: isCurrentPick ? '2px' : '1px',
                      textAlign: 'center',
                      verticalAlign: 'middle'
                    }}>
                      {isCurrentPick && !pick && (
                        <div>
                          <div style={{ 
                            color: '#22c55e', 
                            fontSize: '10px', 
                            fontWeight: 'bold',
                            marginBottom: '4px'
                          }}>
                            ON CLOCK
                          </div>
                          <div style={{ 
                            width: '8px', 
                            height: '8px', 
                            borderRadius: '50%',
                            backgroundColor: '#22c55e',
                            margin: '0 auto',
                            animation: 'pulse 1s infinite'
                          }} />
                        </div>
                      )}
                      {pick && (
                        <div>
                          <div style={{ 
                            color: '#ffffff', 
                            fontSize: '12px', 
                            fontWeight: 'bold',
                            marginBottom: '2px'
                          }}>
                            {playersById.get(pick.playerId)?.fullName || pick.playerId || 'Player'}
                          </div>
                          {(() => {
                            const player = playersById.get(pick.playerId);
                            const playerProfile = player?.profile?.case === 'playerProfile' ? player.profile.value : null;
                            const nflProfile = playerProfile?.profile?.case === 'nflProfile' ? playerProfile.profile.value : null;
                            return (
                              <div style={{ fontSize: '10px', color: '#a0aec0', marginBottom: '4px' }}>
                                {nflProfile?.position && (
                                  <div style={{ 
                                    padding: '2px 6px',
                                    borderRadius: '4px',
                                    backgroundColor: POSITION_COLORS[nflProfile.position as keyof typeof POSITION_COLORS] || '#6b7280',
                                    color: '#ffffff',
                                    fontSize: '10px',
                                    fontWeight: 'bold',
                                    display: 'inline-block',
                                    marginRight: '4px'
                                  }}>
                                    {nflProfile.position}
                                  </div>
                                )}
                                {nflProfile?.jerseyNumber && (
                                  <span style={{ marginRight: '4px' }}>#{nflProfile.jerseyNumber}</span>
                                )}
                                {nflProfile?.college && (
                                  <div style={{ fontSize: '9px', marginTop: '2px' }}>{nflProfile.college}</div>
                                )}
                              </div>
                            );
                          })()}
                        </div>
                      )}
                      {!pick && !isCurrentPick && (
                        <div style={{ color: '#4a5568', fontSize: '16px' }}>—</div>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
        
        <style>{`
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}</style>
      </div>
    );
  }

  // Fallback to React Native components for mobile
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
                          {playersById.get(pick.playerId)?.fullName || pick.playerId || 'Player'}
                        </Text>
                        <View style={styles.pickInfo}>
                          {(() => {
                            const player = playersById.get(pick.playerId);
                            const playerProfile = player?.profile?.case === 'playerProfile' ? player.profile.value : null;
                            const nflProfile = playerProfile?.profile?.case === 'nflProfile' ? playerProfile.profile.value : null;
                            const position = nflProfile?.position || 'TBD';
                            
                            return (
                              <>
                                <View style={[styles.positionBadge, { backgroundColor: POSITION_COLORS[position as keyof typeof POSITION_COLORS] || '#6b7280' }]}>
                                  <Text style={styles.positionText}>{position}</Text>
                                </View>
                                {nflProfile?.jerseyNumber && (
                                  <Text style={styles.jerseyNumber}>#{nflProfile.jerseyNumber}</Text>
                                )}
                                {nflProfile?.college && (
                                  <Text style={styles.collegeText} numberOfLines={1}>{nflProfile.college}</Text>
                                )}
                              </>
                            );
                          })()}
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
  jerseyNumber: {
    color: '#a0aec0',
    fontSize: 10,
    marginLeft: 4,
  },
  collegeText: {
    color: '#a0aec0',
    fontSize: 9,
    marginTop: 2,
  },
});