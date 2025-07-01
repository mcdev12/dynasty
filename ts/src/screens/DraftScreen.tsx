import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import DraftGrid from '../components/DraftGrid';
import DraftTimer from '../components/DraftTimer';
import LeftSidebar from '../components/LeftSidebar';
import { DraftUIState, DraftEvent, DraftStatus, CurrentPickState } from '../types/draft';
import { create } from '@bufbuild/protobuf';
import { DraftService, GetDraftRequestSchema } from '../genproto/draft/v1/draft_service_pb';
import { DraftSettingsSchema } from '../genproto/draft/v1/draft_pb';
import { FantasyTeamService, GetFantasyTeamsByLeagueRequestSchema } from '../genproto/fantasyteam/v1/service_pb';
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

export default function DraftScreen() {
  const [draftState, setDraftState] = useState<DraftUIState | null>(null);
  const { connected, sendMessage } = useDraftWebSocket(DRAFT_ID, USER_ID, handleDraftEvent);

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
        picks: [], // Will be populated by GetDraftPicks
        availablePlayers: [], // Will be populated by GetAvailablePlayers
      };
      
      console.log('Mapped draft state:', mappedState);
      setDraftState(mappedState);
    } catch (error) {
      console.error('Failed to fetch draft data:', error);
    }
  };


  function handleDraftEvent(event: any) {
    console.log('Draft event:', event);
    
    switch (event.type) {
      case 'PickStarted':
        if (event.data?.started_at && event.data?.time_per_pick_sec) {
          const newCurrentPick: CurrentPickState = {
            teamId: event.data.team_id || '',
            round: event.data.round || 1,
            pickNumber: event.data.pick || 1,
            overallPick: event.data.overall_pick || 1,
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
        // Refetch state to get updated picks
        fetchDraftData();
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