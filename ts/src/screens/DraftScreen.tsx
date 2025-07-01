import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import DraftGrid from '../components/DraftGrid';
import DraftTimer from '../components/DraftTimer';
import LeftSidebar from '../components/LeftSidebar';
import { DraftUIState, DraftEvent, DraftStatus, CurrentPickState } from '../types/draft';
import { create } from '@bufbuild/protobuf';
import { DraftService, GetDraftRequestSchema } from '../genproto/draft/v1/draft_service_pb';
import { DraftSettingsSchema, DraftPickSchema } from '../genproto/draft/v1/draft_pb';
import { FantasyTeamService, GetFantasyTeamsByLeagueRequestSchema } from '../genproto/fantasyteam/v1/service_pb';
import { PlayerService, GetPlayerRequestSchema } from '../genproto/player/v1/service_pb';
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { useDraftWebSocket } from '../hooks/useDraftWebSocket';

const DRAFT_ID = 'e6a3e117-e1d0-49bb-8d64-fb0d9217c230'; // TODO: Get from navigation/props
const USER_ID = '667f2190-b9bc-48e2-8683-963b62ac5428'; // TODO: Get from auth/context

// Create Connect client
const transport = createConnectTransport({
  baseUrl: 'http://localhost:8080', // Your Connect RPC server
});
const draftClient = createClient(DraftService, transport);
const fantasyTeamClient = createClient(FantasyTeamService, transport);
const playerClient = createClient(PlayerService, transport);

export default function DraftScreen() {
  const [draftState, setDraftState] = useState<DraftUIState | null>(null);
  const { connected, sendMessage } = useDraftWebSocket(DRAFT_ID, USER_ID, handleDraftEvent);

  // Helper function to calculate which team should pick based on overall pick number (snake draft)
  const getCorrectSnakeTeam = (overallPick: number) => {
    const teams = draftState?.teams || [];
    if (teams.length === 0) return '';
    
    const totalTeams = teams.length;
    const round = Math.ceil(overallPick / totalTeams);
    const pickInRound = ((overallPick - 1) % totalTeams) + 1;
    
    // Snake draft: odd rounds go left-to-right, even rounds go right-to-left
    const isEvenRound = round % 2 === 0;
    let teamIndex;
    if (isEvenRound) {
      // Even round: reverse order (last team from previous round picks first)
      teamIndex = totalTeams - pickInRound; // This gives: pick1->index4, pick2->index3, etc.
    } else {
      // Odd round: normal order (first team picks first)
      teamIndex = pickInRound - 1; // This gives: pick1->index0, pick2->index1, etc.
    }
    
    // Ensure teamIndex is within bounds
    if (teamIndex < 0 || teamIndex >= totalTeams) {
      console.log(`ERROR: teamIndex ${teamIndex} out of bounds for ${totalTeams} teams`);
      teamIndex = 0;
    }
    
    console.log(`Snake calculation: Overall=${overallPick}, Round=${round}, PickInRound=${pickInRound}, IsEven=${isEvenRound}, TeamIndex=${teamIndex}`);
    console.log(`Available teams:`, teams.map(t => ({ id: t.id, name: t.name })));
    console.log(`Selected team:`, teams[teamIndex]);
    
    return teams[teamIndex]?.id || '';
  };

  useEffect(() => {
    fetchDraftData();
  }, []);

  const fetchDraftData = async () => {
    try {
      console.log('Fetching draft data via RPC for draft:', DRAFT_ID);
      
      // Call GetDraft RPC
      const draftResponse = await draftClient.getDraft(
        create(GetDraftRequestSchema, { draftId: DRAFT_ID })
      );
      
      if (!draftResponse.draft) {
        throw new Error('No draft data returned');
      }
      
      const draft = draftResponse.draft;
      console.log('Draft RPC response:', draft);
      
      // Fetch fantasy teams for this league
      const teamsResponse = await fantasyTeamClient.getFantasyTeamsByLeague(
        create(GetFantasyTeamsByLeagueRequestSchema, { leagueId: draft.leagueId })
      );
      
      console.log('Fantasy teams response:', teamsResponse.fantasyTeams);
      
      // Map teams and order them by draft order
      const draftOrder = draft.settings?.draftOrder || [];
      const teamsById = new Map(teamsResponse.fantasyTeams.map(team => [team.id, team]));
      
      // Order teams according to draft order, or use original order if no draft order
      const orderedTeams = draftOrder.length > 0 
        ? draftOrder.map(teamId => teamsById.get(teamId)).filter(Boolean)
        : teamsResponse.fantasyTeams;
      
      console.log('Draft order:', draftOrder);
      console.log('Ordered teams:', orderedTeams);
      console.log('Draft order team IDs:', draftOrder);
      
      // Map teams to UI format
      const uiTeams = orderedTeams.map(team => ({
        id: team!.id,
        name: team!.name,
        avatar: team!.logoUrl ? '🏈' : '👤', // Use emoji for now
        color: '#3b82f6', // Default color for now
      }));
      
      // Map proto response to UI state
      const mappedState: DraftUIState = {
        draft_id: draft.id,
        status: draft.status,
        metadata: {
          draft_type: draft.draftType,
          league_id: draft.leagueId,
          total_rounds: draft.settings?.rounds || 12,
          total_teams: draft.settings?.draftOrder?.length || 12,
        },
        // Include actual draft settings
        settings: draft.settings,
        // No current pick from GetDraft - will come from WebSocket events
        currentPick: undefined,
        // Populated arrays
        teams: uiTeams,
        picks: [], // Will be populated by WebSocket events
        availablePlayers: [], // Will be populated by GetAvailablePlayers
        playersById: new Map(), // Will be populated as players are fetched
      };
      
      console.log('Mapped draft state:', mappedState);
      setDraftState(mappedState);
    } catch (error) {
      console.error('Failed to fetch draft data:', error);
    }
  };

  const handlePickMade = async (pickData: any) => {
    try {
      console.log('handlePickMade called with pickData:', pickData);
      
      // Fetch player details
      const playerResponse = await playerClient.getPlayer(
        create(GetPlayerRequestSchema, { id: pickData.PlayerID })
      );

      if (!playerResponse.player) {
        console.error('Player not found:', pickData.PlayerID);
        return;
      }

      const player = playerResponse.player;
      console.log('Fetched player details:', player);

      // Calculate round and pick from overall pick number
      const overallPick = pickData.Overall || 1;
      const totalTeams = draftState?.metadata.total_teams || 12;
      const round = Math.ceil(overallPick / totalTeams);
      const pickInRound = ((overallPick - 1) % totalTeams) + 1;
      
      // Try to calculate correct snake draft team, fallback to backend
      const correctTeamId = getCorrectSnakeTeam(overallPick);
      const finalTeamId = correctTeamId || pickData.TeamID;
      
      console.log(`Pick details: Overall=${overallPick}, Round=${round}, PickInRound=${pickInRound}`);
      console.log(`Team override: Backend=${pickData.TeamID}, Correct=${correctTeamId}, Final=${finalTeamId}`);

      // Create pick with player details - use calculated team or fallback
      const newPick = create(DraftPickSchema, {
        id: pickData.PickID,
        draftId: DRAFT_ID,
        round: round,
        pick: pickInRound,
        overallPick: overallPick,
        teamId: finalTeamId,
        playerId: pickData.PlayerID,
        pickedAt: pickData.PickedAt ? { seconds: BigInt(Math.floor(new Date(pickData.PickedAt).getTime() / 1000)), nanos: 0 } : undefined,
        keeperPick: pickData.KeeperPick || false,
        auctionAmount: pickData.AuctionAmount,
      });

      console.log('Created pick object:', newPick);

      // Update picks state and add player to map
      setDraftState(prev => prev ? {
        ...prev,
        picks: [...prev.picks.filter(p => p.id !== newPick.id), newPick],
        playersById: new Map([...prev.playersById, [player.id, player]])
      } : null);
      
      console.log('Updated draft state with new pick');
      
    } catch (error) {
      console.error('Failed to fetch player details for pick:', error);
    }
  };

  function handleDraftEvent(event: any) {
    console.log('Draft event:', event);
    
    switch (event.type) {
      case 'PickStarted':
        if (event.data?.started_at && event.data?.time_per_pick_sec) {
          // Calculate round and pick from overall pick number
          const overallPick = event.data.overall_pick || 1;
          const totalTeams = draftState?.metadata.total_teams || 12;
          const round = Math.ceil(overallPick / totalTeams);
          const pickInRound = ((overallPick - 1) % totalTeams) + 1;
          
          // Try to calculate correct snake draft team, fallback to backend
          const correctTeamId = getCorrectSnakeTeam(overallPick);
          const teamId = correctTeamId || event.data.team_id || '';
          
          const newCurrentPick: CurrentPickState = {
            teamId: teamId,
            round: round,
            pickNumber: pickInRound,
            overallPick: overallPick,
            startedAt: event.data.started_at,
            timePerPickSec: event.data.time_per_pick_sec,
          };
          
          console.log('Setting new current pick:', newCurrentPick);
          setDraftState(prev => prev ? {
            ...prev,
            currentPick: newCurrentPick,
            status: DraftStatus.IN_PROGRESS
          } : null);
        }
        break;
        
      case 'PickMade':
        // Add the pick directly to state from WebSocket data and fetch player details
        console.log('PickMade event data:', event.data);
        if (event.data?.PickID && event.data?.PlayerID) {
          console.log('Calling handlePickMade with:', event.data);
          handlePickMade(event.data);
        } else {
          console.log('Missing PickID or PlayerID in PickMade event:', {
            PickID: event.data?.PickID,
            PlayerID: event.data?.PlayerID,
            full_data: event.data
          });
        }
        break;
        
      case 'PickEnded':
        setDraftState(prev => prev ? {
          ...prev,
          currentPick: undefined
        } : null);
        break;
        
      case 'DraftPaused':
        setDraftState(prev => prev ? {
          ...prev,
          status: DraftStatus.PAUSED
        } : null);
        break;
        
      case 'DraftResumed':
        setDraftState(prev => prev ? {
          ...prev,
          status: DraftStatus.IN_PROGRESS
        } : null);
        break;
        
      case 'DraftCompleted':
        setDraftState(prev => prev ? {
          ...prev,
          status: DraftStatus.COMPLETED,
          currentPick: undefined
        } : null);
        break;
        
      default:
        // For unknown events, refetch state
        fetchDraftData();
    }
  }

  const handleMakePick = (playerId: string) => {
    sendMessage({
      type: 'MAKE_PICK',
      playerId,
    });
  };

  if (!draftState) {
    return (
      <View style={styles.container}>
        <View style={styles.loadingContainer}>
          <Text style={styles.loadingText}>Loading draft...</Text>
          <Text style={styles.debugText}>Draft ID: {DRAFT_ID}</Text>
          <Text style={styles.debugText}>User ID: {USER_ID}</Text>
          <Text style={styles.debugText}>Check console for API errors...</Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <DraftTimer 
          currentPick={draftState.currentPick}
          status={draftState.status}
        />
      </View>
      <View style={styles.content}>
        <LeftSidebar
          availablePlayers={draftState.availablePlayers || []}
          teams={draftState.teams || []}
          onMakePick={handleMakePick}
          currentPick={draftState.currentPick}
        />
        <View style={styles.gridContainer}>
          <DraftGrid
            teams={draftState.teams || []}
            picks={draftState.picks || []}
            currentPick={draftState.currentPick}
            settings={draftState.settings || create(DraftSettingsSchema, {
              rounds: draftState.metadata.total_rounds,
              timePerPickSec: 120,
              draftOrder: [],
              thirdRoundReversal: false,
            })}
            draftType={draftState.metadata.draft_type}
            playersById={draftState.playersById || new Map()}
          />
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f1419',
  },
  header: {
    height: 60,
    borderBottomWidth: 1,
    borderBottomColor: '#2d3748',
  },
  content: {
    flex: 1,
    flexDirection: 'row',
  },
  gridContainer: {
    flex: 1,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
  loadingText: {
    color: '#ffffff',
    fontSize: 18,
    fontWeight: 'bold',
    marginBottom: 10,
  },
  debugText: {
    color: '#a0aec0',
    fontSize: 12,
    marginBottom: 5,
  },
});