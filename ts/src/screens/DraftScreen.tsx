import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import DraftGrid from '../components/DraftGrid';
import DraftTimer from '../components/DraftTimer';
import LeftSidebar from '../components/LeftSidebar';
import { DraftUIState, DraftGatewayStateResponse, DraftEvent, DraftStatus, DraftType, DraftSettings } from '../types/draft';
import { create } from '@bufbuild/protobuf';
import { DraftSettingsSchema } from '../genproto/draft/v1/draft_pb';
import { useDraftWebSocket } from '../hooks/useDraftWebSocket';

const DRAFT_ID = 'e6a3e117-e1d0-49bb-8d64-fb0d9217c230'; // TODO: Get from navigation/props
const USER_ID = '667f2190-b9bc-48e2-8683-963b62ac5428'; // TODO: Get from auth/context

export default function DraftScreen() {
  const [draftState, setDraftState] = useState<DraftUIState | null>(null);
  const { connected, sendMessage } = useDraftWebSocket(DRAFT_ID, USER_ID, handleDraftEvent);

  useEffect(() => {
    fetchDraftState();
  }, []);

  const fetchDraftState = async () => {
    try {
      console.log('Fetching draft state from:', `http://localhost:8081/api/drafts/${DRAFT_ID}/state`);
      const response = await fetch(`http://localhost:8081/api/drafts/${DRAFT_ID}/state`);
      console.log('Response status:', response.status);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const rawData: DraftGatewayStateResponse = await response.json();
      console.log('Raw gateway response:', rawData);
      
      // Map gateway response to UI state
      const mappedState: DraftUIState = {
        draft_id: rawData.draft_id,
        status: mapStringToDraftStatus(rawData.status),
        metadata: {
          draft_type: mapStringToDraftType(rawData.metadata.draft_type),
          league_id: rawData.metadata.league_id,
          total_rounds: rawData.metadata.total_rounds,
          total_teams: rawData.metadata.total_teams,
        },
        // Map current pick if available
        currentPick: rawData.current_pick ? {
          teamId: rawData.current_pick.team_id,
          round: rawData.current_pick.round,
          pickNumber: rawData.current_pick.pick,
          overallPick: rawData.current_pick.overall_pick,
          startedAt: rawData.current_pick.started_at,
          timePerPickSec: rawData.current_pick.time_per_pick_sec,
        } : undefined,
        // Initialize empty arrays for now - these will be populated from other endpoints
        teams: [],
        picks: [],
        availablePlayers: [],
      };
      
      console.log('Mapped state:', mappedState);
      setDraftState(mappedState);
      console.log('Draft state currentPick:', mappedState.currentPick);
    } catch (error) {
      console.error('Failed to fetch draft state:', error);
    }
  };

  // Helper functions to map string values to enums
  const mapStringToDraftStatus = (status: string): DraftStatus => {
    switch (status) {
      case 'NOT_STARTED': return DraftStatus.NOT_STARTED;
      case 'IN_PROGRESS': return DraftStatus.IN_PROGRESS;
      case 'PAUSED': return DraftStatus.PAUSED;
      case 'COMPLETED': return DraftStatus.COMPLETED;
      case 'CANCELLED': return DraftStatus.CANCELLED;
      default: return DraftStatus.UNSPECIFIED;
    }
  };

  const mapStringToDraftType = (type: string): DraftType => {
    switch (type) {
      case 'SNAKE': return DraftType.SNAKE;
      case 'AUCTION': return DraftType.AUCTION;
      case 'ROOKIE': return DraftType.ROOKIE;
      default: return DraftType.UNSPECIFIED;
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
        fetchDraftState();
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
        fetchDraftState();
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
            settings={create(DraftSettingsSchema, {
              rounds: draftState.metadata.total_rounds,
              timePerPickSec: 120, // Default - will need to get from actual settings
              draftOrder: [], // Will need to populate
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